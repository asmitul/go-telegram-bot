package errors

// NotFound 创建资源不存在错误
func NotFound(code, message string) Error {
	if code == "" {
		code = CodeNotFound
	}
	return New(code, message)
}

// Validation 创建验证错误
func Validation(code, message string) Error {
	if code == "" {
		code = CodeValidation
	}
	return New(code, message)
}

// Permission 创建权限错误
func Permission(code, message string) Error {
	if code == "" {
		code = CodePermission
	}
	return New(code, message)
}

// Internal 创建内部错误
func Internal(code, message string) Error {
	if code == "" {
		code = CodeInternal
	}
	return New(code, message)
}

// External 创建外部服务错误
func External(code, message string) Error {
	if code == "" {
		code = CodeExternal
	}
	return New(code, message)
}

// Conflict 创建冲突错误
func Conflict(code, message string) Error {
	if code == "" {
		code = CodeConflict
	}
	return New(code, message)
}

// RateLimit 创建限流错误
func RateLimit(message string) Error {
	return New(CodeRateLimit, message)
}

// Timeout 创建超时错误
func Timeout(message string) Error {
	return New(CodeTimeout, message)
}

// IsNotFound 检查是否为 NotFound 错误
func IsNotFound(err error) bool {
	return HasCode(err, CodeNotFound)
}

// IsValidation 检查是否为验证错误
func IsValidation(err error) bool {
	return HasCode(err, CodeValidation)
}

// IsPermission 检查是否为权限错误
func IsPermission(err error) bool {
	return HasCode(err, CodePermission)
}

// IsInternal 检查是否为内部错误
func IsInternal(err error) bool {
	return HasCode(err, CodeInternal)
}

// IsExternal 检查是否为外部服务错误
func IsExternal(err error) bool {
	return HasCode(err, CodeExternal)
}

// IsConflict 检查是否为冲突错误
func IsConflict(err error) bool {
	return HasCode(err, CodeConflict)
}

// IsRateLimit 检查是否为限流错误
func IsRateLimit(err error) bool {
	return HasCode(err, CodeRateLimit)
}

// IsTimeout 检查是否为超时错误
func IsTimeout(err error) bool {
	return HasCode(err, CodeTimeout)
}