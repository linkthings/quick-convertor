package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
)

type CSVConvertorConfig struct {
	Input     string     `config:"input"`
	Output    string     `config:"output"`
	SheetName string     `config:"sheetName"`
	Fields    []string   `config:"fields"`
	Subfiles  []*SubFile `config:"subfile"`
	Lookups   []*Lookup  `config:"lookup"`
	Filters   []*Filter  `config:"filter"`
	// below attributes to keep the converted result
	fieldSlice  []*Field          // define the output fields setting based on the Fields configs
	fieldsMap   map[string]*Field // memory map for quick references to fieldSlice,using input csv's field name as key
	subfilesMap map[string]*SubFile
	lookupMap   map[string]*Lookup
}

type SubFile struct {
	Name      string   `config:"name"`
	SheetName string   `config:"sheetName"`
	Output    string   `config:"output"`
	Fields    []string `config:"fields"`
	// below attributes to keep the converted result
	fieldsMap  map[string]*Field
	fieldSlice []*Field
	records    [][]string
}

type Lookup struct {
	Name      string `config:"name"`
	FileName  string `config:"fileName"`
	SheetName string `config:"sheetName"`
	Type      string `config:"type"`
	Option    string `config:"option"`
	Default   string `config:"default"`
	// below attributes to keep the mapping content
	lookupOption  LookupOption
	keyValueMap   map[string][]string
	keyValueSlice []*LookupRegex // for substring and regex
	converterType ConverterType
	converter     Converter
	err           error
}

type Filter struct {
	Field  string   `config:"field"`
	Values []string `config:"values"`

	// processed result
	fieldPos int
	valueMap map[string]int
}

type Field struct {
	// the field name in the input csv file
	InputName string

	// the field name in the output csv file
	OutputName string

	// the cell width in the output xlsx file
	Width int

	// the cell type in the xlsx, if it is int, float, then will be converted into int, float instead of string
	Type string

	// the remaining parameters for the field
	Params []interface{}

	// the position of the field in the input csv file, this is dynamic calculated when reading the input file
	inputPos      int    // the position of the field in the input CSV file; default is -1
	inputPosArray []int  // the position array of the fields (with same name) in the input CSV file
	value         string // the actual field value in the csv file, save for temp use
	converterType ConverterType
	converter     Converter
}

var config = CSVConvertorConfig{}

func ReadConfig(path string) {

	configRaw, err := yaml.NewConfigWithFile(path, ucfg.PathSep("."))
	if err != nil {
		errorf("ReadConfig NewConfigWithFile return: %s", err)
		os.Exit(1)
	}
	err = configRaw.Unpack(&config)
	if err != nil {
		errorf("ReadConfig Unpack return: %s", err)
		os.Exit(1)
	}

	config.subfilesMap = make(map[string]*SubFile)
	config.lookupMap = make(map[string]*Lookup)
	// read the mapping file and store the mapping in memory
	for _, iter := range config.Lookups {
		// update the LookupOption
		iter.lookupOption = LookupOptionConvert(iter.Option)
		iter.converterType, iter.converter = FieldTypeConvert(iter.Type)
		iter.keyValueMap = make(map[string][]string)
		iter.keyValueSlice = make([]*LookupRegex, 0)

		infof(6, "Lookup[%s] with option: %s", iter.Name, iter.lookupOption.String())
		infof(6, "loadExcelFile processing lookup[%s] in file: %s, sheet: %s", iter.Name, iter.FileName, iter.SheetName)
		//save the load result, keep the return error string into the iter.err thus the error can be output to the resulting file
		iter.err = loadLookupExcelFile(iter)
		config.lookupMap[iter.Name] = iter
	}

	// processing the subFiles
	for _, iter := range config.Subfiles {
		iter.fieldsMap, iter.fieldSlice = formalizeFieldConfigs(iter.Fields)
		// update the field location id directly
		for id, field := range iter.fieldSlice {
			if field.InputName != "" {
				field.inputPos = id
			}
		}
		//add the subFile name to map
		config.subfilesMap[iter.Name] = iter
		iter.records = make([][]string, 0)
		infof(6, "ReadConfig add subFile [%s]", iter.Name)
	}

	config.fieldsMap, config.fieldSlice = formalizeFieldConfigs(config.Fields)

	// processing the filters
	for _, iter := range config.Filters {
		iter.fieldPos = getOutputFieldPos(iter.Field, config.fieldSlice)
		if iter.fieldPos >= 0 {
			iter.valueMap = make(map[string]int)
			for _, field := range iter.Values {
				iter.valueMap[field] = 1
			}
		} else {
			errorf("filter field [%s] is not defined in the field list", iter.Field)
		}
	}

	infof(3, "ReadConfig load %s successfully", path)
}

