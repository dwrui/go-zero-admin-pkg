package excel

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gstr"
	"github.com/xuri/excelize/v2"
)

type FieldConfig struct {
	Field         string
	NameField     string
	OptionValue   string
	Datatable     string
	Datatablename string
	Formtype      string
}

type ColumnConfig struct {
	Title string
	Field string
}

type GetTableFieldValFunc func(tableName, fieldName string, id int64) string

func GetFieldValue(v reflect.Value, fieldName string) interface{} {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	fv := v.FieldByName(fieldName)
	if !fv.IsValid() {
		return ""
	}
	switch fv.Type().String() {
	case "sql.NullTime":
		if fv.FieldByName("Valid").Bool() {
			return fv.FieldByName("Time").Interface().(time.Time).Format("2006-01-02 15:04:05")
		}
		return ""
	case "int64":
		return fv.Int()
	case "string":
		return fv.String()
	default:
		return fv.Interface()
	}
}

func ConvertFieldValue(value interface{}, optionValue string) string {
	if optionValue == "" {
		return fmt.Sprintf("%v", value)
	}
	valStr := fmt.Sprintf("%v", value)
	opts := strings.Split(optionValue, ",")
	for _, opt := range opts {
		parts := strings.Split(opt, "=")
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == valStr {
			return strings.TrimSpace(parts[1])
		}
	}
	return valStr
}

func ExportExcel(columns []ColumnConfig, fieldConfigs []FieldConfig, list interface{}, getTableFieldVal GetTableFieldValFunc) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)

	var headers []string
	var fieldNames []string
	for _, col := range columns {
		headers = append(headers, col.Title)
		fieldNames = append(fieldNames, col.Field)
	}

	if len(headers) == 0 {
		for _, cfg := range fieldConfigs {
			headers = append(headers, cfg.Field)
			fieldNames = append(fieldNames, cfg.Field)
		}
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("list must be a slice")
	}

	for rowIdx := 0; rowIdx < v.Len(); rowIdx++ {
		item := v.Index(rowIdx)
		var rowData []interface{}
		elem := item
		if elem.Kind() == reflect.Interface {
			elem = elem.Elem()
		}
		elemVal := elem
		if elemVal.Kind() == reflect.Ptr {
			elemVal = elemVal.Elem()
		}

		for _, fieldName := range fieldNames {
			var cfg *FieldConfig
			for i := range fieldConfigs {
				if fieldConfigs[i].Field == fieldName || fieldConfigs[i].NameField == fieldName {
					cfg = &fieldConfigs[i]
					break
				}
			}

			actualFieldName := fieldName
			if cfg != nil && cfg.NameField == fieldName {
				actualFieldName = cfg.Field
			}
			camelFieldName := gstr.CaseCamel(actualFieldName)
			rawValue := GetFieldValue(elemVal, camelFieldName)

			if cfg != nil {
				if cfg.Formtype == "belongto" && cfg.Datatable != "" && cfg.Datatablename != "" && getTableFieldVal != nil {
					valInt64, ok := rawValue.(int64)
					if ok && valInt64 != 0 {
						displayValue := getTableFieldVal(cfg.Datatable, cfg.Datatablename, valInt64)
						rowData = append(rowData, displayValue)
					} else {
						rowData = append(rowData, "")
					}
				} else if cfg.OptionValue != "" {
					rowData = append(rowData, ConvertFieldValue(rawValue, cfg.OptionValue))
				} else {
					rowData = append(rowData, rawValue)
				}
			} else {
				rowData = append(rowData, rawValue)
			}
		}
		for colIdx, value := range rowData {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成Excel文件失败: %v", err)
	}

	return buf.Bytes(), nil
}
