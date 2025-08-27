package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// DynAgent is the main agent implementation
type DynAgent struct {
	// Basic info
	id       string
	name     string
	config   *models.AgentConfig
	state    interfaces.AgentState
	mu       sync.RWMutex

	// Core components
	planner        interfaces.TaskPlanner
	executor       interfaces.TaskExecutor
	recordManager  interfaces.RecordManager
	messageManager interfaces.MessageManager
	stateManager   interfaces.StateManager
	resultProcessor interfaces.ResultProcessor
	llmProvider    interfaces.LLMProvider

	// Runtime state
	sessionID      string
	agentChain     []string
	parentRecordID string
	currentContext *models.ExecutionContext
	
	// Tools registry
	tools          map[string]interfaces.Tool
	toolsMu        sync.RWMutex
}

// NewDynAgent creates a new DynAgent instance
func NewDynAgent(config *models.AgentConfig) *DynAgent {
	agentID := uuid.New().String()
	
	agent := &DynAgent{
		id:         agentID,
		name:       config.Name,
		config:     config,
		state:      interfaces.AgentStateIdle,
		tools:      make(map[string]interfaces.Tool),
		agentChain: []string{config.Name},
		sessionID:  fmt.Sprintf("session_%d", time.Now().Unix()),
	}
	
	// Initialize components (will be implemented in separate files)
	agent.initializeComponents()
	
	return agent
}

// Initialize initializes the agent with configuration
func (a *DynAgent) Initialize(config *models.AgentConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.config = config
	a.name = config.Name
	
	// Re-initialize components with new config
	return a.initializeComponents()
}

// ProcessMessage processes a user message and returns result
func (a *DynAgent) ProcessMessage(ctx context.Context, message string) (*models.ProcessMessageResult, error) {
	a.mu.Lock()
	a.state = interfaces.AgentStatePlanning
	a.mu.Unlock()
	
	// Record agent execution start
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
	if err := a.messageManager.AddMessage(userMessage); err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}
	
	// Build execution context
	context := a.buildExecutionContext()
	context.ParentRecordID = executionRecordID
	
	// Planning phase
	var planResult *models.PlanningResult
	if a.planner.NeedsPlan(message) {
		planResult, err = a.planner.Plan(ctx, message, context)
		if err != nil {
			a.recordError(executionRecordID, "planning", err)
			return nil, fmt.Errorf("planning failed: %w", err)
		}
		
		// Record planning result
		_, err = a.recordManager.Record(
			interfaces.RecordTypePlanning,
			map[string]interface{}{
				"parent_id": executionRecordID,
				"plan":      planResult,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to record planning: %w", err)
		}
	} else {
		// Simple response without planning
		planResult = &models.PlanningResult{
			Tasks: []models.SubTask{
				{
					ID:          uuid.New().String(),
					Name:        "direct_response",
					Description: "Directly respond to the user query",
					Type:        string(models.SubTaskTypeTask),
					State:       models.TaskStateWait,
				},
			},
		}
	}
	
	// Execution phase
	a.mu.Lock()
	a.state = interfaces.AgentStateExecuting
	a.mu.Unlock()
	
	// Execute tasks
	if err := a.executor.ExecuteBatch(ctx, planResult.Tasks, context); err != nil {
		a.recordError(executionRecordID, "execution", err)
		return nil, fmt.Errorf("execution failed: %w", err)
	}
	
	// Process results
	results := a.collectTaskResults(planResult.Tasks)
	finalResult, err := a.resultProcessor.ProcessResults(results, interfaces.OutputFormatMarkdown)
	if err != nil {
		return nil, fmt.Errorf("result processing failed: %w", err)
	}
	
	// Generate summary
	summary, err := a.resultProcessor.GenerateSummary(results)
	if err != nil {
		summary = "Task completed successfully"
	}
	
	// Build final result
	result := &models.ProcessMessageResult{
		Code:               0,
		Message:            "success",
		OutputFile:         fmt.Sprintf("%s_%d.md", a.name, time.Now().Unix()),
		OutputTextAbstract: summary,
	}
	
	// Save result to file if needed
	if a.config.CustomConfig["save_output"] == true {
		outputPath := fmt.Sprintf("output/%s", result.OutputFile)
		if err := a.resultProcessor.SaveToFile(finalResult, outputPath); err != nil {
			// Log error but don't fail the whole process
			fmt.Printf("Warning: failed to save output file: %v\n", err)
		}
		result.AllFiles = append(result.AllFiles, outputPath)
	}
	
	// Record completion
	_, err = a.recordManager.Record(
		interfaces.RecordTypeAgentExecution,
		map[string]interface{}{
			"agent_id":   a.id,
			"agent_name": a.name,
			"status":     "completed",
			"result":     result,
			"parent_id":  executionRecordID,
		},
	)
	
	// Update state
	a.mu.Lock()
	a.state = interfaces.AgentStateCompleted
	a.mu.Unlock()
	
	return result, nil
}

