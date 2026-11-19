package logsanitize

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

var sensitiveKeys = map[string]struct{}{
	"password":         {},
	"old_password":     {},
	"new_password":     {},
	"passwd":           {},
	"token":            {},
	"access_token":     {},
	"refresh_token":    {},
	"authorization":    {},
	"captcha":          {},
	"secret":           {},
	"api_key":          {},
	"apikey":           {},
	"private_key":      {},
	"id_card":          {},
	"bank_card":        {},
	"mobile":           {},
	"phone":            {},
}

var sensitiveHeaderKeys = map[string]struct{}{
	"authorization":   {},
	"x-csrf-token":    {},
	"x-refresh-token": {},
	"cookie":          {},
	"accesstoken":     {},
}

// Options 日志脱敏选项。
type Options struct {
	MaxBodyBytes int
	SkipBody     bool
}

// SanitizeJSONString 对 JSON 字符串中的敏感字段打码；JSON 超限时直接记录空对象 "{}"，不记录数据。
func SanitizeJSONString(raw string, maxBytes int) string {
	if raw == "" {
		return ""
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
		// 非 JSON：纯文本截断（调用方本就不会 JSON.parse）
		if maxBytes > 0 && len(raw) > maxBytes {
			return cutUTF8(raw, maxBytes) + "...(truncated)"
		}
		return raw
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	maskValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return raw
	}
	if maxBytes > 0 && len(out) > maxBytes {
		return "{}"
	}
	return string(out)
}

// cutUTF8 按 rune 边界截断，避免切在多字节字符中间产生乱码
func cutUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	cut := s[:maxBytes]
	// 回退到完整 rune 边界
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

// SanitizeHeadersJSON 脱敏 HTTP Header JSON。
func SanitizeHeadersJSON(raw string, maxBytes int) string {
	if raw == "" {
		return ""
	}
	var headers map[string][]string
	if err := json.Unmarshal([]byte(raw), &headers); err != nil {
		return SanitizeJSONString(raw, maxBytes)
	}
	for k, vals := range headers {
		if _, ok := sensitiveHeaderKeys[strings.ToLower(k)]; ok {
			for i := range vals {
				vals[i] = "***"
			}
			headers[k] = vals
		}
	}
	b, err := json.Marshal(headers)
	if err != nil {
		return raw
	}
	s := string(b)
	if maxBytes > 0 && len(s) > maxBytes {
		return "{}"
	}
	return s
}

// ShouldSkipBody 按路径前缀判断是否跳过请求/响应体记录。
func ShouldSkipBody(path string, prefixes []string) bool {
	path = strings.ToLower(path)
	for _, p := range prefixes {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func maskValue(v any) {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			if _, ok := sensitiveKeys[strings.ToLower(k)]; ok {
				node[k] = "***"
				continue
			}
			maskValue(val)
		}
	case []any:
		for _, item := range node {
			maskValue(item)
		}
	}
}
