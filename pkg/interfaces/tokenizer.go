package interfaces

import "go-agent/pkg/models"

// Tokenizer 定义token计算接口
type Tokenizer interface {
	// CountTokens 计算文本的token数量
	CountTokens(text string) int

	// EstimateTokens 估算消息列表的总token数
	EstimateTokens(messages []models.Message) int
}