// GetID returns the agent ID
func (a *DynAgent) GetID() string {
	return a.id
}

// GetName returns the agent name
func (a *DynAgent) GetName() string {
	return a.name
}

// GetState returns the current agent state
func (a *DynAgent) GetState() interfaces.AgentState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// Restore restores agent from previous state
func (a *DynAgent) Restore(sessionID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.sessionID = sessionID
	
	// Load state from state manager
	state, err := a.stateManager.LoadState(a.id)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	
	// Restore components state
	// This would restore message history, planning state, etc.
	if state != nil {
		// Implementation depends on state structure
		// For now, just set the session ID
		a.sessionID = sessionID
	}
	
	return nil
}

// RegisterTool registers a tool with the agent
func (a *DynAgent) RegisterTool(tool interfaces.Tool) {
	a.toolsMu.Lock()
	defer a.toolsMu.Unlock()
	a.tools[tool.GetName()] = tool
}

// GetTool gets a tool by name
func (a *DynAgent) GetTool(name string) (interfaces.Tool, bool) {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()
	tool, ok := a.tools[name]
	return tool, ok
}

// Private methods

func (a *DynAgent) initializeComponents() error {
	// Initialize each component with default implementations if not set
	
	if a.recordManager == nil {
		// Use a simple in-memory record manager for now
		a.recordManager = &SimpleRecordManager{
			records: make(map[string]interface{}),
		}
	}
	
	if a.messageManager == nil {
		// Use a simple message manager
		a.messageManager = &SimpleMessageManager{
			messages: make([]models.Message, 0),
		}
	}
	
	if a.planner == nil {
		// Use a simple planner
		a.planner = &SimplePlanner{}
	}
	
	if a.executor == nil {
		// Use a simple executor
		a.executor = &SimpleExecutor{
			tools: a.tools,
		}
	}
	
	if a.resultProcessor == nil {
		// Use a simple result processor
		a.resultProcessor = &SimpleResultProcessor{}
	}
	
	if a.stateManager == nil {
		// Use a simple state manager
		a.stateManager = &SimpleStateManager{
			states: make(map[string]interface{}),
		}
	}
	
	if a.llmProvider == nil {
		// Use mock provider as fallback
		a.llmProvider = &MockProvider{}
	}
	
	return nil
}

func (a *DynAgent) buildExecutionContext() *models.ExecutionContext {
	return &models.ExecutionContext{
		AgentID:      a.id,
		AgentName:    a.name,
		AgentChain:   a.agentChain,
		SessionID:    a.sessionID,
		ParentRecordID: a.parentRecordID,
		Messages:     a.messageManager.GetHistory(),
		Config:       a.config,
	}
}

func (a *DynAgent) collectTaskResults(tasks []models.SubTask) []interface{} {
	results := make([]interface{}, 0, len(tasks))
	for _, task := range tasks {
		// In real implementation, would collect actual results from task execution
		results = append(results, map[string]interface{}{
			"task_id": task.ID,
			"task_name": task.Name,
			"state": task.State,
			"output": task.OutputFile,
		})
	}
	return results
}

func (a *DynAgent) recordError(parentID string, phase string, err error) {
	a.recordManager.Record(
		interfaces.RecordTypeError,
		map[string]interface{}{
			"parent_id": parentID,
			"phase":     phase,
			"error":     err.Error(),
			"timestamp": time.Now(),
		},
	)
}