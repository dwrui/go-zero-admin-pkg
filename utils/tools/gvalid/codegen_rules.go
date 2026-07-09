package gvalid

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gstr"
)

// FieldValidateRules 代码生成器字段验证配置（存 ga_common_generatecode_field.validate_rules）
type FieldValidateRules struct {
	Tags     []string          `json:"tags"`
	Messages map[string]string `json:"messages"`
	Regex    *RegexValidate    `json:"regex"`
}

// RegexValidate 自定义正则
type RegexValidate struct {
	Pattern string `json:"pattern"`
	Message string `json:"message"`
	Flags   string `json:"flags"`
}

// ParseFieldValidateRules 解析 validate_rules JSON
func ParseFieldValidateRules(raw string) FieldValidateRules {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return FieldValidateRules{Tags: nil, Messages: map[string]string{}}
	}
	var rules FieldValidateRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return FieldValidateRules{Tags: nil, Messages: map[string]string{}}
	}
	if rules.Messages == nil {
		rules.Messages = map[string]string{}
	}
	return rules
}

// MarshalFieldValidateRules 序列化为 JSON
func MarshalFieldValidateRules(rules FieldValidateRules) string {
	if len(rules.Tags) == 0 && rules.Regex == nil && len(rules.Messages) == 0 {
		return ""
	}
	b, err := json.Marshal(rules)
	if err != nil {
		return ""
	}
	return string(b)
}

// SuggestDefaultValidateRules 按字段名/类型推断默认规则
func SuggestDefaultValidateRules(field, name, formtype string) FieldValidateRules {
	f := strings.ToLower(field)
	n := name
	tags := make([]string, 0, 2)
	switch {
	case strings.Contains(f, "mobile") || f == "phone" || strings.Contains(n, "手机"):
		tags = append(tags, "chinaMobile")
	case strings.Contains(f, "id_card") || strings.Contains(f, "idcard") || strings.Contains(n, "身份证"):
		tags = append(tags, "idCard")
	case strings.Contains(f, "bank") || strings.Contains(n, "银行卡"):
		tags = append(tags, "bankAccount")
	case f == "email" || strings.Contains(n, "邮箱"):
		tags = append(tags, "email")
	case f == "website" || f == "url" || strings.Contains(n, "网址"):
		tags = append(tags, "url")
	case strings.Contains(f, "ip") && !strings.Contains(f, "zip"):
		tags = append(tags, "ip")
	case formtype == "date":
		tags = append(tags, "datetime=2006-01-02")
	case formtype == "datetime" || formtype == "time":
		tags = append(tags, "datetime=2006-01-02 15:04:05")
	}
	if len(tags) == 0 {
		return FieldValidateRules{}
	}
	return FieldValidateRules{Tags: tags, Messages: map[string]string{}}
}

// MergeValidateTags 合并 required 与配置 tags，生成 validator tag 列表
func MergeValidateTags(required int64, rulesRaw string) []string {
	rules := ParseFieldValidateRules(rulesRaw)
	tagSet := make(map[string]struct{})
	ordered := make([]string, 0, len(rules.Tags)+4)

	addTag := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return
		}
		if _, ok := tagSet[tag]; ok {
			return
		}
		tagSet[tag] = struct{}{}
		ordered = append(ordered, tag)
	}

	for _, t := range rules.Tags {
		addTag(t)
	}
	if required == 1 {
		addTag("required")
	}
	if rules.Regex != nil && strings.TrimSpace(rules.Regex.Pattern) != "" {
		addTag(buildRegexpTag(rules.Regex.Pattern))
	}

	if len(ordered) == 0 {
		return nil
	}
	if required != 1 && needsOmitempty(ordered) {
		out := make([]string, 0, len(ordered)+1)
		out = append(out, "omitempty")
		for _, t := range ordered {
			if t != "omitempty" {
				out = append(out, t)
			}
		}
		return out
	}
	return ordered
}

func needsOmitempty(tags []string) bool {
	for _, t := range tags {
		if t == "required" || strings.HasPrefix(t, "required_") {
			return false
		}
	}
	return true
}

func buildRegexpTag(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if strings.Contains(pattern, ",") {
		return "regexp=b64:" + base64.StdEncoding.EncodeToString([]byte(pattern))
	}
	return "regexp=" + pattern
}

// BuildValidateStructTag 生成 admin.api 用的 validate struct tag
func BuildValidateStructTag(required int64, rulesRaw string) string {
	tags := MergeValidateTags(required, rulesRaw)
	if len(tags) == 0 {
		return ""
	}
	return ` validate:"` + strings.Join(tags, ",") + `"`
}

