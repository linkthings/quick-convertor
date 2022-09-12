# Readme

This tool extracts data from CSV file, filters out irrelevant rows, transforms the columns based on configured rules, and saves results into Excel file.

## Key Features

The tool provides simple and easy to understand transformation rules to convert CSV files into Microsoft Excel file.

More specifically, it can process CSV files with dynamic or repetitive fields, either discard those repetitive fields, or save them into separated spreadsheet file(s) that convert from columar values to row-based records.

One typical example is to process the CSV data exported from Atlassian Jira software. When you exported the issues data from Jira via "Export Excel CSV (all fields)" option, you may find many repetitive "Log Work" fields which are to track all the time reported to each Jira issue. The total number of "Log Work" fields varies in each export based on the Jira issues included in that export. The example configuration file at "example/config-jira.yaml" provides one example to convert the "Log Work" fields into a separate file, which you can find at the fields config "Log Work,,0,subfile,JiraLogTime".

The tool currently supports the below transformation features:

1. sec2day: convert a value in seconds into days, e.g. "129600" becomes "1.5" when saved into resulting file, since 1.5 = 129600/(3600*24).
2. sec2hour: convert a value in seconds into hours, e.g. "3600" becomes "1" when saved into spreadsheet.
3. float: convert a string value in CSV file into a float value in the Excel cell.
4. int: convert a string value in CSV file into a int value in the Excel cell.
5. time2date: convert a time value in CSV file into date string, e.g. "27/May/21 2:11 AM" becames "27/May/21" in the spreadsheet cell.
6. <a id="subfile-syntax" />subfile: save the fields (specifically for repetitive fields) into separate spreadsheet file, to transpose from column to row.
   - Syntax: `subfile, subfile_definitions`
7. func: include an Excel function in the specific field.
   - Syntax: `func, excel_functions`
   - Note: use keyword "{row}" to request the converter to replace with actual row number, use "\"" if need to include " in the function. e.g.
     - `LEFT(C{row},3)`: when in row 2, the actual cell value is "=LEFT(C2,3)"; when in row 3, the value becomes "=LEFT(C3,3)"
     - `IF(D{row}=\"\", \"\", TEXT(D{row},\"yyyy-mm\"))`: this function is the take the "YYYY-MM" value from column D and save into current column
8. <a id="lookup-syntax" />lookup: lookup and replace the current field value from a dictionary table defined in the specified spreadsheet file.
   - Syntax: `lookup, referenced field name, dictionary definition, number`
   - Note: the `referenced field name` is the output name of the referenced field; the reference field must be defined prior to the current field.
   - Example:  `lookup,Endpoint,Module,2` is to lookup the value of field "Endpoint" in the dictionary "Module", and write the values in the 2nd column of the matching row into the resulting file.

Please refer to the below functions defined in the `converters.go` and add more converters if needed.

```go
type Converter func(itemData *[]interface{}, input string, field *Field) (result *string)
type ConverterType int
func (ft ConverterType) String() string 
func FieldTypeConvert(input string) (ft ConverterType, ct Converter) 
```

## Configuration File

The configuration file is defined in yaml format, and can be divided into 5 portions.

- Basic info
- Field definitions
- Filter settings
- Subfile settings
- Lookup settings

### Basic info

This portion defines the input, output file name, as well as the sheet name in the output spreadsheet.

```yaml
input: 'data/data.csv'
output: 'data/data-gen.xlsx'
sheetName: 'rawdata'
```

### Field definitions

This portion defines the list of relevant fields, either to be written to the result file, or to be referenced and used to derive new value.

The field definition includes a list of strings, separated by comma, in the below format:
  
 `Input field name, Output field name, Cell Width, Transformation Type, Transform Parameters 1, ...`

The `Input field name`, `Output field name` are mandatory attributes, others can be skipped if not needed.

Below are the detail meaning of the parameters:

- Input field name: The original field name in the input CSV file. This will be used to find the field in the CSV file, thus need to be exactly the same value in the CSV file, spaces are allowed inside the name.
  - Note: if the field is to be derived from prior field and doesn't exist in the CSV file, this param can be kept empty.

- Output field name: The column name to be generated in the Excel file. The value will be outputed in the header line of the resulting file.
  - Note: the field won't be output if this parameter is empty. This is useful to define a reference column in the CSV file but don't need it in the resulting file.

- Cell Width: optional, the width of column in the Excel file, default value is 20

