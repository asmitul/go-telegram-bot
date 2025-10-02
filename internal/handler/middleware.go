package handler

// Middleware 中间件类型
// 中间件可以在处理器执行前后执行逻辑（如日志、权限检查、错误恢复等）
type Middleware func(next HandlerFunc) HandlerFunc

// Chain 链式组合多个中间件
func Chain(middlewares ...Middleware) Middleware {
	return func(final HandlerFunc) HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