// BuildFieldValidateMessages 生成 validate/*.go 的 messages 条目
func BuildFieldValidateMessages(jsonField, label string, required int64, rulesRaw string) map[string]string {
	rules := ParseFieldValidateRules(rulesRaw)
	tags := MergeValidateTags(required, rulesRaw)
	if len(tags) == 0 {
		return nil
	}
	msgs := make(map[string]string)
	for _, tag := range tags {
		if tag == "omitempty" {
			continue
		}
		tagKey := tagRuleKey(tag)
		if custom, ok := rules.Messages[tagKey]; ok && custom != "" {
			msgs[tagKey] = custom
			continue
		}
		if custom, ok := rules.Messages[tag]; ok && custom != "" {
			msgs[tagKey] = custom
			continue
		}
		msgs[tagKey] = defaultValidateMessage(label, tagKey, tag)
	}
	return msgs
}

func tagRuleKey(tag string) string {
	if idx := strings.Index(tag, "="); idx > 0 {
		return tag[:idx]
	}
	return tag
}

func defaultValidateMessage(label, tagKey, fullTag string) string {
	param := ""
	if idx := strings.Index(fullTag, "="); idx > 0 {
		param = fullTag[idx+1:]
	}
	templates := map[string]string{
		"required":      "请输入%s",
		"email":         "%s格式不正确",
		"url":           "%s必须是有效的URL",
		"min":           "%s长度不能小于%s",
		"max":           "%s长度不能超过%s",
		"len":           "%s长度必须为%s",
		"numeric":       "%s必须为数字",
		"number":        "%s必须为数字",
		"alpha":         "%s只能包含字母",
		"alphanum":      "%s只能包含字母和数字",
		"uuid":          "%s必须是有效的UUID",
		"ip":            "%s必须是有效的IP地址",
		"ipv4":          "%s必须是有效的IPv4地址",
		"datetime":      "%s日期时间格式不正确",
		"oneof":         "%s取值无效",
		"chinaMobile":   "%s必须是有效的中国手机号",
		"idCard":        "%s格式不正确",
		"bankAccount":   "%s格式不正确",
		"chineseName":   "%s必须是2-10个汉字",
		"username":      "%s格式不正确",
		"password":      "%s格式不正确",
		"regexp":        "%s格式不正确",
		"gtefield":      "%s不能小于关联字段",
		"ltefield":      "%s不能大于关联字段",
		"gtfield":       "%s必须大于关联字段",
		"ltfield":       "%s必须小于关联字段",
		"eqfield":       "%s与关联字段不一致",
		"nefield":       "%s不能与关联字段相同",
		"required_if":   "%s为必填项",
		"required_unless": "%s为必填项",
		"required_with": "%s为必填项",
		"required_without": "%s为必填项",
	}
	if tpl, ok := templates[tagKey]; ok {
		if strings.Count(tpl, "%s") == 2 {
			return fmt.Sprintf(tpl, label, param)
		}
		return fmt.Sprintf(tpl, label)
	}
	return fmt.Sprintf("%s验证失败", label)
}

// FormatValidateMessagesGo 格式化为 Go map 代码块
func FormatValidateMessagesGo(jsonField string, msgs map[string]string) string {
	if len(msgs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\t\t\"%s\": map[string]string{\n", jsonField))
	priority := []string{"required", "regexp", "chinaMobile", "idCard", "bankAccount", "email", "url", "min", "max", "len", "gtefield", "ltefield", "required_if"}
	written := map[string]bool{}
	write := func(k, v string) {
		if written[k] {
			return
		}
		b.WriteString(fmt.Sprintf("\t\t\t\"%s\": %q,\n", k, v))
		written[k] = true
	}
	for _, k := range priority {
		if v, ok := msgs[k]; ok {
			write(k, v)
		}
	}
	for k, v := range msgs {
		write(k, v)
	}
	b.WriteString("\t\t},\n")
	return b.String()
}

// FormRuleActionPrefix 表单控件校验提示前缀
func FormRuleActionPrefix(formtype string) string {
	switch formtype {
	case "switch", "radio", "select", "checkbox", "belongto", "belongDic", "region", "region_city", "region_area", "date", "datetime", "time", "colorpicker":
		return "请选择"
	case "image", "images", "audio", "file", "files":
		return "请上传"
	default:
		return "请填写"
	}
}

