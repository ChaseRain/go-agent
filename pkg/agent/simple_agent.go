package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// AgentWithProvider is a simple agent wrapper that includes an LLM provider
type AgentWithProvider struct {
	agent       *DynAgent
	provider    interfaces.LLMProvider
	initialized bool
}

// NewAgentWithProvider creates a new agent with provider
func NewAgentWithProvider(provider interfaces.LLMProvider) *AgentWithProvider {
	return &AgentWithProvider{
		provider: provider,
	}
}

// Initialize initializes the agent with configuration
func (a *AgentWithProvider) Initialize(config *models.AgentConfig) error {
	if a.initialized {
		return fmt.Errorf("agent already initialized")
	}

	// Create the actual DynAgent
	a.agent = NewDynAgent(config)
	a.initialized = true
	return nil
}

// ProcessMessage processes a user message
func (a *AgentWithProvider) ProcessMessage(ctx context.Context, message string) (*models.ProcessMessageResult, error) {
	if !a.initialized {
		return nil, fmt.Errorf("agent not initialized")
	}

	// For simplicity, just use a basic response
	// In a real implementation, this would use the planner and executor
	response := fmt.Sprintf("Processed: %s", message)
	
	return &models.ProcessMessageResult{
		Code:    0,
		Message: response,
	}, nil
}

// GetID returns the agent ID
func (a *AgentWithProvider) GetID() string {
	if a.agent != nil {
		return a.agent.GetID()
	}
	return uuid.New().String()
}

// GetName returns the agent name
func (a *AgentWithProvider) GetName() string {
	if a.agent != nil {
		return a.agent.GetName()
	}
	return "SimpleAgent"
}

// GetState returns the agent state
func (a *AgentWithProvider) GetState() interfaces.AgentState {
	if a.agent != nil {
		return a.agent.GetState()
	}
	return interfaces.AgentStateIdle
}