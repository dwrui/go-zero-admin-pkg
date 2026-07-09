package excel

import (
	"fmt"
	"strings"
)

const (
	MaskModeFull    = "full"
	MaskModePartial = "partial"
	MaskModeNone    = "none"
)

// SuggestIsSensitive 按字段名/中文名推断是否敏感字段
func SuggestIsSensitive(field, name string) bool {
	f := strings.ToLower(strings.TrimSpace(field))
	n := strings.TrimSpace(name)
	if f == "" && n == "" {
		return false
	}
	keys := []string{
		"mobile", "phone", "tel",
		"id_card", "idcard", "identity",
		"bank", "bank_account", "bankcard",
		"real_name", "realname",
		"address", "home_address", "detail_address",
		"email", "id_no", "tax_no",
	}
	for _, k := range keys {
		if strings.Contains(f, k) {
			return true
		}
	}
	nameKeys := []string{"手机", "电话", "身份证", "银行卡", "姓名", "住址", "地址", "邮箱", "税号"}
	for _, k := range nameKeys {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// DefaultMaskMode 敏感字段默认脱敏方式
func DefaultMaskMode(field string) string {
	f := strings.ToLower(field)
	switch {
	case strings.Contains(f, "mobile") || strings.Contains(f, "phone") || f == "tel":
		return MaskModePartial
	case strings.Contains(f, "id_card") || strings.Contains(f, "idcard"):
		return MaskModePartial
	case strings.Contains(f, "bank"):
		return MaskModePartial
	case strings.Contains(f, "email"):
		return MaskModePartial
	default:
		return MaskModeFull
	}
}

// MaskExportValue 导出脱敏
func MaskExportValue(value interface{}, maskMode string) interface{} {
	if maskMode == "" || maskMode == MaskModeNone {
		return value
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", value))
	if s == "" || s == "<nil>" {
		return ""
	}
	switch maskMode {
	case MaskModeFull:
		return "****"
	case MaskModePartial:
		return maskPartial(s)
	default:
		return "****"
	}
}

func maskPartial(s string) string {
	runes := []rune(s)
	n := len(runes)
	if n <= 1 {
		return "*"
	}
	if n == 2 {
		return string(runes[0]) + "*"
	}
	if n <= 4 {
		return string(runes[0]) + strings.Repeat("*", n-2) + string(runes[n-1])
	}
	if n == 11 && isDigits(s) {
		return string(runes[:3]) + "****" + string(runes[7:])
	}
	if n >= 15 && isDigitsOrX(s) {
		return string(runes[:3]) + strings.Repeat("*", n-7) + string(runes[n-4:])
	}
	showHead := 2
	showTail := 2
	if n > 8 {
		showHead = 3
		showTail = 4
	}
	mid := n - showHead - showTail
	if mid < 1 {
		return string(runes[0]) + "****"
	}
	return string(runes[:showHead]) + strings.Repeat("*", mid) + string(runes[n-showTail:])
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func isDigitsOrX(s string) bool {
	for i, r := range s {
		if r >= '0' && r <= '9' {
			continue
		}
		if (r == 'x' || r == 'X') && i == len(s)-1 {
			continue
		}
		return false
	}
	return s != ""
}

// NormalizeMaskMode 规范化脱敏模式
func NormalizeMaskMode(mode string, sensitive bool) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if !sensitive {
		return MaskModeNone
	}
	switch mode {
	case MaskModeFull, MaskModePartial, MaskModeNone:
		return mode
	default:
		return MaskModeFull
	}
}

// EffectiveMaskMode 计算列最终脱敏模式
func EffectiveMaskMode(sensitive bool, maskMode, dbField string) string {
	if !sensitive {
		return MaskModeNone
	}
	mode := strings.ToLower(strings.TrimSpace(maskMode))
	if mode == "" {
		return DefaultMaskMode(dbField)
	}
	return NormalizeMaskMode(mode, true)
}

// MaskModeForColumnConfig 供 ColumnConfig 使用
func MaskModeForColumnConfig(sensitive bool, maskMode, field string) string {
	return EffectiveMaskMode(sensitive, maskMode, field)
}
