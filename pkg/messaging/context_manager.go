package messaging

import (
	"fmt"
	"sort"
	"sync"

	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// ContextManager 实现完整的消息上下文管理
type ContextManager struct {
	messages    []models.Message
	mu          sync.RWMutex
	maxMessages int
	maxTokens   int
	tokenizer   interfaces.Tokenizer
}

// MessageContext 消息上下文结构
type MessageContext struct {
	Messages     []models.Message `json:"messages"`
	TokenCount   int              `json:"token_count"`
	MessageCount int              `json:"message_count"`
	WindowSize   int              `json:"window_size"`
	Truncated    bool             `json:"truncated"`
}

// NewContextManager 创建新的上下文管理器
func NewContextManager(maxMessages, maxTokens int, tokenizer interfaces.Tokenizer) *ContextManager {
	if maxMessages <= 0 {
		maxMessages = 50 // 默认最大消息数
	}
	if maxTokens <= 0 {
		maxTokens = 4000 // 默认最大token数
	}

	return &ContextManager{
		messages:    make([]models.Message, 0),
		maxMessages: maxMessages,
		maxTokens:   maxTokens,
		tokenizer:   tokenizer,
	}
}

// AddMessage 添加消息到历史记录
func (c *ContextManager) AddMessage(message models.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 添加时间戳
	if message.Type == "" {
		message.Type = "text"
	}

	// 验证消息
	if err := c.validateMessage(&message); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// 添加到历史记录
	c.messages = append(c.messages, message)

	// 自动清理过多的消息
	c.trimMessages()

	return nil
}

// GetHistory 获取完整的消息历史记录
func (c *ContextManager) GetHistory() []models.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本以防止并发修改
	history := make([]models.Message, len(c.messages))
	copy(history, c.messages)
	return history
}

// GetContextMessages 获取适合当前上下文的消息
func (c *ContextManager) GetContextMessages(maxTokens int) []models.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	return c.buildContextWindow(maxTokens)
}

// GetContextWithWindow 获取带窗口信息的上下文
func (c *ContextManager) GetContextWithWindow(maxTokens int) *MessageContext {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if maxTokens <= 0 {
		maxTokens = c.maxTokens
	}

	messages := c.buildContextWindow(maxTokens)
	tokenCount := c.calculateTokens(messages)

	return &MessageContext{
		Messages:     messages,
		TokenCount:   tokenCount,
		MessageCount: len(messages),
		WindowSize:   maxTokens,
		Truncated:    len(messages) < len(c.messages),
	}
}

// ClearHistory 清除消息历史记录
func (c *ContextManager) ClearHistory() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages = make([]models.Message, 0)
	return nil
}

// GetMessagesByRole 获取指定角色的消息
func (c *ContextManager) GetMessagesByRole(role string) []models.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var filtered []models.Message
	for _, msg := range c.messages {
		if msg.Role == role {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// GetRecentMessages 获取最近的N条消息
func (c *ContextManager) GetRecentMessages(count int) []models.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if count <= 0 || count > len(c.messages) {
		count = len(c.messages)
	}

	start := len(c.messages) - count
	recent := make([]models.Message, count)
	copy(recent, c.messages[start:])
	return recent
}

// SearchMessages 搜索包含特定内容的消息
func (c *ContextManager) SearchMessages(query string) []models.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []models.Message
	for _, msg := range c.messages {
		if contains(msg.Content, query) || contains(msg.ReasoningContent, query) {
			results = append(results, msg)
		}
	}
	return results
}

// GetStatistics 获取消息统计信息
func (c *ContextManager) GetStatistics() MessageStatistics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := MessageStatistics{
		TotalMessages: len(c.messages),
		RoleCount:     make(map[string]int),
		TypeCount:     make(map[string]int),
		TotalTokens:   c.calculateTokens(c.messages),
	}

	for _, msg := range c.messages {
		stats.RoleCount[msg.Role]++
		stats.TypeCount[msg.Type]++
	}

	if len(c.messages) > 0 {
		stats.FirstMessage = c.messages[0]
		stats.LastMessage = c.messages[len(c.messages)-1]
	}

	return stats
}

// MessageStatistics 消息统计信息
type MessageStatistics struct {
	TotalMessages int            `json:"total_messages"`
	RoleCount     map[string]int `json:"role_count"`
	TypeCount     map[string]int `json:"type_count"`
	TotalTokens   int            `json:"total_tokens"`
	FirstMessage  models.Message `json:"first_message,omitempty"`
	LastMessage   models.Message `json:"last_message,omitempty"`
}

