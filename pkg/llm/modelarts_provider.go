package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// ModelArtsProvider implements LLM provider for ModelArts MAAS API
type ModelArtsProvider struct {
	config     *models.LLMConfig
	httpClient *http.Client
}

// NewModelArtsProvider creates a new ModelArts MAAS provider
func NewModelArtsProvider(config *models.LLMConfig) *ModelArtsProvider {
	// Set default values for ModelArts MAAS
	if config.BaseURL == "" {
		config.BaseURL = "https://api.modelarts-maas.com/v1"
	}
	if config.Model == "" {
		config.Model = "DeepSeek-V3"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2000
	}

	return &ModelArtsProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ModelArts API Request/Response structures (OpenAI compatible format)
type modelArtsRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Temperature float32                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
}

type modelArtsResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int                    `json:"index"`
		Message      map[string]interface{} `json:"message"`
		Delta        map[string]interface{} `json:"delta,omitempty"`
		FinishReason string                 `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type modelArtsError struct {
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
	SpanID    string `json:"span_id"`
}

// Call makes a synchronous call to ModelArts MAAS API
func (p *ModelArtsProvider) Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	// Merge configs
	finalConfig := p.mergeConfig(config)

	// Build request
	reqBody := modelArtsRequest{
		Model:       finalConfig.Model,
		Messages:    p.convertMessages(messages),
		Temperature: finalConfig.Temperature,
		MaxTokens:   finalConfig.MaxTokens,
		Stream:      false,
	}

	// Make API call
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", 
		fmt.Sprintf("%s/chat/completions", finalConfig.BaseURL),
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for ModelArts MAAS
	req.Header.Set("Content-Type", "application/json")
	// ModelArts MAAS uses X-Auth-Token instead of Authorization Bearer
	req.Header.Set("X-Auth-Token", finalConfig.APIKey)
	
	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for ModelArts error first
	if resp.StatusCode != http.StatusOK {
		var errResp modelArtsError
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.ErrorCode != "" {
			return nil, fmt.Errorf("ModelArts API error %s: %s (span_id: %s)", 
				errResp.ErrorCode, errResp.ErrorMsg, errResp.SpanID)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse success response
	var apiResp modelArtsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract response content
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := ""
	if msg := apiResp.Choices[0].Message; msg != nil {
		if c, ok := msg["content"].(string); ok {
			content = c
		}
	}

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

// StreamCall makes a streaming call to ModelArts MAAS API
func (p *ModelArtsProvider) StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *interfaces.LLMStreamChunk, error) {
	// Merge configs
	finalConfig := p.mergeConfig(config)

	// Build request
	reqBody := modelArtsRequest{
		Model:       finalConfig.Model,
		Messages:    p.convertMessages(messages),
		Temperature: finalConfig.Temperature,
		MaxTokens:   finalConfig.MaxTokens,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/chat/completions", finalConfig.BaseURL),
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for streaming
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", finalConfig.APIKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var errResp modelArtsError
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.ErrorCode != "" {
			return nil, fmt.Errorf("ModelArts API error %s: %s", errResp.ErrorCode, errResp.ErrorMsg)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create channel for streaming
	chunkChan := make(chan *interfaces.LLMStreamChunk, 100)

	// Start goroutine to read SSE stream
	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					chunkChan <- &interfaces.LLMStreamChunk{
						Error: fmt.Errorf("stream read error: %w", err),
					}
				}
				break
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Parse SSE data
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				
				// Check for stream end
				if data == "[DONE]" {
					chunkChan <- &interfaces.LLMStreamChunk{
						Finish: true,
					}
					break
				}

				// Parse JSON
				var streamResp modelArtsResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					// Check if it's an error response
					var errResp modelArtsError
					if err := json.Unmarshal([]byte(data), &errResp); err == nil && errResp.ErrorCode != "" {
						chunkChan <- &interfaces.LLMStreamChunk{
							Error: fmt.Errorf("ModelArts error: %s", errResp.ErrorMsg),
						}
						break
					}
					continue // Skip malformed data
				}

				// Extract delta content
				if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta != nil {
					if content, ok := streamResp.Choices[0].Delta["content"].(string); ok {
						chunk := &interfaces.LLMStreamChunk{
							Delta:  content,
							Finish: streamResp.Choices[0].FinishReason != "",
						}
						
						select {
						case chunkChan <- chunk:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return chunkChan, nil
}

// CountTokens estimates token count
func (p *ModelArtsProvider) CountTokens(messages []models.Message) (int, error) {
	// Simple estimation - ModelArts uses similar tokenization to OpenAI
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Role) + len(msg.Content) + 4
		if msg.ReasoningContent != "" {
			totalChars += len(msg.ReasoningContent)
		}
	}
	
	// Rough estimation: ~4 chars per token for English/Chinese mixed text
	tokens := (totalChars / 4) + (len(messages) * 3)
	
	return tokens, nil
}

// Private helper methods

func (p *ModelArtsProvider) mergeConfig(config *models.LLMConfig) *models.LLMConfig {
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

func (p *ModelArtsProvider) convertMessages(messages []models.Message) []map[string]interface{} {
	converted := make([]map[string]interface{}, 0, len(messages))
	
	for _, msg := range messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		
		// Add name field if present in message Type
		if msg.Type != "" {
			m["name"] = msg.Type
		}
		
		converted = append(converted, m)
	}
	
	return converted
}