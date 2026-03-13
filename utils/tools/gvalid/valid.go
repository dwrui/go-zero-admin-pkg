package gvalid

import (
	"fmt"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gerror"
	"github.com/go-playground/validator/v10"
	"reflect"
	"regexp"
	"strings"
)

// MessageValidator 支持自定义错误消息的验证器
type MessageValidator struct {
	validator  *validator.Validate
	messages   map[string]interface{} // 自定义错误消息
	fieldNames map[string]string      // 字段中文名称
}

// NewMessageValidator 创建新的消息验证器
func NewMessageValidator() *MessageValidator {
	v := validator.New()

	// 设置标签名函数
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		// 优先使用json标签
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				return jsonTag[:idx]
			}
			return jsonTag
		}
		return field.Name
	})

	// 注册自定义验证规则
	registerCustomRules(v)

	return &MessageValidator{
		validator:  v,
		messages:   make(map[string]interface{}),
		fieldNames: make(map[string]string),
	}
}

// SetMessages 设置自定义错误消息
//
//	messages 格式：map[string]interface{}{
//	    "fieldName": "通用错误消息|{tag:消息}",
//	    "fieldName": map[string]string{
//	        "required": "必填错误消息",
//	        "min": "最小值错误消息",
//	    },
//	}
func (mv *MessageValidator) SetMessages(messages map[string]interface{}) *MessageValidator {
	mv.messages = messages
	return mv
}

// SetFieldNames 设置字段中文名称
func (mv *MessageValidator) SetFieldNames(names map[string]string) *MessageValidator {
	mv.fieldNames = names
	return mv
}
func (mv *MessageValidator) Validate(data interface{}) error {
	if err := mv.validator.Struct(data); err != nil {
		return mv.convertValidationError(err.(validator.ValidationErrors), data)
	}
	return nil
}

// ValidateOne 验证结构体，只返回第一个错误
func (mv *MessageValidator) ValidateOne(data interface{}) error {
	if err := mv.validator.Struct(data); err != nil {
		return mv.convertFirstValidationError(err.(validator.ValidationErrors), data)
	}
	return nil
}

// ValidateAll 验证结构体，返回所有错误信息
func (mv *MessageValidator) ValidateAll(data interface{}) []error {
	if err := mv.validator.Struct(data); err != nil {
		return mv.convertAllValidationErrors(err.(validator.ValidationErrors), data)
	}
	return nil
}

// ValidateMap 验证结构体，返回字段名到错误信息的映射
func (mv *MessageValidator) ValidateMap(data interface{}) map[string]string {
	if err := mv.validator.Struct(data); err != nil {
		return mv.convertValidationErrorMap(err.(validator.ValidationErrors), data)
	}
	return make(map[string]string)
}

// ValidateVar 验证单个变量
func (mv *MessageValidator) ValidateVar(field interface{}, tag string, fieldName string) error {
	if err := mv.validator.Var(field, tag); err != nil {
		return mv.convertSingleError(err.(validator.ValidationErrors), fieldName, tag)
	}
	return nil
}

// 转换验证错误为自定义消息
func (mv *MessageValidator) convertValidationError(errs validator.ValidationErrors, data interface{}) error {
	var errorMessages []string

	for _, err := range errs {
		fieldName := err.Field()
		tag := err.Tag()
		param := err.Param()

		// 获取自定义错误消息
		customMsg := mv.getCustomMessage(fieldName, tag, param)
		if customMsg != "" {
			errorMessages = append(errorMessages, customMsg)
			continue
		}

		// 使用默认转换
		defaultMsg := mv.convertDefaultError(err)
		errorMessages = append(errorMessages, defaultMsg)
	}

	if len(errorMessages) > 0 {
		return gerror.New(strings.Join(errorMessages, "；"))
	}

	return nil
}

// convertFirstValidationError 转换第一个验证错误
func (mv *MessageValidator) convertFirstValidationError(errs validator.ValidationErrors, data interface{}) error {
	if len(errs) == 0 {
		return nil
	}

	// 只处理第一个错误
	err := errs[0]
	fieldName := err.Field()
	tag := err.Tag()
	param := err.Param()

	// 获取自定义错误消息
	customMsg := mv.getCustomMessage(fieldName, tag, param)
	if customMsg != "" {
		return gerror.New(customMsg)
	}

	// 使用默认转换
	return gerror.New(mv.convertDefaultError(err))
}

