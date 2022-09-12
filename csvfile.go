package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// processCSVHeader identify the field position and update in the fieldMap, return the total of found fields
func processCSVHeader(header []string, fieldMap map[string]*Field) int {
	count := 0
	// remove the \ufeff from the header[0] if it exist
	if len(header) > 0 {
		header[0] = strings.Replace(header[0], "\ufeff", "", -1)
	}

	infof(7, "csv header list: %s", header)

	for id, iter := range header {
		field := fieldMap[iter]
		if field != nil {
			switch field.Type {
			case "subfile":
				field.inputPosArray = append(field.inputPosArray, id)
			default:
				field.inputPos = id
				infof(6, "Position in csv file: %d for field [%s]", field.inputPos, field.InputName)
			}
			count++
		}
	}
	return count
}

// processCSVRecord takes the record content as the input, and generates the output slice based on the fieldSlice definition
func processCSVRecord(record []string, fieldSlice []*Field, fieldsMap map[string]*Field) ([]string, error) {
	result := make([]string, 0)
	len := len(record)
	for _, value := range fieldSlice {
		switch value.converterType {
		case ConverterTypeSubfile:
			subFile := config.subfilesMap[value.Params[0].(string)]
			if subFile == nil {
				return nil, fmt.Errorf("cannot find subfile: %s in config file", value.Params[0])
			}
			//process the fields in the subFile
			for _, field := range subFile.fieldSlice {
				masterField := fieldsMap[field.InputName]
				if masterField != nil {
					field.value = record[masterField.inputPos]
				} else {
					field.value = ""
				}
			}
			for _, id := range value.inputPosArray {
				// add the content to the array
				subRecord := make([]string, 0)
				var err error
				var content string
				for _, field := range subFile.fieldSlice {
					if field.InputName != "value" {
						subRecord = append(subRecord, field.value)
					} else {
						content, err = processSubFields(subFile.Name, record[id], field.OutputName)
						if err == nil {
							subRecord = append(subRecord, content)
						}
					}
				}
				// save the new record into the subFile master list
				if err == nil {
					subFile.records = append(subFile.records, subRecord)
				}
			}
		default:
			if value.inputPos == -1 {
				result = append(result, "")
			} else if value.inputPos < len {
				// find the value at the corresponding field position
				content := record[value.inputPos]
				result = append(result, content)
			} else {
				return nil, fmt.Errorf("field (%s)'s position: %d is bigger than the actual field size: %d", value.InputName, value.inputPos, len)
			}
		}
	}
	return result, nil
}

func openInput(path string) (f *os.File, err error) {
	if path == "" {
		return os.Stdin, nil
	}

	f, err = os.Open(path)
	return
}

func ProcessCSVFile(config CSVConvertorConfig) error {
	inf, err := openInput(config.Input)
	if err != nil {
		fatalf("unable to open input file: %s", err)
	}
	defer inf.Close()

	r := csv.NewReader(inf)

	xlsFile := createExcelFile(config.fieldSlice, config.Output, config.SheetName)

	defer xlsFile.XLSX.Close()

	header := true

	recordCount := 0
	saveCount := 0

	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to read record: %v", err)
		}

		if header {
			count := processCSVHeader(record, config.fieldsMap)
			if count == 0 {
				return fmt.Errorf("unable to found matched header fields")
			}
			header = false

			continue
		}

		result, err := processCSVRecord(record, config.fieldSlice, config.fieldsMap)
		if err != nil {
			return fmt.Errorf("unable to convertFields: %v", err)
		}

		recordCount++
		if recordCount%1000 == 0 {
			infof(3, "processed csv records: %d", recordCount)
		}

		if res := saveRecordInExcelFile(xlsFile, config.SheetName, result, config.fieldSlice, false); res {
			saveCount++
		}
	}

	infof(3, "process main output file: %s, total records processed: %d, total records saved: %d",
		config.Output, recordCount, saveCount)

	if err := xlsFile.XLSX.SaveAs(config.Output); err != nil {
		fatalf("save xlsx file (%s) failed, err: %s", config.Output, err)
		return err
	}

	return nil
}

func processSubFiles(subFile *SubFile) error {
	xlsSubFile := createExcelFile(subFile.fieldSlice, subFile.Output, subFile.SheetName)
	defer xlsSubFile.XLSX.Close()

	for id, record := range subFile.records {
		// process the subFile
		saveRecordInExcelFile(xlsSubFile, subFile.SheetName, record, subFile.fieldSlice, true)
		infof(10, "process subFile[%s], %d, record [%s]", subFile.Output, id, record)
	}

	if err := xlsSubFile.XLSX.SaveAs(subFile.Output); err != nil {
		fatalf("save xlsx file (%s) failed, err: %s", subFile.Output, err)
		return err
	}

	infof(3, "process subFile: %s, sheet: %s, total records saved: %d", subFile.Output, subFile.SheetName, len(subFile.records))
	return nil
}

// this is only to process the subField, for jira only
func processSubFields(subFileName, record, outputName string) (string, error) {
	if record == "" {
		return "", errors.New("empty record value")
	}

	if subFileName != "JiraLogTime" {
		return record, nil
	}

	// the Jira "Log Work" field format: ";date;user id;total seconds"
	fields := strings.Split(record, ";")
	length := len(fields)

	if length < 4 {
		return "", errors.New("invalid record format, must have 4 sections")
	}

	switch outputName {
	case "Hours":
		return fields[length-1], nil
	case "Reporter":
		return fields[length-2], nil
	case "Date":
		return fields[length-3], nil
	}

	return "", errors.New("unsupported output name")
}