- Transformation Type: optional, the transformation type defined in the [Key Features](#key-features) section. If not defined, then it means no transformation is needed.

- Transform Parameters 1, ...: optional, the list of parameters required by the transformation type. As each transformation require different parameters, please refer to the transformation types defined in "Key Features" section.

```yaml
fields: 
    - "Endpoint,Endpoint,20"
    - "Application,Application,20"
    - ",Endpoint Type,10,lookup,Endpoint,HostType,2"
    - "Log Work,-,0,subfile,JiraLogTime"
```

### Filter settings

The filter settings define the filter to be used against the input or derived fields. If the input records does not include matched values, the record will be discarded and not generated in the resulting file. The example config below will only save the Application whose values are either AppName 1 or AppName 2 in the result file.

```yaml
filter: 
    - field: "Application"
      values: 
        - "AppName 1"
        - "AppName 2"
```

### Subfile settings

The subfile setting is to define how the repetitive fields can be converted into rows in a separated spreadsheet. The attributes include:

1. name: the subfile section name, to be used as the parameter in the transformation type 'subfile', see [subfile syntax](subfile-syntax).
2. sheetName: defines the sheet name in the resulting file.
3. output: defines the resulting file path.
4. fields: defines the fields to be written in the resulting file, the definition follows the same syntax defined in the aforementioned [Field definitions](#field-definitions) section.

As example, if you have a CSV file with header fields like `key, name, field1, field1, field1, field2, field2`, then you can use the below config to save the file into 3 different files:

1. add below 2 fields in the fields settings:

```yaml
- "field1,-,0,subfile,subfile1"
- "field2,-,0,subfile,subfile2"
```

2. add below 2 records in the subfile section, in which the example-data2-subfile1.xlsx contains the fields of Key, Name and field1 values, while example-data2-subfile2.xlsx only contains Key and field2 value:

```yaml
subfile: 
    - name: 'subfile1'
      sheetName: 'Field1'
      output: 'example/example-data2-subfile1.xlsx' # the output file name 
      fields: 
            # the fields to be outputed in the resulting file, the syntax is same
            - "key,Key,12"
            - "name,Name,12"
            - "value,Value,12"
            # "value" in the first parameter is the keyword specifically used in the subfile setting, used to refer to the value of the subfile field, in this case, it is the vaule of the field1 of each column.
    - name: 'subfile2'
      sheetName: 'Field2'
      output: 'example/example-data2-subfile2.xlsx'
      fields: 
            - "key,Key,12"
            - "value,Value,12"
```

Note: The below subfile definition is specially supported for Jira "Log Time" field, in which "JiraLogTime", "Reporter", "Date", "Hours" are keywords supported to separate "Log Time" values into multiple fields for easier processing.

```yaml
fields: 
    - "Log Work,-,0,subfile,JiraLogTime"

subfile: 
    - name: 'JiraLogTime'
      sheetName: 'Time Spent'
      output: 'file1-timespent.xlsx'
      fields: 
            - "Project key,Project Key,12"
            - "Issue key,Issue key,12"
            - "value,Reporter,12"
            - "value,Date,12"
            - "value,Hours,12,sec2hour"
```

### Lookup settings

The Lookup setting is to define search external dictionary file and output the corresponding column's content into resulting file when matched. This setting is used to support the transformation type [lookup](#lookup-syntax)

The attributes in Lookup settings include:

1. name: the Lookup section name, to be used as the 2nd parameter in the transformation type 'lookup', see [lookup](#lookup-syntax).
2. sheetName: defines the sheet name in the dictionary file. Note: the dictionary file can include multiple sheets.
3. filenName: defines the dictionary file path.
4. default: the default value if not matching is found
5. option: defines the matching algorithm when doing the lookup, currently supported value include:
   1. substring: matched if the `referenced field name` include substring defined in the sheet. in this option, the matching is case insensitive.
   2. regexp: matched if the `referenced field name` match the regexp pattern defined in the sheet, e.g. you can define `^\w{2,12}t\-` in the spreadsheet.
   3. default: full string matching, case sensitive

Example:

```yaml
fields: 
    - "Vulnerability Title,VulTitle,50"
    - ",Category,20,lookup,VulTitle,VulnCategory,2"

lookup:
    - name: 'VulnCategory'
      sheetName: 'Vuln Category'
      fileName: 'example/example-dict-1.xlsx'
      option: 'substring'
      default: 'default category'
```

## Commands

Below commands are used to compile and run the tool

``` bash
#go to the directory of source code, to compile and install the tool
go build
go install

#convert the CSV file based on the configuration file
qc -c config.yaml
#if the config.yaml is under the current path
qc 
```
