package llm

import (
	"fmt"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// CreateProvider creates an LLM provider based on the configuration
func CreateProvider(config *models.LLMConfig) (interfaces.LLMProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	switch config.Provider {
	case "openai":
		return NewOpenAIProvider(config), nil
	
	case "deepseek":
		return NewDeepSeekProvider(config), nil
	
	case "deepseek-v3", "modelarts":
		// Use ModelArts provider for DeepSeek-V3 on ModelArts MAAS
		return NewModelArtsProvider(config), nil
	
	case "mock":
		return NewMockLLMProvider([]string{
			"This is a mock response for testing.",
			"Another mock response with different content.",
		}), nil
	
	case "simple":
		return NewSimpleLLMProvider(config), nil
	
	default:
		// Default to OpenAI provider
		return NewOpenAIProvider(config), nil
	}
}

// ValidateProviderConfig validates the LLM provider configuration
func ValidateProviderConfig(config *models.LLMConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Check required fields for non-mock providers
	if config.Provider != "mock" {
		if config.APIKey == "" {
			return fmt.Errorf("API key is required for provider %s", config.Provider)
		}
	}

	// Validate provider-specific requirements
	switch config.Provider {
	case "deepseek-v3":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.modelarts-maas.com/v1"
		}
		if config.Model == "" {
			config.Model = "DeepSeek-V3"
		}
	
	case "deepseek":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.deepseek.com/v1"
		}
		if config.Model == "" {
			config.Model = "deepseek-chat"
		}
	
	case "openai":
		if config.BaseURL == "" {
			config.BaseURL = "https://api.openai.com/v1"
		}
		if config.Model == "" {
			config.Model = "gpt-3.5-turbo"
		}
	}

	return nil
}

// NewProvider is a convenience function that creates a provider
func NewProvider(providerType string, config *models.LLMConfig) (interfaces.LLMProvider, error) {
	if config == nil {
		config = &models.LLMConfig{}
	}
	config.Provider = providerType
	return CreateProvider(config)
}