// BuildFormItemRulesAttr 生成 Vue a-form-item 的 :rules 属性
func BuildFormItemRulesAttr(jsonField, label string, required int64, rulesRaw, formtype string) string {
	prefix := FormRuleActionPrefix(formtype)
	rules := ParseFieldValidateRules(rulesRaw)
	tags := MergeValidateTags(required, rulesRaw)
	if len(tags) == 0 {
		return ""
	}

	parts := make([]string, 0, len(tags)+2)
	escLabel := escapeVueStr(label)

	for _, tag := range tags {
		tagKey := tagRuleKey(tag)
		msg := ""
		if m, ok := rules.Messages[tagKey]; ok && m != "" {
			msg = m
		} else if m, ok := rules.Messages[tag]; ok && m != "" {
			msg = m
		} else {
			msg = defaultValidateMessage(label, tagKey, tag)
		}
		msg = escapeVueStr(msg)

		switch tagKey {
		case "required":
			parts = append(parts, fmt.Sprintf("{required:true,message:'%s'}", msg))
		case "email":
			parts = append(parts, fmt.Sprintf("{type:'email',message:'%s'}", msg))
		case "chinaMobile":
			parts = append(parts, fmt.Sprintf("{match:/^1[3-9]\\d{9}$/,message:'%s'}", msg))
		case "idCard":
			parts = append(parts, fmt.Sprintf("{match:/^[1-9]\\d{5}(18|19|20)\\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\\d|3[01])\\d{3}[\\dXx]$/,message:'%s'}", msg))
		case "bankAccount":
			parts = append(parts, fmt.Sprintf("{match:/^\\d{16,19}$/,message:'%s'}", msg))
		case "chineseName":
			parts = append(parts, fmt.Sprintf("{match:/^[\\u4e00-\\u9fa5]{2,10}$/,message:'%s'}", msg))
		case "username":
			parts = append(parts, fmt.Sprintf("{match:/^[a-zA-Z0-9_]{3,20}$/,message:'%s'}", msg))
		case "url":
			parts = append(parts, fmt.Sprintf("{type:'url',message:'%s'}", msg))
		case "numeric", "number":
			parts = append(parts, fmt.Sprintf("{match:/^-?\\d+(\\.\\d+)?$/,message:'%s'}", msg))
		case "alpha":
			parts = append(parts, fmt.Sprintf("{match:/^[A-Za-z]+$/,message:'%s'}", msg))
		case "alphanum":
			parts = append(parts, fmt.Sprintf("{match:/^[A-Za-z0-9]+$/,message:'%s'}", msg))
		case "min":
			if n := tagParam(tag); n != "" {
				parts = append(parts, fmt.Sprintf("{minLength:%s,message:'%s'}", n, msg))
			}
		case "max":
			if n := tagParam(tag); n != "" {
				parts = append(parts, fmt.Sprintf("{maxLength:%s,message:'%s'}", n, msg))
			}
		case "len":
			if n := tagParam(tag); n != "" {
				parts = append(parts, fmt.Sprintf("{minLength:%s,maxLength:%s,message:'%s'}", n, n, msg))
			}
		case "regexp":
			if rules.Regex != nil && rules.Regex.Pattern != "" {
				flags := rules.Regex.Flags
				pat := escapeJSRegex(rules.Regex.Pattern)
				parts = append(parts, fmt.Sprintf("{match:new RegExp('%s','%s'),message:'%s'}", pat, escapeVueStr(flags), msg))
			}
		case "gtefield", "ltefield", "gtfield", "ltfield", "eqfield", "nefield":
			refJSON := structFieldToJSONField(tagParam(tag))
			parts = append(parts, buildCrossFieldValidator(jsonField, refJSON, tagKey, msg))
		case "required_if", "required_unless", "required_with", "required_without":
			parts = append(parts, buildConditionalValidator(tag, msg))
		default:
			_ = prefix
			_ = escLabel
		}
	}

	// 独立 regex 配置（未并入 tags 时）
	if rules.Regex != nil && strings.TrimSpace(rules.Regex.Pattern) != "" {
		hasRegexp := false
		for _, t := range tags {
			if tagRuleKey(t) == "regexp" {
				hasRegexp = true
				break
			}
		}
		if !hasRegexp {
			msg := rules.Regex.Message
			if msg == "" {
				msg = label + "格式不正确"
			}
			pat := escapeJSRegex(rules.Regex.Pattern)
			flags := escapeVueStr(rules.Regex.Flags)
			parts = append(parts, fmt.Sprintf("{match:new RegExp('%s','%s'),message:'%s'}", pat, flags, escapeVueStr(msg)))
		}
	}

	if len(parts) == 0 {
		if required == 1 {
			return fmt.Sprintf(":rules=\"[{required:true,message:'%s%s'}]\"", prefix, escLabel)
		}
		return ""
	}
	return fmt.Sprintf(":rules=\"[%s]\"", strings.Join(parts, ","))
}