func formalizeFieldConfigs(csvFields []string) (map[string]*Field, []*Field) {
	fieldsMap := make(map[string]*Field)
	fieldSlice := make([]*Field, 0)

	for _, iter := range csvFields {
		fields := strings.Split(iter, ",")
		if len(fields) < 2 {
			errorf("incorrect field format, the %s must have at least 2 parameters separated by `,`", iter)
			continue
		}
		field := new(Field)
		field.inputPos = -1
		field.inputPosArray = make([]int, 0)
		field.InputName = strings.TrimSpace(fields[0])
		field.OutputName = strings.TrimSpace(fields[1])
		field.Width = 20
		//pass the default converter
		field.converterType, field.converter = FieldTypeConvert("")
		if field.InputName != "" {
			fieldsMap[field.InputName] = field
		}

		size := len(fields)

		if size >= 3 {
			field.Width, _ = strconv.Atoi(strings.TrimSpace(fields[2]))
		}
		if size >= 4 {
			field.Type = strings.TrimSpace(fields[3])
			field.converterType, field.converter = FieldTypeConvert(field.Type)
		}

		var err error

		if size >= 5 {
			field.Params = make([]interface{}, 0)
			switch field.converterType {
			case ConverterTypeSubfile:
				field.Params = append(field.Params, strings.TrimSpace(fields[4]))
			case ConverterTypeFunc:
				// need to pull all the remaining fields together
				field.Params = append(field.Params, strings.Join(fields[4:], ","))
			case ConverterTypeLookup:
				if size != 7 {
					//for errors, change it to be a default converter and export the constant string in the output
					errorf("Processing field [%s]: invalid parameter size for ConverterTypeLookup, should have: src index, map name, result index", field.OutputName)
					err = errors.New("invalid parameter size for " + field.OutputName)
				} else {
					// get the index value based on the reference field's output name
					var mapIndex int
					if mapIndex, err = strconv.Atoi(strings.TrimSpace(fields[6])); err == nil {
						mapIndex -= 2 // convert the src index to map index (need to reduce 2 as the map key is not included in the result list)
						if mapIndex < 0 {
							err = errors.New("incorrect map index value " + strings.TrimSpace(fields[5]))
						}
					}
					refIndex := getOutputFieldPos(strings.TrimSpace(fields[4]), fieldSlice)
					lookup := config.lookupMap[strings.TrimSpace(fields[5])]

					if err == nil {
						if lookup != nil {
							err = lookup.err
						} else {
							// can't find stored lookup map
							err = errors.New("undefined lookup map " + strings.TrimSpace(fields[5]))
						}
					}
					if err == nil && refIndex == -1 {
						err = errors.New("the field not defined yet" + fields[4])
					}
					if err == nil {
						field.Params = append(field.Params, refIndex)
						field.Params = append(field.Params, lookup)
						field.Params = append(field.Params, mapIndex)
					}
				}
			}
		}

		if err != nil {
			// for any error, keep adding the field, but convey the error into the resulting file
			errorf("Processing field: %s return error: %s", field.OutputName, err)
			field.converterType = ConverterTypeConstantString
			field.converter = converterConstantString
			field.Params = append(field.Params, err.Error())
		}

		fieldSlice = append(fieldSlice, field)
	}
	return fieldsMap, fieldSlice
}

// this function is used for the lookup func as it need to find the position for a output field
func getOutputFieldPos(fieldName string, fieldSlice []*Field) int {
	for id, iter := range fieldSlice {
		if iter.OutputName == fieldName {
			return id
		}
	}
	return -1
}
