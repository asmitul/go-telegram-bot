package pattern

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/handler"
)

const (
	FeatureCalculator = "calculator" // 计算器功能名称
)

// GroupRepository 群组仓储接口
type GroupRepository interface {
	FindByID(ctx context.Context, id int64) (*group.Group, error)
}

// CalculatorHandler 数学表达式计算器处理器
// 自动识别并计算数学表达式（支持 +, -, *, /, 括号）
type CalculatorHandler struct {
	pattern   *regexp.Regexp
	chatTypes []string
	groupRepo GroupRepository
}

// NewCalculatorHandler 创建计算器处理器
func NewCalculatorHandler(groupRepo GroupRepository) *CalculatorHandler {
	return &CalculatorHandler{
		// 匹配数学表达式：包含数字、运算符、括号、空格
		// 必须包含至少一个运算符（排除纯数字）
		pattern:   regexp.MustCompile(`^[\d\s\.\+\-\*/\(\)]+$`),
		chatTypes: []string{"group", "supergroup"},
		groupRepo: groupRepo,
	}
}

// Match 判断是否匹配
func (h *CalculatorHandler) Match(ctx *handler.Context) bool {
	// 1. 检查聊天类型（仅群组/超级群组）
	if !h.isSupportedChatType(ctx.ChatType) {
		return false
	}

	// 2. 检查文本是否为空
	if ctx.Text == "" {
		return false
	}

	// 3. 检查是否匹配数学表达式格式
	if !h.pattern.MatchString(ctx.Text) {
		return false
	}

	// 4. 排除纯数字（必须包含至少一个运算符）
	if !containsOperator(ctx.Text) {
		return false
	}

	// 5. 检查群组是否启用了计算器功能
	if h.groupRepo != nil {
		reqCtx := context.TODO()
		g, err := h.groupRepo.FindByID(reqCtx, ctx.ChatID)
		if err != nil {
			// 群组不存在时默认启用
			if err == group.ErrGroupNotFound {
				return true
			}
			// 数据库错误，为了安全起见不匹配
			return false
		}

		// 检查功能是否启用（默认启用）
		if !g.IsFeatureEnabled(FeatureCalculator) {
			return false
		}
	}

	return true
}

// Handle 处理消息
func (h *CalculatorHandler) Handle(ctx *handler.Context) error {
	expression := strings.TrimSpace(ctx.Text)

	// 计算表达式
	result, err := evaluateExpression(expression)
	if err != nil {
		return ctx.ReplyHTML(fmt.Sprintf("❌ %s", err.Error()))
	}

	// 格式化响应：<code>表达式</code> = <b>结果</b>
	return ctx.ReplyHTML(fmt.Sprintf("<code>%s</code> = <b>%s</b>", expression, formatNumber(result)))
}

// Priority 优先级
func (h *CalculatorHandler) Priority() int {
	return 310 // 正则处理器范围 300-399
}

// ContinueChain 处理后停止执行链
func (h *CalculatorHandler) ContinueChain() bool {
	return false
}

// isSupportedChatType 检查是否支持该聊天类型
func (h *CalculatorHandler) isSupportedChatType(chatType string) bool {
	for _, t := range h.chatTypes {
		if t == chatType {
			return true
		}
	}
	return false
}

// containsOperator 检查字符串是否包含至少一个运算符
func containsOperator(text string) bool {
	operators := "+-*/()"
	for _, char := range text {
		if strings.ContainsRune(operators, char) {
			return true
		}
	}
	return false
}