// convertAllValidationErrors 转换所有验证错误为错误数组
func (mv *MessageValidator) convertAllValidationErrors(errs validator.ValidationErrors, data interface{}) []error {
	var errors []error

	for _, err := range errs {
		fieldName := err.Field()
		tag := err.Tag()
		param := err.Param()

		// 获取自定义错误消息
		customMsg := mv.getCustomMessage(fieldName, tag, param)
		if customMsg != "" {
			errors = append(errors, gerror.New(customMsg))
			continue
		}

		// 使用默认转换
		defaultMsg := mv.convertDefaultError(err)
		errors = append(errors, gerror.New(defaultMsg))
	}

	return errors
}

// convertValidationErrorMap 转换验证错误为字段名到错误信息的映射
func (mv *MessageValidator) convertValidationErrorMap(errs validator.ValidationErrors, data interface{}) map[string]string {
	errorMap := make(map[string]string)

	for _, err := range errs {
		fieldName := err.Field()
		tag := err.Tag()
		param := err.Param()

		// 获取自定义错误消息
		customMsg := mv.getCustomMessage(fieldName, tag, param)
		if customMsg != "" {
			errorMap[fieldName] = customMsg
			continue
		}

		// 使用默认转换
		defaultMsg := mv.convertDefaultError(err)
		errorMap[fieldName] = defaultMsg
	}

	return errorMap
}

// 转换单个错误
func (mv *MessageValidator) convertSingleError(errs validator.ValidationErrors, fieldName, tag string) error {
	for _, err := range errs {
		customMsg := mv.getCustomMessage(fieldName, tag, err.Param())
		if customMsg != "" {
			return gerror.New(customMsg)
		}

		return gerror.New(mv.convertDefaultError(err))
	}
	return nil
}

// 获取自定义错误消息
func (mv *MessageValidator) getCustomMessage(fieldName, tag, param string) string {
	// 检查是否有字段的特定消息
	if fieldMsgs, ok := mv.messages[fieldName]; ok {
		switch msg := fieldMsgs.(type) {
		case string:
			// 处理格式如："账号不能为空|min:账号长度应当在{min}到{max}之间|max:账号长度超过最大限制"
			return mv.parseMessageTemplate(msg, fieldName, tag, param)
		case map[string]string:
			// 处理格式如：map[string]string{"required": "账号不能为空", "min": "账号长度应当在{min}到{max}之间"}
			if customMsg, exists := msg[tag]; exists {
				return mv.replaceParams(customMsg, param)
			}
		}
	}

	// 检查是否有通用消息（使用字段中文名）
	chineseFieldName := mv.getFieldName(fieldName)
	if chineseFieldName != fieldName {
		return mv.getCustomMessage(chineseFieldName, tag, param)
	}

	return ""
}

// 解析消息模板
func (mv *MessageValidator) parseMessageTemplate(template, fieldName, tag, param string) string {
	// 分割不同的验证规则消息
	parts := strings.Split(template, "|")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 检查是否包含标签指定
		if strings.Contains(part, ":") {
			tagParts := strings.SplitN(part, ":", 2)
			if len(tagParts) == 2 && tagParts[0] == tag {
				return mv.replaceParams(tagParts[1], param)
			}
		} else if tag == "required" {
			// 默认消息（通常是required）
			return mv.replaceParams(part, param)
		}
	}

	return ""
}

// 替换参数
func (mv *MessageValidator) replaceParams(msg, param string) string {
	// 替换{param}占位符
	msg = strings.ReplaceAll(msg, "{param}", param)

	// 替换其他可能的占位符
	if strings.Contains(msg, "{") && strings.Contains(msg, "}") {
		// 使用正则表达式替换所有{xxx}格式的占位符
		re := regexp.MustCompile(`\{([^}]+)\}`)
		msg = re.ReplaceAllStringFunc(msg, func(match string) string {
			placeholder := strings.Trim(match, "{}")
			switch placeholder {
			case "min", "max", "len", "eq", "ne", "gt", "gte", "lt", "lte":
				return param
			default:
				return match // 保持原样
			}
		})
	}

	return msg
}

