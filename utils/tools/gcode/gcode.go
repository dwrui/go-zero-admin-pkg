// Package gcode 提供了通用的错误码定义和常见错误码的实现。
package gcode

// 代码是通用错误代码接口定义。
type Code interface {
	// Code 返回当前错误码的整数编号。
	Code() int

	// Message 返回当前错误码的简要消息。
	Message() string

	// Detail 返回当前错误码的详细信息，
	// 主要设计为错误码的扩展字段。
	Detail() interface{}
}

// ================================================================================================================
// CommonErrorCode 是通用错误码的定义。
// 框架保留了内部错误码：code < 1000。
// ================================================================================================================

var (
	CodeNil                       = localCode{-1, "", nil}                             // No error code specified.
	CodeOK                        = localCode{0, "OK", nil}                            // It is OK.
	CodeInternalError             = localCode{50, "Internal Error", nil}               // An error occurred internally.
	CodeValidationFailed          = localCode{51, "Validation Failed", nil}            // Data validation failed.
	CodeDbOperationError          = localCode{52, "Database Operation Error", nil}     // Database operation error.
	CodeInvalidParameter          = localCode{53, "Invalid Parameter", nil}            // The given parameter for current operation is invalid.
	CodeMissingParameter          = localCode{54, "Missing Parameter", nil}            // Parameter for current operation is missing.
	CodeInvalidOperation          = localCode{55, "Invalid Operation", nil}            // The function cannot be used like this.
	CodeInvalidConfiguration      = localCode{56, "Invalid Configuration", nil}        // The configuration is invalid for current operation.
	CodeMissingConfiguration      = localCode{57, "Missing Configuration", nil}        // The configuration is missing for current operation.
	CodeNotImplemented            = localCode{58, "Not Implemented", nil}              // The operation is not implemented yet.
	CodeNotSupported              = localCode{59, "Not Supported", nil}                // The operation is not supported yet.
	CodeOperationFailed           = localCode{60, "Operation Failed", nil}             // I tried, but I cannot give you what you want.
	CodeNotAuthorized             = localCode{61, "Not Authorized", nil}               // Not Authorized.
	CodeSecurityReason            = localCode{62, "Security Reason", nil}              // Security Reason.
	CodeServerBusy                = localCode{63, "Server Is Busy", nil}               // Server is busy, please try again later.
	CodeUnknown                   = localCode{64, "Unknown Error", nil}                // Unknown error.
	CodeNotFound                  = localCode{65, "Not Found", nil}                    // Resource does not exist.
	CodeInvalidRequest            = localCode{66, "Invalid Request", nil}              // Invalid request.
	CodeNecessaryPackageNotImport = localCode{67, "Necessary Package Not Import", nil} // It needs necessary package import.
	CodeInternalPanic             = localCode{68, "Internal Panic", nil}               // A panic occurred internally.
	CodeBusinessValidationFailed  = localCode{300, "Business Validation Failed", nil}  // Business validation failed.
)

// New 创建并返回一个错误码。
// 注意，它返回的是 Code 接口对象。
func New(code int, message string, detail interface{}) Code {
	return localCode{
		code:    code,
		message: message,
		detail:  detail,
	}
}

// WithCode 创建并返回一个基于给定 Code 的新错误码。
// 代码和消息来自给定的 `code`，但详细信息来自给定的 `detail`。
func WithCode(code Code, detail interface{}) Code {
	return localCode{
		code:    code.Code(),
		message: code.Message(),
		detail:  detail,
	}
}
