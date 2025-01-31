package core;

import (
	"fmt"
	"strings"
)

func WriteListCsv(builder *strings.Builder, propsArray []map[string]interface{}, columnsList []string, useHeaders bool) {
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
	for i := 0; i < len(propsArray); i++ {
		isFirstCol = true
		for _, c := range(columnsList) {
			if !isFirstCol {
				builder.WriteString("\t")
			}
			if value, ok := propsArray[i][c]; ok {
				if valueStr, ok := value.(string); ok {
					builder.WriteString(valueStr)
				} else {
					builder.WriteString(fmt.Sprintf("%v", value))
				}
			} else {
				builder.WriteString("-")
			}
			isFirstCol = false
		}
		builder.WriteString("\n")
	}
}

func WriteInspectCsv(builder *strings.Builder, propsArray []map[string]interface{}, columnsList []string, useHeaders bool) {
	for _, c := range(columnsList) {
		if useHeaders {
			builder.WriteString(c)
			builder.WriteString("\t")
		}
		for j := 0; j < len(propsArray); j++ {
			if j > 0 {
				builder.WriteString("\t")
			}
			if value, ok := propsArray[j][c]; ok {
				if valueStr, ok := value.(string); ok {
					builder.WriteString(valueStr)
				} else {
					builder.WriteString(fmt.Sprintf("%v", value))
				}
			} else {
				builder.WriteString("-")
			}
		}
		builder.WriteString("\n")
	}
}

func WriteJson(builder *strings.Builder, propsArray []map[string]interface{}) {
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

func WriteInspectTable(builder *strings.Builder, propsArray []map[string]interface{}, columnsList []string, useHeaders bool) {
	headerInc := 0
	if useHeaders {
		headerInc = 1
	}

	nRows := len(columnsList)
	nCols := headerInc + len(propsArray)
	nStrings := nRows * nCols
	allStrings := make([]string, nStrings, nStrings)

	for i := 0; i < len(columnsList); i++ {
		if useHeaders {
			allStrings[i * nCols] = columnsList[i]
		}
		for j := 0; j < len(propsArray); j++ {
			var str string
			if value, ok := propsArray[j][columnsList[i]]; ok {
				if valueStr, ok := value.(string); ok {
					str = valueStr
				} else {
					str = fmt.Sprintf("%v", value)
				}
			}
			allStrings[i * nCols + (headerInc + j)] = str
		}
	}

	writeTable(builder, allStrings, len(columnsList), headerInc + len(propsArray), false)
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

	isFirstCol := true
	for i := 0; i < nRows; i++ {
		isFirstCol = true
		for j := 0; j < nCols; j++ {
			idx := i * nCols + j
			sp := columnWidths[j] - len(allStrings[idx])

			if !isFirstCol {
				builder.WriteString("|")
			}
			builder.WriteString(" ")
			if useHeaders && i == 0 {
				builder.Write(bufSpaces[0:sp/2])
				builder.WriteString(allStrings[idx])
				builder.Write(bufSpaces[0:sp/2+(sp%2)])
			} else {
				builder.WriteString(allStrings[idx])
				builder.Write(bufSpaces[0:sp])
			}
			builder.WriteString(" ")

			isFirstCol = false
		}
		if useHeaders && i == 0 {
			builder.WriteString("\n")

			isFirstCol = true
			for i := 0; i < nCols; i++ {
				if !isFirstCol {
					builder.WriteString("+")
				}
				builder.Write(bufHyphens[0:columnWidths[i]+2])
				isFirstCol = false
			}
		}
		builder.WriteString("\n")
	}
}
