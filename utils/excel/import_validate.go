package excel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gvalid"
)

// FieldValidateRule 导入行字段 validate_rules（来自代码生成器）
type FieldValidateRule struct {
	DbField       string
	Label         string
	Required      int64
	ValidateRules string
}

// BuildFieldValidateRules 从 mapping 列与代码生成器字段构建校验规则
func BuildFieldValidateRules(mapping *ImportMapping, fields []CodegenFieldMeta) []FieldValidateRule {
	if mapping == nil {
		return nil
	}
	byField := make(map[string]CodegenFieldMeta, len(fields))
	for _, f := range fields {
		if f.Field != "" {
			byField[f.Field] = f
		}
	}
	var rules []FieldValidateRule
	for _, col := range mapping.ImportableColumns() {
		gf := byField[col.DbField]
		req := int64(0)
		if col.Required {
			req = 1
		}
		vr := strings.TrimSpace(gf.ValidateRules)
		if vr == "" && req == 0 {
			continue
		}
		label := strings.TrimSpace(col.ExcelHeader)
		if label == "" {
			label = col.DbField
		}
		rules = append(rules, FieldValidateRule{
			DbField:       col.DbField,
			Label:         label,
			Required:      req,
			ValidateRules: vr,
		})
	}
	return rules
}

// ValidateImportRowsWithRules 对已通过基础解析的行应用 validate_rules
func ValidateImportRowsWithRules(rows []map[string]interface{}, excelRowNums []int, rules []FieldValidateRule) ([]map[string]interface{}, []int, []ImportRowError) {
	if len(rules) == 0 || len(rows) == 0 {
		return rows, excelRowNums, nil
	}
	ruleByField := make(map[string]FieldValidateRule, len(rules))
	for _, r := range rules {
		ruleByField[r.DbField] = r
	}

	valid := make([]map[string]interface{}, 0, len(rows))
	validRows := make([]int, 0, len(rows))
	var allErrors []ImportRowError

	for i, row := range rows {
		excelRow := i + 2
		if i < len(excelRowNums) && excelRowNums[i] > 0 {
			excelRow = excelRowNums[i]
		}
		rowErrs := validateOneImportRowRules(row, excelRow, ruleByField)
		if len(rowErrs) > 0 {
			allErrors = append(allErrors, rowErrs...)
			continue
		}
		valid = append(valid, row)
		validRows = append(validRows, excelRow)
	}
	return valid, validRows, allErrors
}

func validateOneImportRowRules(row map[string]interface{}, excelRow int, ruleByField map[string]FieldValidateRule) []ImportRowError {
	var errs []ImportRowError
	mv := gvalid.NewMessageValidator()

	for field, rule := range ruleByField {
		tags := gvalid.MergeValidateTags(rule.Required, rule.ValidateRules)
		if len(tags) == 0 {
			continue
		}
		simple, cross := splitValidateTags(tags)
		for _, tag := range cross {
			if msg := validateCrossFieldTag(row, field, rule.Label, tag, ruleByField); msg != "" {
				errs = append(errs, ImportRowError{Row: excelRow, Field: field, Message: msg})
			}
		}
		if len(simple) == 0 {
			continue
		}
		val := stringifyImportCellValue(row[field])
		if val == "" && !containsRequiredTag(simple) {
			continue
		}
		msgs := gvalid.BuildFieldValidateMessages(field, rule.Label, rule.Required, rule.ValidateRules)
		if len(msgs) > 0 {
			msgMap := map[string]interface{}{field: msgs}
			mv.SetMessages(msgMap).SetFieldNames(map[string]string{field: rule.Label})
		}
		if err := mv.ValidateVar(val, strings.Join(simple, ","), field); err != nil {
			errs = append(errs, ImportRowError{Row: excelRow, Field: field, Message: err.Error()})
		}
	}
	return errs
}

