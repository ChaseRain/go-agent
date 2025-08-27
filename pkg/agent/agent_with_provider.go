package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// NewDynAgentWithProvider creates a new DynAgent with explicit provider
func NewDynAgentWithProvider(config *models.AgentConfig, provider interfaces.LLMProvider, 
	planner interfaces.TaskPlanner, executor interfaces.TaskExecutor,
	recordManager interfaces.RecordManager, messageManager interfaces.MessageManager) *DynAgent {
	
	agentID := uuid.New().String()
	
	agent := &DynAgent{
		id:             agentID,
		name:           config.Name,
		config:         config,
		state:          interfaces.AgentStateIdle,
		tools:          make(map[string]interfaces.Tool),
		agentChain:     []string{config.Name},
		sessionID:      fmt.Sprintf("session_%d", time.Now().Unix()),
		llmProvider:    provider,
		planner:        planner,
		executor:       executor,
		recordManager:  recordManager,
		messageManager: messageManager,
	}
	
	return agent
}

// ProcessMessageWithLLM processes a message using the configured LLM
func (a *DynAgent) ProcessMessageWithLLM(ctx context.Context, message string) (*models.ProcessMessageResult, error) {
	// Record start
	executionRecordID, err := a.recordManager.Record(
		interfaces.RecordTypeAgentExecution,
		map[string]interface{}{
			"agent_id":   a.id,
			"agent_name": a.name,
			"message":    message,
			"status":     "started",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record execution start: %w", err)
	}
	
	// Add message to history
	userMessage := models.Message{
		Role:    "user",
		Content: message,
	}
	a.messageManager.AddMessage(userMessage)
	
	// Add system message if first message
	history := a.messageManager.GetHistory()
	if len(history) == 1 {
		systemMessage := models.Message{
			Role:    "system",
			Content: a.config.RoleDescription,
		}
		a.messageManager.AddMessage(systemMessage)
	}
	
	// Get messages for context
	messages := a.messageManager.GetContextMessages(a.config.LLMConfig.MaxTokens)
	
	// Call LLM directly
	response, err := a.llmProvider.Call(ctx, messages, &a.config.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Add assistant response to history
	assistantMessage := models.Message{
		Role:    "assistant", 
		Content: response.Content,
	}
	a.messageManager.AddMessage(assistantMessage)
	
	// Record completion
	a.recordManager.Record(
		interfaces.RecordTypeAgentExecution,
		map[string]interface{}{
			"agent_id":     a.id,
			"agent_name":   a.name,
			"execution_id": executionRecordID,
			"status":       "completed",
			"response":     response.Content,
		},
	)
	
	return &models.ProcessMessageResult{
		Code:               0,
		Message:            "Success",
		OutputTextAbstract: response.Content,
		OutputFile:         "",
		AllFiles:           []string{},
	}, nil
}