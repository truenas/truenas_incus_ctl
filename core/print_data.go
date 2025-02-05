package core;

import (
	"fmt"
	"os"
	"strings"
)

func PrintTableDataList(format string, jsonName string, columnsList []string, data []map[string]interface{}) {
	var table strings.Builder
	f := strings.ToLower(format)

	switch f {
	case "compact":
		WriteListCsv(&table, data, columnsList, false)
	case "csv":
		WriteListCsv(&table, data, columnsList, true)
	case "json":
		table.WriteString("{")
		WriteEncloseAndEscape(&table, jsonName, "\"")
		table.WriteString(":")
		WriteJson(&table, data)
		table.WriteString("}\n")
	case "table":
		WriteListTable(&table, data, columnsList, true)
	default:
		fmt.Fprintln(os.Stderr, "Unrecognised table format", f)
		return
	}

	os.Stdout.WriteString(table.String())
}

func WriteListCsv(builder *strings.Builder, propsArray []map[string]interface{}, columnsList []string, useHeaders bool) {
	if len(propsArray) == 0 || len(columnsList) == 0 {
		return
	}

	isFirstCol := true
	if useHeaders {
		for _, c := range(columnsList) {
			if !isFirstCol {
				builder.WriteString("\t")
			}
			builder.WriteString(c)
			isFirstCol = false
		}
		builder.WriteString("\n")
	}
	var line strings.Builder
	for i := 0; i < len(propsArray); i++ {
		line.Reset()
		isFirstCol = true
		hits := 0
		for _, c := range(columnsList) {
			if !isFirstCol {
				line.WriteString("\t")
			}
			if value, ok := propsArray[i][c]; ok {
				if valueStr, ok := value.(string); ok {
					line.WriteString(valueStr)
				} else {
					line.WriteString(fmt.Sprintf("%v", value))
				}
				hits++
			} else {
				line.WriteString("-")
			}
			isFirstCol = false
		}
		if hits > 0 {
			builder.WriteString(line.String())
			builder.WriteString("\n")
		}
	}
}

func WriteJson(builder *strings.Builder, propsArray []map[string]interface{}) {
	if len(propsArray) == 0 {
		builder.WriteString("{}")
		return
	}

	builder.WriteString("{")
	for i, p := range(propsArray) {
		if i > 0 {
			builder.WriteString(",")
		}
		name := EncloseAndEscape(p["name"].(string), "\"")
		builder.WriteString(name)
		builder.WriteString(":{\"name\":")
		builder.WriteString(name)
		for key, value := range(p) {
			if key == "name" {
				continue
			}
			builder.WriteString(",")
			WriteEncloseAndEscape(builder, key, "\"")
			builder.WriteString(":")
			if value == nil {
				builder.WriteString("null")
			} else if valueStr, ok := value.(string); ok {
				WriteEncloseAndEscape(builder, valueStr, "\"")
			} else {
				builder.WriteString(fmt.Sprintf("%v", value))
			}
		}
		builder.WriteString("}")
	}
	builder.WriteString("}")
}

func WriteListTable(builder *strings.Builder, propsArray []map[string]interface{}, columnsList []string, useHeaders bool) {
	if len(propsArray) == 0 || len(columnsList) == 0 {
		return
	}

	headerInc := 0
	if useHeaders {
		headerInc = 1
	}

	allStrings := make([]string, 0, len(columnsList) * (headerInc + len(propsArray)))
	if useHeaders {
		for i := 0; i < len(columnsList); i++ {
			allStrings = append(allStrings, columnsList[i])
		}
	}
	for i := 0; i < len(propsArray); i++ {
		for _, c := range(columnsList) {
			var str string
			if value, ok := propsArray[i][c]; ok {
				if valueStr, ok := value.(string); ok {
					str = valueStr
				} else {
					str = fmt.Sprintf("%v", value)
				}
			}
			allStrings = append(allStrings, str)
		}
	}

	writeTable(builder, allStrings, headerInc + len(propsArray), len(columnsList), useHeaders)
}

func writeTable(builder *strings.Builder, allStrings []string, nRows int, nCols int, useHeaders bool) {
	columnWidths := make([]int, nCols, nCols)
	for i := 0; i < nRows; i++ {
		for j := 0; j < nCols; j++ {
			size := len(allStrings[i*nCols+j])
			if size > columnWidths[j] {
				columnWidths[j] = size
			}
		}
	}

	widestCol := columnWidths[0]
	for i := 1; i < nCols; i++ {
		if columnWidths[i] > widestCol {
			widestCol = columnWidths[i]
		}
	}

	bufSpaces  := make([]byte, widestCol+2, widestCol+2)
	bufHyphens := make([]byte, widestCol+2, widestCol+2)
	for i := 0; i < widestCol+2; i++ {
		bufSpaces[i]  = 0x20; // space
		bufHyphens[i] = 0x2d; // -
	}

	var line strings.Builder
	for i := 0; i < nRows; i++ {
		line.Reset()
		hits := 0
		isFirstCol := true
		for j := 0; j < nCols; j++ {
			idx := i * nCols + j
			sp := columnWidths[j] - len(allStrings[idx])

			if !isFirstCol {
				line.WriteString("|")
			}
			line.WriteString(" ")
			if useHeaders && i == 0 {
				line.Write(bufSpaces[0:sp/2])
				line.WriteString(allStrings[idx])
				line.Write(bufSpaces[0:sp/2+(sp%2)])
				hits++
			} else {
				line.WriteString(allStrings[idx])
				line.Write(bufSpaces[0:sp])
				if allStrings[idx] != "" {
					hits++
				}
			}
			line.WriteString(" ")

			isFirstCol = false
		}
		if useHeaders && i == 0 {
			line.WriteString("\n")

			isFirstCol = true
			for i := 0; i < nCols; i++ {
				if !isFirstCol {
					line.WriteString("+")
				}
				line.Write(bufHyphens[0:columnWidths[i]+2])
				isFirstCol = false
			}
			// ensures the column headers and separating line are not culled
			hits++
		}
		if hits > 0 {
			builder.WriteString(line.String())
			builder.WriteString("\n")
		}
	}
}