func splitValidateTags(tags []string) (simple, cross []string) {
	for _, t := range tags {
		key := validateTagKey(t)
		switch key {
		case "gtefield", "ltefield", "gtfield", "ltfield", "eqfield", "nefield",
			"required_if", "required_unless", "required_with", "required_without":
			cross = append(cross, t)
		default:
			simple = append(simple, t)
		}
	}
	return simple, cross
}

func validateTagKey(tag string) string {
	if idx := strings.Index(tag, "="); idx > 0 {
		return tag[:idx]
	}
	return tag
}

func validateTagParam(tag string) string {
	if idx := strings.Index(tag, "="); idx > 0 {
		return strings.TrimSpace(tag[idx+1:])
	}
	return ""
}

func containsRequiredTag(tags []string) bool {
	for _, t := range tags {
		if t == "required" || strings.HasPrefix(t, "required_") {
			return true
		}
	}
	return false
}

func stringifyImportCellValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func validateCrossFieldTag(row map[string]interface{}, field, label, tag string, ruleByField map[string]FieldValidateRule) string {
	key := validateTagKey(tag)
	param := validateTagParam(tag)
	refField := structFieldToSnakeField(param)
	if refField == "" {
		return ""
	}
	left := stringifyImportCellValue(row[field])
	right := stringifyImportCellValue(row[refField])

	switch key {
	case "required_if":
		parts := strings.Fields(param)
		if len(parts) < 2 {
			return ""
		}
		refField = structFieldToSnakeField(parts[0])
		expect := parts[1]
		if stringifyImportCellValue(row[refField]) == expect && left == "" {
			return fmt.Sprintf("%s为必填项", label)
		}
	case "required_unless":
		parts := strings.Fields(param)
		if len(parts) < 2 {
			return ""
		}
		refField = structFieldToSnakeField(parts[0])
		expect := parts[1]
		if stringifyImportCellValue(row[refField]) != expect && left == "" {
			return fmt.Sprintf("%s为必填项", label)
		}
	case "required_with":
		if right != "" && left == "" {
			return fmt.Sprintf("%s为必填项", label)
		}
	case "required_without":
		if right == "" && left == "" {
			return fmt.Sprintf("%s为必填项", label)
		}
	case "gtefield", "ltefield", "gtfield", "ltfield", "eqfield", "nefield":
		if left == "" || right == "" {
			return ""
		}
		ln, lok := parseComparableNumber(left)
		rn, rok := parseComparableNumber(right)
		if lok && rok {
			switch key {
			case "gtefield":
				if ln < rn {
					return fmt.Sprintf("%s不能小于%s", label, refLabel(ruleByField, refField))
				}
			case "ltefield":
				if ln > rn {
					return fmt.Sprintf("%s不能大于%s", label, refLabel(ruleByField, refField))
				}
			case "gtfield":
				if ln <= rn {
					return fmt.Sprintf("%s必须大于%s", label, refLabel(ruleByField, refField))
				}
			case "ltfield":
				if ln >= rn {
					return fmt.Sprintf("%s必须小于%s", label, refLabel(ruleByField, refField))
				}
			case "eqfield":
				if ln != rn {
					return fmt.Sprintf("%s与%s不一致", label, refLabel(ruleByField, refField))
				}
			case "nefield":
				if ln == rn {
					return fmt.Sprintf("%s不能与%s相同", label, refLabel(ruleByField, refField))
				}
			}
			return ""
		}
		switch key {
		case "eqfield":
			if left != right {
				return fmt.Sprintf("%s与%s不一致", label, refLabel(ruleByField, refField))
			}
		case "nefield":
			if left == right {
				return fmt.Sprintf("%s不能与%s相同", label, refLabel(ruleByField, refField))
			}
		}
	}
	return ""
}

func refLabel(ruleByField map[string]FieldValidateRule, field string) string {
	if r, ok := ruleByField[field]; ok && r.Label != "" {
		return r.Label
	}
	return field
}

func structFieldToSnakeField(structField string) string {
	structField = strings.TrimSpace(structField)
	if structField == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range structField {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func parseComparableNumber(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}
