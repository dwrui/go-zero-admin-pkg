package excel

import "testing"

func TestSuggestIsSensitive(t *testing.T) {
	if !SuggestIsSensitive("mobile", "手机号") {
		t.Fatal("mobile should be sensitive")
	}
	if SuggestIsSensitive("supplier_code", "编码") {
		t.Fatal("code should not be sensitive")
	}
}

func TestMaskExportValue(t *testing.T) {
	if got := MaskExportValue("13812345678", MaskModePartial); got != "138****5678" {
		t.Fatalf("partial mobile got %v", got)
	}
	if got := MaskExportValue("secret", MaskModeFull); got != "****" {
		t.Fatalf("full got %v", got)
	}
	if got := MaskExportValue("plain", MaskModeNone); got != "plain" {
		t.Fatalf("none got %v", got)
	}
}
