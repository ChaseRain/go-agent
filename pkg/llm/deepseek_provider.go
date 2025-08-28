package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// DeepSeekPersonalProvider 实现DeepSeek个人版API的LLM提供者
// 使用DeepSeek个人API端点：https://api.deepseek.com/chat/completions
type DeepSeekPersonalProvider struct {
	config     *models.LLMConfig
	httpClient *http.Client
}

// NewDeepSeekPersonalProvider 创建DeepSeek个人版提供者
func NewDeepSeekPersonalProvider(config *models.LLMConfig) *DeepSeekPersonalProvider {
	// 设置默认值
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com"
	}
	if config.Model == "" {
		config.Model = "deepseek-chat"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4000
	}

	return &DeepSeekPersonalProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // 增加到120秒
		},
	}
}

// DeepSeek API 请求结构
type deepSeekRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Temperature float32                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
}

// DeepSeek API 响应结构
type deepSeekResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Call 进行同步LLM调用
func (p *DeepSeekPersonalProvider) Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	// 合并配置
	finalConfig := p.mergeConfig(config)

	// 构建请求体
	reqBody := deepSeekRequest{
		Model:       finalConfig.Model,
		Messages:    p.convertMessages(messages),
		Temperature: finalConfig.Temperature,
		MaxTokens:   finalConfig.MaxTokens,
		Stream:      false,
	}

	// 序列化请求
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/chat/completions", finalConfig.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", finalConfig.APIKey))

	// 执行请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求执行失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("DeepSeek API错误 [%d]: %s (类型: %s)", 
				resp.StatusCode, errorResp.Error.Message, errorResp.Error.Type)
		}
		return nil, fmt.Errorf("API返回错误状态 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp deepSeekResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API错误
	if apiResp.Error != nil {
		return nil, fmt.Errorf("DeepSeek API错误: %s (类型: %s, 代码: %s)", 
			apiResp.Error.Message, apiResp.Error.Type, apiResp.Error.Code)
	}

	// 提取响应内容
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("响应中没有选择项")
	}

	content := apiResp.Choices[0].Message.Content

	return &interfaces.LLMResponse{
		Content: content,
		Model:   apiResp.Model,
		Usage: interfaces.TokenUsage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
		Raw: apiResp,
	}, nil
}

// StreamCall 进行流式LLM调用（暂时使用非流式模拟实现）
func (p *DeepSeekPersonalProvider) StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *interfaces.LLMStreamChunk, error) {
	chunkChan := make(chan *interfaces.LLMStreamChunk, 100)

	go func() {
		defer close(chunkChan)

		// 先获取完整响应
		response, err := p.Call(ctx, messages, config)
		if err != nil {
			chunkChan <- &interfaces.LLMStreamChunk{
				Error: err,
			}
			return
		}

		// 模拟流式输出
		content := response.Content
		chunkSize := 30 // 每个块的字符数
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}

			chunk := &interfaces.LLMStreamChunk{
				Delta:  content[i:end],
				Finish: end >= len(content),
			}

			select {
			case chunkChan <- chunk:
			case <-ctx.Done():
				return
			}

			// 模拟流式延迟
			time.Sleep(50 * time.Millisecond)
		}
	}()

	return chunkChan, nil
}

// CountTokens 估算token数量
func (p *DeepSeekPersonalProvider) CountTokens(messages []models.Message) (int, error) {
	// 简单估算：中文约1.5字符/token，英文约4字符/token
	// 这里使用混合估算方式
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Role) + len(msg.Content) + 4 // 消息结构的额外token
		if msg.ReasoningContent != "" {
			totalChars += len(msg.ReasoningContent)
		}
	}

	// 粗略估算：平均每2.5个字符一个token（考虑中英文混合）
	tokens := (totalChars*2 + 4) / 5 // 约等于totalChars / 2.5
	
	// 为消息格式添加一些额外开销
	tokens += len(messages) * 3

	return tokens, nil
}

// 私有辅助方法

func (p *DeepSeekPersonalProvider) mergeConfig(config *models.LLMConfig) *models.LLMConfig {
	if config == nil {
		return p.config
	}

	merged := *p.config
	if config.Provider != "" {
		merged.Provider = config.Provider
	}
	if config.Model != "" {
		merged.Model = config.Model
	}
	if config.Temperature > 0 {
		merged.Temperature = config.Temperature
	}
	if config.MaxTokens > 0 {
		merged.MaxTokens = config.MaxTokens
	}
	if config.APIKey != "" {
		merged.APIKey = config.APIKey
	}
	if config.BaseURL != "" {
		merged.BaseURL = config.BaseURL
	}

	return &merged
}

func (p *DeepSeekPersonalProvider) convertMessages(messages []models.Message) []map[string]interface{} {
	converted := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}

		converted = append(converted, m)
	}

	return converted
}