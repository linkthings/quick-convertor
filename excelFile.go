package main

import (
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

// loadLookupExcelFile initializes and loads the mapping from configured file for the lookup function
func loadLookupExcelFile(lookup *Lookup) error {
	var err error

	xlFile, err := excelize.OpenFile(lookup.FileName)
	if err != nil {
		errorf("loadExcelFile return err: %s when processing file %s", err, lookup.FileName)
		return err
	}

	rows, err := xlFile.GetRows(lookup.SheetName)
	if err != nil {
		errorf("GetRows return err: %s when processing sheet %s", err, lookup.SheetName)
		return err
	}

	// skip the first row as it is for header
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) > 1 {
			key := rows[i][0]
			content := make([]string, len(rows[i])-1)
			copy(content, rows[i][1:])

			if lookup.lookupOption != LookupOptionDefault {
				lookupRegex := new(LookupRegex)
				if lookup.lookupOption == LookupOptionSubstring {
					// change to lower case for the key as Substring will always match with caseinsensitive
					lookupRegex.Key = strings.ToLower(key)
				} else {
					lookupRegex.Key = key
				}

				lookupRegex.Value = content
				if lookup.lookupOption == LookupOptionRegexp {
					lookupRegex.Regexp = regexp.MustCompile(key)
				}
				lookup.keyValueSlice = append(lookup.keyValueSlice, lookupRegex)
			} else {
				existing := lookup.keyValueMap[key]
				// if need to display warning
				if *warning && existing != nil {
					log.Warn("lookup file (", lookup.FileName, ") sheet (", lookup.SheetName, ") has duplicated key ", key, " with value ",
						existing, ", will be overrided by new value ", content)
				}
				lookup.keyValueMap[key] = content
			}
		}
	}

	return nil
}

func createExcelFile(fieldSlice []*Field, fileName string, sheetName string) (xlsFile *File) {
	header := []string{}
	width := []int{}
	funcCelll := []bool{}

	for _, iter := range fieldSlice {
		if iter.OutputName != "" {
			// if the output name is not defined, then the field won't be output
			width = append(width, iter.Width)
			header = append(header, iter.OutputName)
			if iter.converterType == ConverterTypeFunc {
				funcCelll = append(funcCelll, true)
			} else {
				funcCelll = append(funcCelll, false)
			}
		}
	}

	xlFile := NewExcel(fileName, sheetName, header, width, funcCelll)

	return xlFile
}

// the fieldSlice is the description of each field in the record list
// return true if the save result is successful.
func saveRecordInExcelFile(xlsFile *File, sheetName string, record []string, fieldSlice []*Field, isSubfile bool) bool {
	itemData := make([]interface{}, 0)
	var res *string
	for id, iter := range record {
		field := fieldSlice[id]
		if field.OutputName != "" && field.converter != nil {
			res = field.converter(&itemData, iter, field)
			if res != nil {
				record[id] = *res
			}
		}
	}

	// the filter function is not applicable to subfile data
	if isSubfile || filterRecord(itemData, config.fieldSlice, config.Filters) {
		err := xlsFile.AddRow(sheetName, itemData)
		if err != nil {
			errorf("AddRow return error: %s for items: %s", err, itemData)
			return false
		}
		return true
	}
	return false
}
