package interfaces

import (
	"context"
	"go-agent/pkg/models"
)

// Agent defines the main agent interface
type Agent interface {
	// Initialize the agent with configuration
	Initialize(config *models.AgentConfig) error
	
	// Process a user message and return result
	ProcessMessage(ctx context.Context, message string) (*models.ProcessMessageResult, error)
	
	// Get agent ID
	GetID() string
	
	// Get agent name
	GetName() string
	
	// Get current state
	GetState() AgentState
	
	// Restore from previous state
	Restore(sessionID string) error
}

// TaskPlanner defines the interface for task planning
type TaskPlanner interface {
	// Plan tasks based on the input message and context
	Plan(ctx context.Context, message string, context *models.ExecutionContext) (*models.PlanningResult, error)
	
	// Validate if planning is needed
	NeedsPlan(message string) bool
	
	// Clean and optimize the planning list
	OptimizePlan(tasks []models.SubTask) []models.SubTask
}

// TaskExecutor defines the interface for task execution
type TaskExecutor interface {
	// Execute a single task
	ExecuteTask(ctx context.Context, task *models.SubTask, context *models.ExecutionContext) error
	
	// Execute multiple tasks (parallel or serial based on dependencies)
	ExecuteBatch(ctx context.Context, tasks []models.SubTask, context *models.ExecutionContext) error
	
	// Check if tasks can be executed in parallel
	CanParallelize(tasks []models.SubTask) bool
}

// RecordManager defines the interface for record management
type RecordManager interface {
	// Record an event
	Record(recordType RecordType, data interface{}) (string, error)
	
	// Get record by ID
	GetRecord(recordID string) (interface{}, error)
	
	// Get all records for a session
	GetSessionRecords(sessionID string) ([]interface{}, error)
	
	// Save session state
	SaveSession(sessionID string, data interface{}) error
	
	// Load session state
	LoadSession(sessionID string) (interface{}, error)
}

// MessageManager defines the interface for message management
type MessageManager interface {
	// Add a message to history
	AddMessage(message models.Message) error
	
	// Get message history
	GetHistory() []models.Message
	
	// Clear message history
	ClearHistory() error
	
	// Get messages for context (with truncation if needed)
	GetContextMessages(maxTokens int) []models.Message
}

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// Call the LLM with messages
	Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*LLMResponse, error)
	
	// Stream call to the LLM
	StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *LLMStreamChunk, error)
	
	// Count tokens in messages
	CountTokens(messages []models.Message) (int, error)
}

// Tool defines the interface for tools/functions
type Tool interface {
	// Get tool name
	GetName() string
	
	// Get tool description
	GetDescription() string
	
	// Execute the tool with arguments
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	
	// Validate arguments
	ValidateArgs(args map[string]interface{}) error
}

// ResultProcessor defines the interface for result processing
type ResultProcessor interface {
	// Process and format results
	ProcessResults(results []interface{}, format OutputFormat) (interface{}, error)
	
	// Generate summary from results
	GenerateSummary(results []interface{}) (string, error)
	
	// Save results to file
	SaveToFile(results interface{}, filepath string) error
}

// StateManager defines the interface for state management
type StateManager interface {
	// Save agent state
	SaveState(agentID string, state interface{}) error
	
	// Load agent state
	LoadState(agentID string) (interface{}, error)
	
	// Delete agent state
	DeleteState(agentID string) error
	
	// List all states
	ListStates() ([]string, error)
}

// Types and Enums

// AgentState represents the current state of an agent
type AgentState string

const (
	AgentStateIdle     AgentState = "idle"
	AgentStatePlanning AgentState = "planning"
	AgentStateExecuting AgentState = "executing"
	AgentStateCompleted AgentState = "completed"
	AgentStateError     AgentState = "error"
)

// RecordType represents different types of records
type RecordType string

const (
	RecordTypeAgentExecution RecordType = "agent_execution"
	RecordTypeLLMCall        RecordType = "llm_call"
	RecordTypePlanning       RecordType = "planning"
	RecordTypeSubtask        RecordType = "subtask_execution"
	RecordTypeFunctionCall   RecordType = "function_call"
	RecordTypeError          RecordType = "error"
)

// OutputFormat represents different output formats
type OutputFormat string

const (
	OutputFormatMarkdown OutputFormat = "markdown"
	OutputFormatJSON     OutputFormat = "json"
	OutputFormatText     OutputFormat = "text"
	OutputFormatHTML     OutputFormat = "html"
)

// LLMResponse represents a response from an LLM
type LLMResponse struct {
	Content string
	Model   string
	Usage   TokenUsage
	Raw     interface{}
}

// LLMStreamChunk represents a chunk in streaming response
type LLMStreamChunk struct {
	Delta   string
	Finish  bool
	Error   error
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}