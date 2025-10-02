package handler

// Handler 统一的消息处理器接口
// 所有类型的处理器（命令、关键词、正则、监听器等）都实现此接口
type Handler interface {
	// Match 判断是否应该处理这条消息
	// 返回 true 表示匹配，将执行 Handle 方法
	Match(ctx *Context) bool

	// Handle 处理消息
	// 返回 error 表示处理失败，error 会被记录并可能发送给用户
	Handle(ctx *Context) error

	// Priority 优先级（数字越小越优先执行）
	// 建议值：
	//   - 0-99:   系统级处理器
	//   - 100-199: 命令处理器
	//   - 200-299: 关键词处理器
	//   - 300-399: 正则处理器
	//   - 400-499: 交互式处理器（按钮、表单等）
	//   - 900-999: 监听器（记录、统计等）
	Priority() int

	// ContinueChain 处理后是否继续执行后续处理器
	// 返回 true 表示继续执行链，false 表示停止
	// 通常命令返回 false，监听器返回 true
	ContinueChain() bool
}

// HandlerFunc 处理函数类型
type HandlerFunc func(ctx *Context) error
