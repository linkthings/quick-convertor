package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const defaultHeadingRow = 1
const defaultHeadingStyle = `{"font": {"bold": true}, "alignment":{"horizontal":"center","ident":1,"justify_last_line":true,"reading_order":0,"relative_indent":1,"shrink_to_fit":true,"vertical":"middle","wrap_text":true}}`

// File represents a single-sheet xlsx file
type File struct {
	SheetName string
	Columns   []column
	NextRow   int
	XLSX      *excelize.File
}

type column struct {
	Ref         string
	Style       string
	HeadingCell string
	Heading     string
	Width       int
	FuncCell    bool
}

func IsFile(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// NewExcel returns a pointer to an excel.File with all columns initialised to defaults
func NewExcel(fileName string, sheetName string, colNames []string, width []int, funcCell []bool) *File {
	f := new(File)
	f.SheetName = sheetName
	f.NextRow = defaultHeadingRow
	var err error

	if IsFile(fileName) {
		f.XLSX, err = excelize.OpenFile(fileName)
		if err != nil {
			errorf("Output file already exist, but return error when open: %s", err)
			os.Exit(0)
		}
		infof(2, "excelize.OpenFile open file: %s for sheet [%s]", fileName, sheetName)
		// for existing file, make sure to delete the old sheet first
		sheetID := f.XLSX.GetSheetIndex(sheetName)
		if sheetID == 0 {
			infof(2, "sheet [%s] doesn't exist in the current file, create a new sheet", sheetName)
		} else {
			infof(2, "sheet [%s] exist in the current file %s, removed", sheetName, fileName)
			f.XLSX.DeleteSheet(sheetName)
			if err := f.XLSX.SaveAs(config.Output); err != nil {
				errorf("save spreadsheet first after deleting the old sheet, err: %s", err)
			}
		}
	} else {
		infof(2, "Output file doesn't exist, creating a new file at %s", fileName)
		//for new file, then create new sheet
		f.XLSX = excelize.NewFile()
		// delete the default sheet
		f.XLSX.DeleteSheet("Sheet1")
		if err := f.XLSX.SaveAs(config.Output); err != nil {
			errorf("save spreadsheet first by deleting the default sheet: sheet1, err: %s", err)
		}
	}

	sheetID := f.XLSX.NewSheet(sheetName)
	f.XLSX.SetActiveSheet(sheetID)

	xc := columnRefs(len(colNames))
	for i := range colNames {
		c := column{
			Ref:         xc[i],
			HeadingCell: xc[i] + strconv.Itoa(f.NextRow), // "A1", "A2" etc
			Heading:     colNames[i],
			Width:       width[i],
			FuncCell:    funcCell[i],
		}
		f.Columns = append(f.Columns, c)
		f.XLSX.SetCellValue(f.SheetName, c.HeadingCell, c.Heading)
		f.XLSX.SetColWidth(f.SheetName, c.Ref, c.Ref, float64(c.Width))
	}

	f.SetHeadingStyle(defaultHeadingStyle)

	return f
}

// SetHeadingStyle sets the column heading style
func (f *File) SetHeadingStyle(style string) {
	startCell := f.Columns[0].HeadingCell
	endCell := f.Columns[len(f.Columns)-1].HeadingCell

	st, _ := f.XLSX.NewStyle(style)
	f.XLSX.SetCellStyle(f.SheetName, startCell, endCell, st)
}

// AddRow adds a row of data to the sheet
func (f *File) AddRow(sheetName string, data []interface{}) error {

	if len(data) != len(f.Columns) {
		return fmt.Errorf("number of data items (%d) does not equal the number of columns (%d)", len(data), len(f.Columns))
	}

	f.NextRow++
	for i, c := range f.Columns {
		cell := c.Ref + strconv.Itoa(f.NextRow) // eg "A1", "A2"... "AA26"
		if c.FuncCell {
			// replace the row name with actual row value
			value := strings.Replace(data[i].(string), "{row}", strconv.Itoa(f.NextRow), -1)
			f.XLSX.SetCellFormula(sheetName, cell, value)
		} else {
			f.XLSX.SetCellValue(sheetName, cell, data[i])
		}
	}

	return nil
}

// columnRefs generates the specified number of column references - eg "A", "B" ... "Z", "AA", "AB" etc.
func columnRefs(numCols int) []string {

	result := []string{}
	xa := []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
		"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	}

	for i := 0; i < numCols; i++ {

		var colName string
		var colPrefix string

		set := int(math.Floor(float64(i) / float64(26)))
		if set > 0 {
			colPrefix = xa[set-1]
		}
		colName = colPrefix + xa[i-(set*26)]
		result = append(result, colName)
	}

	return result
}
