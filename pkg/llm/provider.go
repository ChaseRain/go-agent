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

// SimpleLLMProvider implements a basic LLM provider using HTTP API
type SimpleLLMProvider struct {
	config     *models.LLMConfig
	httpClient *http.Client
}

// NewSimpleLLMProvider creates a new SimpleLLMProvider instance
func NewSimpleLLMProvider(config *models.LLMConfig) *SimpleLLMProvider {
	return &SimpleLLMProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Call makes a synchronous call to the LLM
func (p *SimpleLLMProvider) Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	// Merge configs (parameter config overrides provider config)
	finalConfig := p.mergeConfig(config)
	
	// Build request
	request := p.buildRequest(messages, finalConfig)
	
	// Make API call
	response, err := p.makeAPICall(ctx, request, finalConfig)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	
	return response, nil
}

// StreamCall makes a streaming call to the LLM
func (p *SimpleLLMProvider) StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *interfaces.LLMStreamChunk, error) {
	chunkChan := make(chan *interfaces.LLMStreamChunk, 100)
	
	go func() {
		defer close(chunkChan)
		
		// For simplicity, simulate streaming by calling regular API and chunking response
		response, err := p.Call(ctx, messages, config)
		if err != nil {
			chunkChan <- &interfaces.LLMStreamChunk{
				Error: err,
			}
			return
		}
		
		// Simulate streaming by sending chunks
		chunkSize := 50
		content := response.Content
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
			
			// Simulate streaming delay
			time.Sleep(10 * time.Millisecond)
		}
	}()
	
	return chunkChan, nil
}

// CountTokens counts the number of tokens in messages
func (p *SimpleLLMProvider) CountTokens(messages []models.Message) (int, error) {
	// Simple estimation: ~4 characters per token
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Role) + len(msg.Content)
		if msg.ReasoningContent != "" {
			totalChars += len(msg.ReasoningContent)
		}
	}
	return totalChars / 4, nil
}

// Private methods

func (p *SimpleLLMProvider) mergeConfig(config *models.LLMConfig) *models.LLMConfig {
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

func (p *SimpleLLMProvider) buildRequest(messages []models.Message, config *models.LLMConfig) map[string]interface{} {
	// Build OpenAI-compatible request format
	request := map[string]interface{}{
		"model":       config.Model,
		"messages":    p.convertMessages(messages),
		"temperature": config.Temperature,
	}
	
	if config.MaxTokens > 0 {
		request["max_tokens"] = config.MaxTokens
	}
	
	return request
}

func (p *SimpleLLMProvider) convertMessages(messages []models.Message) []map[string]interface{} {
	converted := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		converted[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return converted
}

func (p *SimpleLLMProvider) makeAPICall(ctx context.Context, requestBody map[string]interface{}, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	// Prepare request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	url := p.getAPIURL(config)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	}
	
	// Make request
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
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Extract content (OpenAI format)
	content := ""
	if choices, ok := apiResponse["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if c, ok := msg["content"].(string); ok {
					content = c
				}
			}
		}
	}
	
	// Extract usage
	usage := interfaces.TokenUsage{}
	if u, ok := apiResponse["usage"].(map[string]interface{}); ok {
		if pt, ok := u["prompt_tokens"].(float64); ok {
			usage.PromptTokens = int(pt)
		}
		if ct, ok := u["completion_tokens"].(float64); ok {
			usage.CompletionTokens = int(ct)
		}
		if tt, ok := u["total_tokens"].(float64); ok {
			usage.TotalTokens = int(tt)
		}
	}
	
	return &interfaces.LLMResponse{
		Content: content,
		Model:   config.Model,
		Usage:   usage,
		Raw:     apiResponse,
	}, nil
}

func (p *SimpleLLMProvider) getAPIURL(config *models.LLMConfig) string {
	baseURL := config.BaseURL
	if baseURL == "" {
		// Default to OpenAI API
		baseURL = "https://api.openai.com/v1"
	}
	return fmt.Sprintf("%s/chat/completions", baseURL)
}

// MockLLMProvider implements a mock LLM provider for testing
type MockLLMProvider struct {
	responses []string
	index     int
}

// NewMockLLMProvider creates a new mock provider
func NewMockLLMProvider(responses []string) *MockLLMProvider {
	return &MockLLMProvider{
		responses: responses,
		index:     0,
	}
}

// Call returns a mock response
func (m *MockLLMProvider) Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	response := "This is a mock response."
	if m.index < len(m.responses) {
		response = m.responses[m.index]
		m.index++
	}
	
	// Extract user query for more realistic response
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			response = fmt.Sprintf("Mock response to: %s\n\n%s", messages[i].Content, response)
			break
		}
	}
	
	return &interfaces.LLMResponse{
		Content: response,
		Model:   "mock-model",
		Usage: interfaces.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

// StreamCall returns a mock streaming response
func (m *MockLLMProvider) StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *interfaces.LLMStreamChunk, error) {
	chunkChan := make(chan *interfaces.LLMStreamChunk, 10)
	
	go func() {
		defer close(chunkChan)
		
		response, _ := m.Call(ctx, messages, config)
		chunks := []string{
			"Mock ", "streaming ", "response: ", response.Content,
		}
		
		for i, chunk := range chunks {
			select {
			case chunkChan <- &interfaces.LLMStreamChunk{
				Delta:  chunk,
				Finish: i == len(chunks)-1,
			}:
			case <-ctx.Done():
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	
	return chunkChan, nil
}

// CountTokens returns a mock token count
func (m *MockLLMProvider) CountTokens(messages []models.Message) (int, error) {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content) / 4 // Simple estimation
	}
	return total, nil
}