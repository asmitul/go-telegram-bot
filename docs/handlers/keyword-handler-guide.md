# 关键词处理器开发指南

## 📚 目录

- [概述](#概述)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [完整代码示例](#完整代码示例)
- [关键词匹配策略](#关键词匹配策略)
- [注册流程](#注册流程)
- [测试方法](#测试方法)
- [实际场景示例](#实际场景示例)
- [常见问题](#常见问题)

---

## 概述

**关键词处理器** (Keyword Handler) 用于检测消息中是否包含特定关键词，并自动触发响应。适合自然对话场景和简单的关键词监控。

### 适用场景

- ✅ 礼貌用语自动回复（如"谢谢"→"不客气"）
- ✅ 常见问题自动解答（如"怎么用"→引导文档）
- ✅ 关键词监控和提醒（如敏感词检测）
- ✅ 多语言问候响应（如"你好"、"hello"）
- ✅ 简单的意图识别（如"帮助"、"价格"、"联系方式"）

### 不适用场景

- ❌ 需要精确命令格式 → 使用 **命令处理器** (`/command`)
- ❌ 需要复杂模式匹配或信息提取 → 使用 **正则匹配处理器**
- ❌ 需要处理所有消息 → 使用 **监听器**

---

## 核心概念

### 处理器接口

所有关键词处理器必须实现 `handler.Handler` 接口：

```go
type Handler interface {
    Match(ctx *Context) bool      // 判断是否包含关键词
    Handle(ctx *Context) error    // 处理匹配的消息
    Priority() int                // 优先级（200-299）
    ContinueChain() bool          // 是否继续执行后续处理器
}
```

### 优先级规则

- **优先级范围**：`200-299`
- **数值越小，优先级越高**（越早执行）
- **标准优先级**：`200`（推荐）
- **执行顺序**：命令处理器（100） → 关键词处理器（200） → 正则处理器（300） → 监听器（900+）

### 执行链控制

- `ContinueChain() = true`：继续执行后续处理器（**推荐**，允许日志记录）
- `ContinueChain() = false`：停止执行后续处理器（仅在确定是最终响应时使用）

### 匹配策略

- **包含匹配**：消息中包含关键词即触发（最常用）
- **精确匹配**：消息完全等于关键词
- **前缀匹配**：消息以关键词开头
- **后缀匹配**：消息以关键词结尾
- **大小写**：通常使用不区分大小写匹配

---

## 快速开始

### 步骤 1：创建处理器文件

在 `internal/handlers/keyword/` 目录下创建新文件，例如 `thanks.go`：

```bash
touch internal/handlers/keyword/thanks.go
```

### 步骤 2：编写处理器代码

```go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type ThanksHandler struct {
    keywords  []string
    chatTypes []string
}

func NewThanksHandler() *ThanksHandler {
    return &ThanksHandler{
        keywords:  []string{"谢谢", "感谢", "thanks", "thank you"},
        chatTypes: []string{"private"},
    }
}

func (h *ThanksHandler) Match(ctx *handler.Context) bool {
    // 检查聊天类型
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    // 检查是否包含关键词（不区分大小写）
    text := strings.ToLower(ctx.Text)
    for _, keyword := range h.keywords {
        if strings.Contains(text, strings.ToLower(keyword)) {
            return true
        }
    }

    return false
}

func (h *ThanksHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("不客气！很高兴能帮到你 😊")
}

func (h *ThanksHandler) Priority() int {
    return 200
}

func (h *ThanksHandler) ContinueChain() bool {
    return true // 继续记录日志
}

func (h *ThanksHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 步骤 3：注册处理器

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中添加：

```go
// 2. 关键词处理器（优先级 200）
router.Register(keyword.NewGreetingHandler())
router.Register(keyword.NewThanksHandler())  // 新增
```

### 步骤 4：测试

向机器人发送 `谢谢` 或 `thank you`，验证功能。

---

## 完整代码示例

### 示例 1：多语言问候（项目内置示例）

```go
package keyword

import (
    "fmt"
    "strings"
    "telegram-bot/internal/handler"
)

type GreetingHandler struct {
    keywords  []string
    chatTypes []string
}

func NewGreetingHandler() *GreetingHandler {
    return &GreetingHandler{
        keywords: []string{
            "你好", "您好", "hello", "hi", "嗨",
            "早上好", "晚上好", "下午好",
        },
        chatTypes: []string{"private"}, // 仅私聊响应
    }
}

func (h *GreetingHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(strings.TrimSpace(ctx.Text))
    for _, keyword := range h.keywords {
        if strings.Contains(text, strings.ToLower(keyword)) {
            return true
        }
    }

    return false
}

func (h *GreetingHandler) Handle(ctx *handler.Context) error {
    name := ctx.FirstName
    if name == "" {
        name = "朋友"
    }

    response := fmt.Sprintf(
        "你好，%s！👋\n\n"+
            "有什么可以帮你的吗？\n"+
            "输入 /help 查看可用命令。",
        name,
    )

    return ctx.Reply(response)
}

func (h *GreetingHandler) Priority() int {
    return 200
}

func (h *GreetingHandler) ContinueChain() bool {
    return true
}

func (h *GreetingHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 示例 2：FAQ 自动回复

```go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type FAQHandler struct {
    faqMap    map[string]string // 关键词 -> 回复
    chatTypes []string
}

func NewFAQHandler() *FAQHandler {
    return &FAQHandler{
        faqMap: map[string]string{
            "价格":    "💰 价格信息：\n• 基础版：免费\n• 专业版：¥99/月\n• 企业版：联系客服",
            "怎么用":   "📖 使用方法：\n1. 输入 /help 查看所有命令\n2. 输入 /start 开始使用\n3. 查看文档：https://docs.example.com",
            "联系":    "📞 联系我们：\n• 邮箱：support@example.com\n• 电话：400-123-4567\n• 工作时间：9:00-18:00",
            "功能":    "✨ 主要功能：\n• 自动回复\n• 权限管理\n• 数据统计\n• 自定义命令",
        },
        chatTypes: []string{"private", "group", "supergroup"},
    }
}

func (h *FAQHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(ctx.Text)

    // 检查是否包含任何 FAQ 关键词
    for keyword := range h.faqMap {
        if strings.Contains(text, strings.ToLower(keyword)) {
            return true
        }
    }

    return false
}

func (h *FAQHandler) Handle(ctx *handler.Context) error {
    text := strings.ToLower(ctx.Text)

    // 找到匹配的 FAQ 并回复
    for keyword, answer := range h.faqMap {
        if strings.Contains(text, strings.ToLower(keyword)) {
            return ctx.Reply(answer)
        }
    }

    return nil
}

func (h *FAQHandler) Priority() int {
    return 210 // 稍低优先级，避免覆盖问候语
}

func (h *FAQHandler) ContinueChain() bool {
    return true
}

func (h *FAQHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 示例 3：敏感词监控

```go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type SensitiveWordHandler struct {
    sensitiveWords []string
    chatTypes      []string
    logger         Logger
}

type Logger interface {
    Warn(msg string, fields ...interface{})
}

func NewSensitiveWordHandler(logger Logger) *SensitiveWordHandler {
    return &SensitiveWordHandler{
        sensitiveWords: []string{
            "广告", "spam", "垃圾信息",
            // 实际项目中应该从配置文件或数据库加载
        },
        chatTypes: []string{"group", "supergroup"},
        logger:    logger,
    }
}

func (h *SensitiveWordHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(ctx.Text)

    for _, word := range h.sensitiveWords {
        if strings.Contains(text, strings.ToLower(word)) {
            return true
        }
    }

    return false
}

func (h *SensitiveWordHandler) Handle(ctx *handler.Context) error {
    // 记录敏感词日志
    h.logger.Warn("sensitive_word_detected",
        "chat_id", ctx.ChatID,
        "user_id", ctx.UserID,
        "text", ctx.Text,
    )

    // 可选：通知管理员
    // notifyAdmins(ctx.ChatID, ctx.UserID, ctx.Text)

    // 可选：删除消息（需要机器人有删除权限）
    // ctx.DeleteMessage(ctx.MessageID)

    // 警告用户
    return ctx.Reply("⚠️ 检测到敏感内容，请注意言论规范")
}

func (h *SensitiveWordHandler) Priority() int {
    return 200 // 高优先级，尽早检测
}

func (h *SensitiveWordHandler) ContinueChain() bool {
    return true // 继续执行其他处理器（如日志记录）
}

func (h *SensitiveWordHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 示例 4：智能关键词响应（带优先级）

```go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type SmartKeywordHandler struct {
    keywordGroups []KeywordGroup
    chatTypes     []string
}

type KeywordGroup struct {
    Keywords []string
    Response string
    Priority int // 内部优先级（关键词组之间的优先级）
}

func NewSmartKeywordHandler() *SmartKeywordHandler {
    return &SmartKeywordHandler{
        keywordGroups: []KeywordGroup{
            // 优先级 1：紧急问题
            {
                Keywords: []string{"无法登录", "登录失败", "忘记密码"},
                Response: "🔑 登录问题：\n1. 点击"忘记密码"重置\n2. 检查用户名是否正确\n3. 联系客服：/contact",
                Priority: 1,
            },
            // 优先级 2：账户问题
            {
                Keywords: []string{"账户", "账号", "会员"},
                Response: "👤 账户相关：\n• 查看信息：/whoami\n• 升级会员：/upgrade\n• 修改资料：/profile",
                Priority: 2,
            },
            // 优先级 3：一般问题
            {
                Keywords: []string{"问题", "疑问", "不懂"},
                Response: "❓ 遇到问题了吗？\n• 常见问题：/faq\n• 联系客服：/contact\n• 查看文档：https://docs.example.com",
                Priority: 3,
            },
        },
        chatTypes: []string{"private"},
    }
}

func (h *SmartKeywordHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(ctx.Text)

    for _, group := range h.keywordGroups {
        for _, keyword := range group.Keywords {
            if strings.Contains(text, strings.ToLower(keyword)) {
                return true
            }
        }
    }

    return false
}

func (h *SmartKeywordHandler) Handle(ctx *handler.Context) error {
    text := strings.ToLower(ctx.Text)

    // 找到优先级最高的匹配组
    var matchedGroup *KeywordGroup

    for i := range h.keywordGroups {
        group := &h.keywordGroups[i]
        for _, keyword := range group.Keywords {
            if strings.Contains(text, strings.ToLower(keyword)) {
                if matchedGroup == nil || group.Priority < matchedGroup.Priority {
                    matchedGroup = group
                }
            }
        }
    }

    if matchedGroup != nil {
        return ctx.Reply(matchedGroup.Response)
    }

    return nil
}

func (h *SmartKeywordHandler) Priority() int {
    return 200
}

func (h *SmartKeywordHandler) ContinueChain() bool {
    return true
}

func (h *SmartKeywordHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

---

## 关键词匹配策略

### 1. 包含匹配（最常用）

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    return strings.Contains(text, "关键词")
}
```

**匹配示例**：
- ✅ "你好" → "你好吗"
- ✅ "你好" → "我想说你好"
- ✅ "谢谢" → "非常谢谢你"

### 2. 精确匹配

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(strings.TrimSpace(ctx.Text))
    return text == "关键词"
}
```

**匹配示例**：
- ✅ "help" → "help"
- ❌ "help" → "help me"
- ❌ "help" → "I need help"

### 3. 前缀匹配

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    return strings.HasPrefix(text, "关键词")
}
```

**匹配示例**：
- ✅ "查询" → "查询余额"
- ✅ "查询" → "查询订单"
- ❌ "查询" → "余额查询"

### 4. 后缀匹配

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    return strings.HasSuffix(text, "关键词")
}
```

**匹配示例**：
- ✅ "吗" → "是这样吗"
- ✅ "吗" → "你好吗"
- ❌ "吗" → "吗啡"

### 5. 多关键词匹配（任意一个）

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    keywords := []string{"谢谢", "感谢", "thanks"}

    for _, keyword := range keywords {
        if strings.Contains(text, keyword) {
            return true
        }
    }

    return false
}
```

### 6. 多关键词匹配（全部）

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    requiredKeywords := []string{"价格", "会员"}

    for _, keyword := range requiredKeywords {
        if !strings.Contains(text, keyword) {
            return false
        }
    }

    return true
}
```

**匹配示例**：
- ✅ "会员价格是多少" → 包含"价格"和"会员"
- ❌ "价格是多少" → 只包含"价格"
- ❌ "会员有什么特权" → 只包含"会员"

### 7. 排除关键词

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)

    // 必须包含
    if !strings.Contains(text, "帮助") {
        return false
    }

    // 必须不包含（排除命令）
    excludeKeywords := []string{"/help", "/帮助"}
    for _, exclude := range excludeKeywords {
        if strings.Contains(text, strings.ToLower(exclude)) {
            return false
        }
    }

    return true
}
```

**匹配示例**：
- ✅ "需要帮助" → 包含"帮助"，不是命令
- ❌ "/help" → 是命令
- ❌ "无关内容" → 不包含"帮助"

### 8. 完整单词匹配

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    keyword := "go"

    // 使用正则或手动检查边界
    words := strings.Fields(text)
    for _, word := range words {
        if word == keyword {
            return true
        }
    }

    return false
}
```

**匹配示例**：
- ✅ "I love go" → "go" 是完整单词
- ❌ "I love golang" → "go" 不是完整单词

---

## 注册流程

### 1. 基本注册

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中：

```go
func registerHandlers(
    router *handler.Router,
    groupRepo *mongodb.GroupRepository,
    userRepo *mongodb.UserRepository,
    appLogger logger.Logger,
) {
    // 1. 命令处理器（优先级 100）
    router.Register(command.NewPingHandler(groupRepo))
    router.Register(command.NewHelpHandler(groupRepo, router))
    router.Register(command.NewStatsHandler(groupRepo, userRepo))

    // 2. 关键词处理器（优先级 200）
    router.Register(keyword.NewGreetingHandler())
    router.Register(keyword.NewThanksHandler())    // 新增
    router.Register(keyword.NewFAQHandler())       // 新增

    // 3. 正则处理器（优先级 300）
    router.Register(pattern.NewWeatherHandler())

    // 4. 监听器（优先级 900+）
    router.Register(listener.NewMessageLoggerHandler(appLogger))
    router.Register(listener.NewAnalyticsHandler())

    appLogger.Info("Registered handlers breakdown",
        "commands", 3,
        "keywords", 3, // 更新数量
        "patterns", 1,
        "listeners", 2,
    )
}
```

### 2. 带依赖注入

```go
// 需要 logger 依赖
router.Register(keyword.NewSensitiveWordHandler(appLogger))

// 需要数据库依赖
type KeywordConfigHandler struct {
    keywords  []string
    groupRepo GroupRepository
}

func NewKeywordConfigHandler(groupRepo GroupRepository) *KeywordConfigHandler {
    return &KeywordConfigHandler{
        keywords:  loadKeywordsFromDB(groupRepo),
        groupRepo: groupRepo,
    }
}

router.Register(keyword.NewKeywordConfigHandler(groupRepo))
```

---

## 测试方法

### 1. 单元测试

创建 `internal/handlers/keyword/thanks_test.go`：

```go
package keyword

import (
    "testing"
    "telegram-bot/internal/handler"
    "github.com/stretchr/testify/assert"
)

func TestThanksHandler_Match(t *testing.T) {
    h := NewThanksHandler()

    tests := []struct {
        name     string
        text     string
        chatType string
        want     bool
    }{
        {"匹配-谢谢", "谢谢", "private", true},
        {"匹配-感谢", "非常感谢你", "private", true},
        {"匹配-thanks", "thanks a lot", "private", true},
        {"匹配-大小写", "THANK YOU", "private", true},
        {"不匹配-其他", "你好", "private", false},
        {"不匹配-群组", "谢谢", "group", false}, // 仅私聊
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := &handler.Context{
                Text:     tt.text,
                ChatType: tt.chatType,
            }
            got := h.Match(ctx)
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestThanksHandler_Priority(t *testing.T) {
    h := NewThanksHandler()
    assert.Equal(t, 200, h.Priority())
}

func TestThanksHandler_ContinueChain(t *testing.T) {
    h := NewThanksHandler()
    assert.True(t, h.ContinueChain())
}
```

运行测试：

```bash
go test ./internal/handlers/keyword/... -v
```

### 2. 手动测试

1. 启动机器人：
   ```bash
   make run
   ```

2. 在 Telegram 中测试：
   - 发送 `谢谢` → 验证基本匹配
   - 发送 `非常感谢你` → 验证包含匹配
   - 发送 `THANK YOU` → 验证大小写不敏感
   - 在群组中发送 → 验证聊天类型过滤

3. 检查日志：
   ```
   INFO  message_logged text="谢谢"
   INFO  keyword_matched handler=ThanksHandler
   ```

---

## 实际场景示例

### 场景 1：客服引导

```go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type CustomerServiceHandler struct {
    keywords  []string
    chatTypes []string
}

func NewCustomerServiceHandler() *CustomerServiceHandler {
    return &CustomerServiceHandler{
        keywords:  []string{"客服", "人工", "投诉", "反馈"},
        chatTypes: []string{"private"},
    }
}

func (h *CustomerServiceHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(ctx.Text)
    for _, keyword := range h.keywords {
        if strings.Contains(text, keyword) {
            return true
        }
    }

    return false
}

func (h *CustomerServiceHandler) Handle(ctx *handler.Context) error {
    response := "👨‍💼 *客服服务*\n\n" +
        "我们提供以下服务：\n" +
        "• 💬 在线客服：周一至周五 9:00-18:00\n" +
        "• 📧 邮箱：support@example.com\n" +
        "• 📞 电话：400-123-4567\n" +
        "• 🎫 提交工单：/ticket\n\n" +
        "请描述您的问题，我们会尽快回复！"

    return ctx.ReplyMarkdown(response)
}

func (h *CustomerServiceHandler) Priority() int {
    return 200
}

func (h *CustomerServiceHandler) ContinueChain() bool {
    return true
}

func (h *CustomerServiceHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 场景 2：情感识别

```go
package keyword

import (
    "math/rand"
    "strings"
    "telegram-bot/internal/handler"
)

type EmotionHandler struct {
    positiveWords []string
    negativeWords []string
    chatTypes     []string
}

func NewEmotionHandler() *EmotionHandler {
    return &EmotionHandler{
        positiveWords: []string{"开心", "高兴", "快乐", "棒", "太好了"},
        negativeWords: []string{"难过", "伤心", "失望", "糟糕", "不开心"},
        chatTypes:     []string{"private"},
    }
}

func (h *EmotionHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(ctx.Text)

    for _, word := range h.positiveWords {
        if strings.Contains(text, word) {
            return true
        }
    }

    for _, word := range h.negativeWords {
        if strings.Contains(text, word) {
            return true
        }
    }

    return false
}

func (h *EmotionHandler) Handle(ctx *handler.Context) error {
    text := strings.ToLower(ctx.Text)

    // 判断情感
    isPositive := false
    for _, word := range h.positiveWords {
        if strings.Contains(text, word) {
            isPositive = true
            break
        }
    }

    if isPositive {
        responses := []string{
            "太好了！我也很高兴 😊",
            "真为你开心！✨",
            "继续保持好心情哦 🌟",
        }
        return ctx.Reply(responses[rand.Intn(len(responses))])
    }

    // 负面情感
    responses := []string{
        "别难过，一切都会好起来的 💪",
        "要相信明天会更好！☀️",
        "有什么可以帮到你的吗？",
    }
    return ctx.Reply(responses[rand.Intn(len(responses))])
}

func (h *EmotionHandler) Priority() int {
    return 250 // 较低优先级
}

func (h *EmotionHandler) ContinueChain() bool {
    return true
}

func (h *EmotionHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 场景 3：群组规则提醒

```go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type GroupRulesHandler struct {
    keywords  []string
    chatTypes []string
}

func NewGroupRulesHandler() *GroupRulesHandler {
    return &GroupRulesHandler{
        keywords:  []string{"群规", "规则", "规定", "规矩"},
        chatTypes: []string{"group", "supergroup"},
    }
}

func (h *GroupRulesHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    text := strings.ToLower(ctx.Text)
    for _, keyword := range h.keywords {
        if strings.Contains(text, keyword) {
            return true
        }
    }

    return false
}

func (h *GroupRulesHandler) Handle(ctx *handler.Context) error {
    response := "📋 *群组规则*\n\n" +
        "1️⃣ 禁止发送广告和垃圾信息\n" +
        "2️⃣ 尊重他人，文明交流\n" +
        "3️⃣ 不得发送违法违规内容\n" +
        "4️⃣ 禁止恶意刷屏\n" +
        "5️⃣ 遵守 Telegram 使用条款\n\n" +
        "违规者将被警告或移出群组 ⚠️"

    return ctx.ReplyMarkdown(response)
}

func (h *GroupRulesHandler) Priority() int {
    return 200
}

func (h *GroupRulesHandler) ContinueChain() bool {
    return true
}

func (h *GroupRulesHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

---

## 常见问题

### Q1：关键词处理器和命令处理器的区别？

| 特性 | 关键词处理器 | 命令处理器 |
|------|------------|-----------|
| **触发方式** | 包含关键词 | `/command` 格式 |
| **匹配方式** | 模糊匹配 | 精确匹配 |
| **优先级** | 200-299 | 100-199 |
| **权限系统** | 需手动实现 | 内置 BaseCommand |
| **适用场景** | 自然对话 | 明确指令 |

### Q2：关键词处理器和正则处理器的区别？

| 特性 | 关键词处理器 | 正则处理器 |
|------|------------|-----------|
| **复杂度** | 简单 | 复杂 |
| **性能** | 更快（字符串包含） | 较慢（正则匹配） |
| **信息提取** | 不支持 | 支持捕获组 |
| **适用场景** | 简单关键词 | 复杂模式 |

### Q3：如何避免关键词冲突？

1. **使用优先级**：重要的关键词处理器使用更低的数字
2. **精确的关键词**：避免过于宽泛的关键词（如"的"、"是"）
3. **检查聊天类型**：限制在特定聊天类型中生效
4. **使用 ContinueChain**：如果不是最终响应，设置为 `true`

### Q4：如何实现区分大小写的匹配？

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    text := ctx.Text // 不转小写

    for _, keyword := range h.keywords {
        if strings.Contains(text, keyword) { // 区分大小写
            return true
        }
    }

    return false
}
```

### Q5：如何动态加载关键词（从数据库）？

```go
type DynamicKeywordHandler struct {
    chatTypes []string
    groupRepo GroupRepository
}

func (h *DynamicKeywordHandler) Match(ctx *handler.Context) bool {
    // 从数据库加载当前群组的关键词配置
    group, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        return false
    }

    text := strings.ToLower(ctx.Text)

    for _, keyword := range group.CustomKeywords {
        if strings.Contains(text, strings.ToLower(keyword)) {
            return true
        }
    }

    return false
}
```

### Q6：应该使用 `ContinueChain() = true` 还是 `false`？

**推荐使用 `true`**，原因：

- ✅ 允许监听器记录日志
- ✅ 允许多个关键词处理器同时响应
- ✅ 不会阻断后续的监控和统计

**使用 `false` 的场景**：

- 确定这是用户意图的最终响应
- 避免触发其他可能冲突的处理器

### Q7：如何处理多个关键词同时匹配？

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    // 优先级最高的关键词
    highPriority := []string{"紧急", "urgent"}
    // 普通关键词
    normalPriority := []string{"帮助", "help"}

    text := strings.ToLower(ctx.Text)

    // 先检查高优先级
    for _, keyword := range highPriority {
        if strings.Contains(text, keyword) {
            ctx.Set("keyword_priority", "high")
            return true
        }
    }

    // 再检查普通优先级
    for _, keyword := range normalPriority {
        if strings.Contains(text, keyword) {
            ctx.Set("keyword_priority", "normal")
            return true
        }
    }

    return false
}

func (h *Handler) Handle(ctx *handler.Context) error {
    priority, _ := ctx.Get("keyword_priority")

    if priority == "high" {
        return ctx.Reply("🚨 紧急问题将优先处理！")
    }

    return ctx.Reply("ℹ️ 我来帮助你")
}
```

---

## 附录

### 相关资源

- [项目内置示例](../../internal/handlers/keyword/greeting.go)
- [Go strings 包文档](https://pkg.go.dev/strings)

### 相关文档

- [命令处理器开发指南](./command-handler-guide.md)
- [正则匹配处理器开发指南](./pattern-handler-guide.md)
- [监听器开发指南](./listener-handler-guide.md)（待创建）
- [架构总览](../../CLAUDE.md)

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