// 私有方法

func (c *ContextManager) validateMessage(message *models.Message) error {
	if message.Role == "" {
		return fmt.Errorf("message role cannot be empty")
	}
	if message.Content == "" && message.ReasoningContent == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	// 验证角色
	validRoles := map[string]bool{
		"system": true, "user": true, "assistant": true, "tool": true,
	}
	if !validRoles[message.Role] {
		return fmt.Errorf("invalid message role: %s", message.Role)
	}

	return nil
}

func (c *ContextManager) trimMessages() {
	// 如果超过最大消息数，删除最旧的消息
	if len(c.messages) > c.maxMessages {
		excess := len(c.messages) - c.maxMessages
		c.messages = c.messages[excess:]
	}
}

func (c *ContextManager) buildContextWindow(maxTokens int) []models.Message {
	if len(c.messages) == 0 {
		return []models.Message{}
	}

	// 总是包含系统消息
	var contextMessages []models.Message
	var currentTokens int

	// 首先添加系统消息
	for _, msg := range c.messages {
		if msg.Role == "system" {
			contextMessages = append(contextMessages, msg)
			if c.tokenizer != nil {
				currentTokens += c.tokenizer.CountTokens(msg.Content + msg.ReasoningContent)
			}
		}
	}

	// 从最新的消息开始往前添加
	nonSystemMessages := make([]models.Message, 0)
	for _, msg := range c.messages {
		if msg.Role != "system" {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	// 倒序遍历，优先保留最新的消息
	for i := len(nonSystemMessages) - 1; i >= 0; i-- {
		msg := nonSystemMessages[i]
		var tokens int
		if c.tokenizer != nil {
			tokens = c.tokenizer.CountTokens(msg.Content + msg.ReasoningContent)
		} else {
			// 简单估算：每个token约4个字符
			tokens = (len(msg.Content) + len(msg.ReasoningContent)) / 4
		}

		if currentTokens+tokens > maxTokens {
			break
		}

		contextMessages = append([]models.Message{msg}, contextMessages...)
		currentTokens += tokens
	}

	// 重新排序以保持时间顺序
	sort.Slice(contextMessages, func(i, j int) bool {
		// 系统消息始终在前
		if contextMessages[i].Role == "system" && contextMessages[j].Role != "system" {
			return true
		}
		if contextMessages[i].Role != "system" && contextMessages[j].Role == "system" {
			return false
		}
		// 其他消息按原始顺序
		return false
	})

	return contextMessages
}

func (c *ContextManager) calculateTokens(messages []models.Message) int {
	if c.tokenizer == nil {
		// 简单估算
		total := 0
		for _, msg := range messages {
			total += (len(msg.Content) + len(msg.ReasoningContent)) / 4
		}
		return total
	}

	total := 0
	for _, msg := range messages {
		total += c.tokenizer.CountTokens(msg.Content + msg.ReasoningContent)
	}
	return total
}

func contains(text, query string) bool {
	if query == "" {
		return true
	}
	// 简单的字符串包含检查，生产环境可以使用更复杂的搜索算法
	return len(text) > 0 && len(query) <= len(text) &&
		findSubstring(text, query) >= 0
}

func findSubstring(text, pattern string) int {
	if len(pattern) == 0 {
		return 0
	}
	if len(pattern) > len(text) {
		return -1
	}

	for i := 0; i <= len(text)-len(pattern); i++ {
		if text[i:i+len(pattern)] == pattern {
			return i
		}
	}
	return -1
}

// SimpleTokenizer 简单的token计算器实现
type SimpleTokenizer struct{}

func (t *SimpleTokenizer) CountTokens(text string) int {
	// 简单估算：平均每个token约4个字符
	if len(text) == 0 {
		return 0
	}
	return (len(text) + 3) / 4 // 向上取整
}

func (t *SimpleTokenizer) EstimateTokens(messages []models.Message) int {
	total := 0
	for _, msg := range messages {
		total += t.CountTokens(msg.Content + msg.ReasoningContent)
		total += 4 // 每个消息的元数据开销
	}
	return total
}

// NewSimpleTokenizer 创建简单token计算器
func NewSimpleTokenizer() interfaces.Tokenizer {
	return &SimpleTokenizer{}
}
