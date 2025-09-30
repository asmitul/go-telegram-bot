package errors

// 错误码定义
const (
	// CodeNotFound 资源不存在
	CodeNotFound = "NOT_FOUND"

	// CodeValidation 验证错误
	CodeValidation = "VALIDATION_ERROR"

	// CodePermission 权限错误
	CodePermission = "PERMISSION_DENIED"

	// CodeInternal 内部错误
	CodeInternal = "INTERNAL_ERROR"

	// CodeExternal 外部服务错误
	CodeExternal = "EXTERNAL_ERROR"

	// CodeConflict 冲突错误（如重复创建）
	CodeConflict = "CONFLICT"

	// CodeRateLimit 限流错误
	CodeRateLimit = "RATE_LIMIT_EXCEEDED"

	// CodeTimeout 超时错误
	CodeTimeout = "TIMEOUT"

	// CodeUnknown 未知错误
	CodeUnknown = "UNKNOWN"
)