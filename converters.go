package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Converter func(itemData *[]interface{}, input string, field *Field) (result *string)
type ConverterType int

const (
	ConverterTypeDefault ConverterType = iota
	ConverterTypeSec2day
	ConverterTypeSec2hour
	ConverterTypeFloat
	ConverterTypeInt
	ConverterTypeTime2date
	ConverterTypeSubfile
	ConverterTypeFunc
	ConverterTypeLookup
	ConverterTypeConstantString
)

var converterError string = "invalid parameter in config file"

func (ft ConverterType) String() string {
	return []string{"default", "sec2day", "sec2hour", "float", "int", "time2date", "subfile", "func", "lookup", "constant"}[ft]
}

// FieldTypeConvert convert the string values from the configuration file into internal enum and function values
func FieldTypeConvert(input string) (ft ConverterType, ct Converter) {
	ft = ConverterTypeDefault
	ct = converterDefault

	switch strings.ToLower(input) {
	case "sec2day":
		ft = ConverterTypeSec2day
		ct = converterSec2Day
	case "sec2hour":
		ft = ConverterTypeSec2hour
		ct = converterSec2Hour
	case "float":
		ft = ConverterTypeFloat
		ct = converterFloat
	case "int":
		ft = ConverterTypeInt
		ct = converterInt
	case "time2date":
		ft = ConverterTypeTime2date
		ct = converterTime2date
	case "subfile":
		ft = ConverterTypeSubfile
		ct = converterSubfile
	case "func":
		ft = ConverterTypeFunc
		ct = converterFunc
	case "lookup":
		ft = ConverterTypeLookup
		ct = converterLookup
	case "constant":
		ft = ConverterTypeConstantString
		ct = converterConstantString
	}
	return ft, ct
}

func converterDefault(itemData *[]interface{}, input string, field *Field) (result *string) {
	*itemData = append(*itemData, input)
	return nil
}

func converterConstantString(itemData *[]interface{}, input string, field *Field) (result *string) {
	if field == nil || len(field.Params) != 1 {
		errorf("converterConstantString invalid parameter in Field, return nil")
		result = &converterError
		*itemData = append(*itemData, converterError)
		return result
	}

	*itemData = append(*itemData, field.Params[0])
	return nil
}

func converterSec2Day(itemData *[]interface{}, input string, field *Field) (result *string) {
	if intValue, err := strconv.Atoi(input); err == nil {
		days := intValue / (3600 * 24)
		hours := intValue % (3600 * 24) / 3600
		mins := intValue % (3600 * 24) % 3600 / 60
		desc := fmt.Sprintf("%02dd %02dh %02ds", days, hours, mins)
		*itemData = append(*itemData, desc)
	} else {
		*itemData = append(*itemData, input)
	}
	return nil
}

func converterSec2Hour(itemData *[]interface{}, input string, field *Field) (result *string) {
	if floatValue, err := strconv.ParseFloat(input, 64); err == nil {
		floatValue = floatValue / 3600
		*itemData = append(*itemData, floatValue)
	} else {
		*itemData = append(*itemData, input)
	}
	return nil
}

func converterInt(itemData *[]interface{}, input string, field *Field) (result *string) {
	if intValue, err := strconv.Atoi(input); err == nil {
		*itemData = append(*itemData, intValue)
	} else {
		*itemData = append(*itemData, input)
	}
	return nil
}

func converterFloat(itemData *[]interface{}, input string, field *Field) (result *string) {
	if floatValue, err := strconv.ParseFloat(input, 64); err == nil {
		*itemData = append(*itemData, floatValue)
	} else {
		*itemData = append(*itemData, input)
	}
	return nil
}

func converterTime2date(itemData *[]interface{}, input string, field *Field) (result *string) {
	// convert time format "27/May/21 2:11 AM" to keep date only
	fields := strings.Split(input, " ")
	*itemData = append(*itemData, fields[0])
	return nil
}

func converterSubfile(itemData *[]interface{}, input string, field *Field) (result *string) {
	// no action need in this function for subfile type
	return nil
}

func converterFunc(itemData *[]interface{}, input string, field *Field) (result *string) {
	// the excel func content are all saved in Params[0]
	*itemData = append(*itemData, field.Params[0])
	return nil
}

func converterLookup(itemData *[]interface{}, input string, field *Field) (result *string) {
	var resLookup string

	if field == nil || len(field.Params) != 3 {
		errorf("converterLookup invalid parameter in Field, return nil")
		result = &converterError
		*itemData = append(*itemData, converterError)
		return result
	}

	srcIndex := field.Params[0].(int)
	mapLookup := field.Params[1].(*Lookup)
	dstIndex := field.Params[2].(int)

	var resultList []string

	lenItemData := len(*itemData)

	// the input is empty in this case
	infof(10, "converterLookup for field [%s], itemData Size: %d with reference field index: %d", field.OutputName, lenItemData, srcIndex)

	if lenItemData > 0 && srcIndex < lenItemData {
		// cross check the slice size
		switch mapLookup.lookupOption {
		case LookupOptionDefault:
			resultList = mapLookup.keyValueMap[(*itemData)[srcIndex].(string)]
		case LookupOptionSubstring:
			resultList = lookupViaSubstring((*itemData)[srcIndex].(string), mapLookup.keyValueSlice)
		case LookupOptionRegexp:
			resultList = lookupViaRegexp((*itemData)[srcIndex].(string), mapLookup.keyValueSlice)
		}
	} else {
		resultList = nil
	}

	if resultList != nil {
		// the result list contains the mapping list,
		// find the result based on the dstIndex value
		resultLength := len(resultList)
		if dstIndex < resultLength {
			resLookup = resultList[dstIndex]
		}
	}

	if resLookup == "" && mapLookup.Default != "" {
		// set the default value of the result
		resLookup = mapLookup.Default
	}

	if field.OutputName != "" {
		//save the result into output file if it is requested in the config file
		mapLookup.converter(itemData, resLookup, field)
	}
	// save the result back to record array in case it is referenced by other fields
	result = &resLookup
	return result
}