// 转换默认错误
func (mv *MessageValidator) convertDefaultError(err validator.FieldError) string {
	fieldName := err.Field()
	tag := err.Tag()
	param := err.Param()
	chineseFieldName := mv.getFieldName(fieldName)

	// 默认中文错误消息模板
	templates := map[string]string{
		"required":    "%s不能为空",
		"email":       "%s格式不正确",
		"min":         "%s长度不能小于%s",
		"max":         "%s长度不能超过%s",
		"len":         "%s长度必须为%s",
		"numeric":     "%s必须为数字",
		"number":      "%s必须为数字",
		"alpha":       "%s只能包含字母",
		"alphanum":    "%s只能包含字母和数字",
		"contains":    "%s必须包含%s",
		"excludes":    "%s不能包含%s",
		"startswith":  "%s必须以%s开头",
		"endswith":    "%s必须以%s结尾",
		"url":         "%s必须是有效的URL",
		"uri":         "%s必须是有效的URI",
		"ip":          "%s必须是有效的IP地址",
		"ipv4":        "%s必须是有效的IPv4地址",
		"ipv6":        "%s必须是有效的IPv6地址",
		"uuid":        "%s必须是有效的UUID",
		"phone":       "%s格式不正确",
		"e164":        "%s必须是有效的E164电话号码",
		"base64":      "%s必须是有效的Base64编码",
		"hexadecimal": "%s必须是有效的十六进制",
		"json":        "%s必须是有效的JSON格式",
		"jwt":         "%s必须是有效的JWT格式",
		"oneof":       "%s必须是%s中的一个",
	}

	if template, exists := templates[tag]; exists {
		return fmt.Sprintf(template, chineseFieldName, param)
	}

	// 自定义规则
	switch tag {
	case "username":
		return fmt.Sprintf("%s只能包含字母、数字和下划线，长度为3-20位", chineseFieldName)
	case "password":
		return fmt.Sprintf("%s必须包含字母和数字，长度为6-20位", chineseFieldName)
	case "chinaMobile":
		return fmt.Sprintf("%s必须是有效的中国手机号", chineseFieldName)
	case "chineseName":
		return fmt.Sprintf("%s必须是2-10个汉字", chineseFieldName)
	case "captchaCode":
		return fmt.Sprintf("%s必须是4位数字", chineseFieldName)
	case "idCard":
		return fmt.Sprintf("%s格式不正确", chineseFieldName)
	default:
		return fmt.Sprintf("%s验证失败：%s", chineseFieldName, tag)
	}
}

// 获取字段中文名称
func (mv *MessageValidator) getFieldName(fieldName string) string {
	if name, ok := mv.fieldNames[fieldName]; ok {
		return name
	}

	// 默认中文映射
	defaultNames := map[string]string{
		"username":        "用户名",
		"password":        "密码",
		"email":           "邮箱",
		"phone":           "手机号",
		"realName":        "真实姓名",
		"codeid":          "验证码ID",
		"captcha":         "验证码",
		"captchaType":     "验证码类型",
		"nickname":        "昵称",
		"avatar":          "头像",
		"gender":          "性别",
		"status":          "状态",
		"page":            "页码",
		"pageSize":        "每页条数",
		"keyword":         "关键词",
		"type":            "类型",
		"name":            "名称",
		"title":           "标题",
		"content":         "内容",
		"description":     "描述",
		"address":         "地址",
		"birthday":        "生日",
		"age":             "年龄",
		"id":              "ID",
		"userId":          "用户ID",
		"roleId":          "角色ID",
		"deptId":          "部门ID",
		"postId":          "岗位ID",
		"menuId":          "菜单ID",
		"dictId":          "字典ID",
		"configId":        "配置ID",
		"noticeId":        "公告ID",
		"loginName":       "登录名",
		"loginIp":         "登录IP",
		"loginDate":       "登录时间",
		"createBy":        "创建者",
		"createTime":      "创建时间",
		"updateBy":        "更新者",
		"updateTime":      "更新时间",
		"remark":          "备注",
		"params":          "参数",
		"oldPassword":     "旧密码",
		"newPassword":     "新密码",
		"confirmPassword": "确认密码",
	}

	if name, ok := defaultNames[fieldName]; ok {
		return name
	}
	return fieldName
}

// 注册自定义验证规则
func registerCustomRules(v *validator.Validate) {
	// 中文姓名验证
	v.RegisterValidation("chineseName", func(fl validator.FieldLevel) bool {
		return regexp.MustCompile(`^[\u4e00-\u9fa5]{2,10}$`).MatchString(fl.Field().String())
	})

	// 身份证号验证
	v.RegisterValidation("idCard", func(fl validator.FieldLevel) bool {
		return regexp.MustCompile(`^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`).MatchString(fl.Field().String())
	})

	// 手机号验证（中国）
	v.RegisterValidation("chinaMobile", func(fl validator.FieldLevel) bool {
		return regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(fl.Field().String())
	})

	// 密码强度验证（必须包含字母和数字）
	v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
		hasDigit := regexp.MustCompile(`\d`).MatchString(password)
		return hasLetter && hasDigit && len(password) >= 6
	})

	// 用户名验证（字母、数字、下划线）
	v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`).MatchString(fl.Field().String())
	})

	// 验证码验证（4位数字）
	v.RegisterValidation("captchaCode", func(fl validator.FieldLevel) bool {
		return regexp.MustCompile(`^\d{4}$`).MatchString(fl.Field().String())
	})
}
