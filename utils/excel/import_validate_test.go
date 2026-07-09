package excel

import "testing"

func TestValidateImportRowsWithRulesChinaMobile(t *testing.T) {
	rules := []FieldValidateRule{{
		DbField:       "mobile",
		Label:         "手机号",
		Required:      1,
		ValidateRules: `{"tags":["chinaMobile"]}`,
	}}
	rows := []map[string]interface{}{
		{"mobile": "13800138000"},
		{"mobile": "12345"},
	}
	nums := []int{2, 3}
	valid, validNums, errs := ValidateImportRowsWithRules(rows, nums, rules)
	if len(valid) != 1 || len(validNums) != 1 || validNums[0] != 2 {
		t.Fatalf("valid=%d nums=%v", len(valid), validNums)
	}
	if len(errs) != 1 || errs[0].Row != 3 {
		t.Fatalf("errs=%+v", errs)
	}
}

func TestCheckDuplicateKeyInFile(t *testing.T) {
	mapping := &ImportMapping{DuplicateKey: []string{"code"}}
	raw := []map[string]string{
		{"code": "A001"},
		{"code": "A001"},
		{"code": "A002"},
	}
	nums := []int{2, 3, 4}
	errs := CheckDuplicateKeyInFile(mapping, raw, nums)
	if len(errs) != 2 {
		t.Fatalf("expected 2 dup errors, got %d", len(errs))
	}
}
