package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// SimpleRecordManager is a basic in-memory record manager
type SimpleRecordManager struct {
	records map[string]interface{}
}

func (r *SimpleRecordManager) Record(recordType interfaces.RecordType, data interface{}) (string, error) {
	id := uuid.New().String()
	r.records[id] = map[string]interface{}{
		"type": recordType,
		"data": data,
	}
	return id, nil
}

func (r *SimpleRecordManager) GetRecord(recordID string) (interface{}, error) {
	if record, ok := r.records[recordID]; ok {
		return record, nil
	}
	return nil, fmt.Errorf("record not found")
}

func (r *SimpleRecordManager) GetSessionRecords(sessionID string) ([]interface{}, error) {
	var results []interface{}
	for _, record := range r.records {
		results = append(results, record)
	}
	return results, nil
}

func (r *SimpleRecordManager) SaveSession(sessionID string, data interface{}) error {
	r.records[sessionID] = data
	return nil
}

func (r *SimpleRecordManager) LoadSession(sessionID string) (interface{}, error) {
	return r.records[sessionID], nil
}

// SimpleMessageManager is a basic message manager
type SimpleMessageManager struct {
	messages []models.Message
}

func (m *SimpleMessageManager) AddMessage(message models.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *SimpleMessageManager) GetHistory() []models.Message {
	return m.messages
}

func (m *SimpleMessageManager) ClearHistory() error {
	m.messages = make([]models.Message, 0)
	return nil
}

func (m *SimpleMessageManager) GetContextMessages(maxTokens int) []models.Message {
	// Return last few messages
	if len(m.messages) <= 5 {
		return m.messages
	}
	return m.messages[len(m.messages)-5:]
}

// SimplePlanner is a basic task planner
type SimplePlanner struct{}

func (p *SimplePlanner) Plan(ctx context.Context, message string, context *models.ExecutionContext) (*models.PlanningResult, error) {
	// Create a simple single-task plan
	task := models.SubTask{
		ID:          uuid.New().String(),
		Name:        "respond",
		Description: "Respond to user query",
		Process:     message,
		Type:        string(models.SubTaskTypeTask),
		State:       models.TaskStateWait,
	}
	
	return &models.PlanningResult{
		Tasks:   []models.SubTask{task},
		Summary: "Direct response",
	}, nil
}

func (p *SimplePlanner) NeedsPlan(message string) bool {
	// Simple heuristic - plan for complex messages
	return len(message) > 100
}

func (p *SimplePlanner) OptimizePlan(tasks []models.SubTask) []models.SubTask {
	return tasks
}

// SimpleExecutor is a basic task executor
type SimpleExecutor struct {
	tools map[string]interfaces.Tool
}

func (e *SimpleExecutor) ExecuteTask(ctx context.Context, task *models.SubTask, context *models.ExecutionContext) error {
	// Mark as success
	task.State = models.TaskStateSuccess
	return nil
}

func (e *SimpleExecutor) ExecuteBatch(ctx context.Context, tasks []models.SubTask, context *models.ExecutionContext) error {
	for i := range tasks {
		if err := e.ExecuteTask(ctx, &tasks[i], context); err != nil {
			return err
		}
	}
	return nil
}

func (e *SimpleExecutor) CanParallelize(tasks []models.SubTask) bool {
	return false
}

// SimpleResultProcessor is a basic result processor
type SimpleResultProcessor struct{}

func (r *SimpleResultProcessor) ProcessResults(results []interface{}, format interfaces.OutputFormat) (interface{}, error) {
	return map[string]interface{}{
		"results": results,
		"format":  format,
	}, nil
}

func (r *SimpleResultProcessor) GenerateSummary(results []interface{}) (string, error) {
	return fmt.Sprintf("Processed %d results", len(results)), nil
}

func (r *SimpleResultProcessor) SaveToFile(results interface{}, filepath string) error {
	// In real implementation, would save to file
	return nil
}

// SimpleStateManager is a basic state manager
type SimpleStateManager struct {
	states map[string]interface{}
}

func (s *SimpleStateManager) SaveState(agentID string, state interface{}) error {
	s.states[agentID] = state
	return nil
}

func (s *SimpleStateManager) LoadState(agentID string) (interface{}, error) {
	return s.states[agentID], nil
}

func (s *SimpleStateManager) DeleteState(agentID string) error {
	delete(s.states, agentID)
	return nil
}

func (s *SimpleStateManager) ListStates() ([]string, error) {
	var keys []string
	for k := range s.states {
		keys = append(keys, k)
	}
	return keys, nil
}

// MockProvider is a mock LLM provider
type MockProvider struct{}

func (m *MockProvider) Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	return &interfaces.LLMResponse{
		Content: "This is a mock response to demonstrate the agent framework.",
		Model:   "mock",
		Usage: interfaces.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

func (m *MockProvider) StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *interfaces.LLMStreamChunk, error) {
	ch := make(chan *interfaces.LLMStreamChunk, 1)
	go func() {
		ch <- &interfaces.LLMStreamChunk{
			Delta:  "Mock stream response",
			Finish: true,
		}
		close(ch)
	}()
	return ch, nil
}

func (m *MockProvider) CountTokens(messages []models.Message) (int, error) {
	return len(messages) * 10, nil
}