package logsanitize

import (
	"encoding/json"
	"strings"
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

// SanitizeJSONString 对 JSON 字符串中的敏感字段打码；非 JSON 则原样截断。
func SanitizeJSONString(raw string, maxBytes int) string {
	if raw == "" {
		return ""
	}
	if maxBytes > 0 && len(raw) > maxBytes {
		raw = raw[:maxBytes] + "...(truncated)"
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
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
	return string(out)
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
		return s[:maxBytes] + "...(truncated)"
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