// evaluateExpression 计算数学表达式（使用双栈算法）
func evaluateExpression(expr string) (float64, error) {
	// 移除所有空格
	expr = strings.ReplaceAll(expr, " ", "")

	// 验证括号匹配
	if !isValidParentheses(expr) {
		return 0, fmt.Errorf("表达式格式错误：括号不匹配")
	}

	// 使用双栈算法求值
	numbers := []float64{}       // 操作数栈
	operators := []rune{}         // 运算符栈
	i := 0

	for i < len(expr) {
		char := rune(expr[i])

		// 1. 处理数字（包括小数和负数）
		if isDigit(char) || (char == '-' && (i == 0 || expr[i-1] == '(' || isOperator(rune(expr[i-1])))) {
			numStr := ""
			// 处理负号
			if char == '-' {
				numStr += "-"
				i++
				if i >= len(expr) {
					return 0, fmt.Errorf("表达式格式错误：负号后缺少数字")
				}
				char = rune(expr[i])
			}

			// 读取完整的数字
			for i < len(expr) && (isDigit(rune(expr[i])) || expr[i] == '.') {
				numStr += string(expr[i])
				i++
			}

			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("表达式格式错误：无效的数字 '%s'", numStr)
			}
			numbers = append(numbers, num)
			continue
		}

		// 2. 处理左括号
		if char == '(' {
			operators = append(operators, char)
			i++
			continue
		}

		// 3. 处理右括号
		if char == ')' {
			for len(operators) > 0 && operators[len(operators)-1] != '(' {
				if err := applyOperator(&numbers, &operators); err != nil {
					return 0, err
				}
			}
			if len(operators) == 0 {
				return 0, fmt.Errorf("表达式格式错误：括号不匹配")
			}
			// 弹出左括号
			operators = operators[:len(operators)-1]
			i++
			continue
		}

		// 4. 处理运算符
		if isOperator(char) {
			// 处理运算符优先级
			for len(operators) > 0 && precedence(operators[len(operators)-1]) >= precedence(char) {
				if err := applyOperator(&numbers, &operators); err != nil {
					return 0, err
				}
			}
			operators = append(operators, char)
			i++
			continue
		}

		// 无效字符
		return 0, fmt.Errorf("表达式格式错误：无效字符 '%c'", char)
	}

	// 处理剩余的运算符
	for len(operators) > 0 {
		if err := applyOperator(&numbers, &operators); err != nil {
			return 0, err
		}
	}

	// 最终结果
	if len(numbers) != 1 {
		return 0, fmt.Errorf("表达式格式错误：运算符或操作数不匹配")
	}

	return numbers[0], nil
}

// isValidParentheses 检查括号是否匹配
func isValidParentheses(expr string) bool {
	count := 0
	for _, char := range expr {
		if char == '(' {
			count++
		} else if char == ')' {
			count--
		}
		if count < 0 {
			return false
		}
	}
	return count == 0
}

// isDigit 检查是否为数字字符
func isDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

// isOperator 检查是否为运算符
func isOperator(char rune) bool {
	return char == '+' || char == '-' || char == '*' || char == '/'
}

// precedence 运算符优先级
func precedence(op rune) int {
	switch op {
	case '+', '-':
		return 1
	case '*', '/':
		return 2
	}
	return 0
}

// applyOperator 应用一个运算符
func applyOperator(numbers *[]float64, operators *[]rune) error {
	if len(*operators) == 0 {
		return fmt.Errorf("表达式格式错误：运算符缺失")
	}
	if len(*numbers) < 2 {
		return fmt.Errorf("表达式格式错误：操作数不足")
	}

	// 弹出运算符
	op := (*operators)[len(*operators)-1]
	*operators = (*operators)[:len(*operators)-1]

	// 弹出两个操作数（注意顺序）
	b := (*numbers)[len(*numbers)-1]
	*numbers = (*numbers)[:len(*numbers)-1]
	a := (*numbers)[len(*numbers)-1]
	*numbers = (*numbers)[:len(*numbers)-1]

	// 计算结果
	var result float64
	switch op {
	case '+':
		result = a + b
	case '-':
		result = a - b
	case '*':
		result = a * b
	case '/':
		if b == 0 {
			return fmt.Errorf("计算错误：除数不能为 0")
		}
		result = a / b
	default:
		return fmt.Errorf("表达式格式错误：未知运算符 '%c'", op)
	}

	// 将结果压入栈
	*numbers = append(*numbers, result)
	return nil
}

// formatNumber 格式化数字（去除不必要的小数点）
func formatNumber(num float64) string {
	// 如果是整数，不显示小数部分
	if num == float64(int64(num)) {
		return fmt.Sprintf("%d", int64(num))
	}
	// 否则显示最多 6 位小数
	return fmt.Sprintf("%.6g", num)
}