func tagParam(tag string) string {
	if idx := strings.Index(tag, "="); idx > 0 {
		return tag[idx+1:]
	}
	return ""
}

func structFieldToJSONField(structField string) string {
	if structField == "" {
		return ""
	}
	// StartTime -> start_time
	return gstr.CaseSnake(structField)
}

func buildCrossFieldValidator(field, refField, op, msg string) string {
	_ = field
	js := fmt.Sprintf(`{validator:(value,cb)=>{const ref=formData.value['%s'];`, refField)
	switch op {
	case "gtefield":
		js += `if(value!=null&&value!==''&&ref!=null&&ref!==''&&value<ref){cb('` + msg + `');return}cb()}`
	case "ltefield":
		js += `if(value!=null&&value!==''&&ref!=null&&ref!==''&&value>ref){cb('` + msg + `');return}cb()}`
	case "gtfield":
		js += `if(value!=null&&value!==''&&ref!=null&&ref!==''&&!(value>ref)){cb('` + msg + `');return}cb()}`
	case "ltfield":
		js += `if(value!=null&&value!==''&&ref!=null&&ref!==''&&!(value<ref)){cb('` + msg + `');return}cb()}`
	case "eqfield":
		js += `if(value!=null&&value!==''&&ref!=null&&ref!==''&&value!==ref){cb('` + msg + `');return}cb()}`
	case "nefield":
		js += `if(value!=null&&value!==''&&ref!=null&&ref!==''&&value===ref){cb('` + msg + `');return}cb()}`
	default:
		js += `cb()}`
	}
	return js
}

func buildConditionalValidator(tag, msg string) string {
	// required_if=CustomerType 1
	param := tagParam(tag)
	pieces := strings.Fields(param)
	if len(pieces) < 1 {
		return fmt.Sprintf("{required:true,message:'%s'}", escapeVueStr(msg))
	}
	refJSON := structFieldToJSONField(pieces[0])
	expect := ""
	if len(pieces) > 1 {
		expect = pieces[1]
	}
	tagKey := tagRuleKey(tag)
	js := fmt.Sprintf(`{validator:(value,cb)=>{const fd=formData.value;const cond=String(fd['%s']??'');`, refJSON)
	switch tagKey {
	case "required_if":
		js += fmt.Sprintf(`if(cond==='%s'&&(value==null||value==='')){cb('%s');return}cb()}`, escapeVueStr(expect), escapeVueStr(msg))
	case "required_unless":
		js += fmt.Sprintf(`if(cond!=='%s'&&(value==null||value==='')){cb('%s');return}cb()}`, escapeVueStr(expect), escapeVueStr(msg))
	case "required_with":
		js += fmt.Sprintf(`if(cond!==''&&cond!=null&&(value==null||value==='')){cb('%s');return}cb()}`, escapeVueStr(msg))
	case "required_without":
		js += fmt.Sprintf(`if((cond===''||cond==null)&&(value==null||value==='')){cb('%s');return}cb()}`, escapeVueStr(msg))
	default:
		js += `cb()}`
	}
	return js
}

func escapeVueStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

func escapeJSRegex(pattern string) string {
	pattern = strings.ReplaceAll(pattern, `\`, `\\`)
	pattern = strings.ReplaceAll(pattern, `'`, `\'`)
	pattern = strings.ReplaceAll(pattern, "/", `\/`)
	return pattern
}

// RefFieldToStructName snake_case -> Struct field for gtefield etc.
func RefFieldToStructName(snakeField string) string {
	return gstr.UcFirst(gstr.CaseCamel(snakeField))
}

// BuildCrossFieldTag 构建跨字段 tag，如 gtefield=StartTime
func BuildCrossFieldTag(op, refSnakeField string) string {
	return op + "=" + RefFieldToStructName(refSnakeField)
}

// BuildRequiredIfTag 构建条件必填 tag
func BuildRequiredIfTag(refSnakeField, expectValue string) string {
	return "required_if=" + RefFieldToStructName(refSnakeField) + " " + expectValue
}

// ValidateRegexpPattern 校验正则是否可编译
func ValidateRegexpPattern(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}
