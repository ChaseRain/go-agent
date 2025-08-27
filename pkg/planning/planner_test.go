package planning

import (
	"context"
	"testing"

	"go-agent/pkg/llm"
	"go-agent/pkg/models"
	"go-agent/pkg/record"
)

func TestTaskPlanner_NeedsPlan(t *testing.T) {
	planner := &TaskPlanner{}
	
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "Complex task with analyze",
			message:  "Analyze the sales data and generate a report",
			expected: true,
		},
		{
			name:     "Multi-step task",
			message:  "First download the file, then process it, finally upload results",
			expected: true,
		},
		{
			name:     "Simple question",
			message:  "What is the capital of France?",
			expected: false,
		},
		{
			name:     "Definition query",
			message:  "Define machine learning",
			expected: false,
		},
		{
			name:     "Research task",
			message:  "Research the latest trends in AI",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := planner.NeedsPlan(tt.message)
			if result != tt.expected {
				t.Errorf("NeedsPlan(%s) = %v, want %v", tt.message, result, tt.expected)
			}
		})
	}
}

func TestTaskPlanner_OptimizePlan(t *testing.T) {
	planner := &TaskPlanner{}
	
	// Test removing empty tasks
	tasks := []models.SubTask{
		{ID: "1", Description: "Valid task 1"},
		{ID: "", Description: ""},
		{ID: "3", Description: "Valid task 2"},
	}
	
	optimized := planner.OptimizePlan(tasks)
	
	if len(optimized) != 2 {
		t.Errorf("Expected 2 tasks after optimization, got %d", len(optimized))
	}
	
	// Test removing duplicates
	tasks = []models.SubTask{
		{ID: "1", Description: "Task A"},
		{ID: "2", Description: "Task A"},
		{ID: "3", Description: "Task B"},
	}
	
	optimized = planner.OptimizePlan(tasks)
	
	if len(optimized) != 2 {
		t.Errorf("Expected 2 unique tasks, got %d", len(optimized))
	}
}

func TestTaskPlanner_Plan(t *testing.T) {
	// Create mock LLM provider
	mockLLM := llm.NewMockLLMProvider([]string{
		`{
			"tasks": [
				{
					"sub_task_name": "Fetch data",
					"sub_task_describe": "Retrieve data from database",
					"process": "Query the database",
					"sub_task_type": "task",
					"dependent": ""
				},
				{
					"sub_task_name": "Process data",
					"sub_task_describe": "Clean and transform data",
					"process": "Apply data transformations",
					"sub_task_type": "task",
					"dependent": "task_0"
				}
			],
			"summary": "Data processing pipeline"
		}`,
	})
	
	// Create record manager
	recordManager := record.NewRecordManager("./test_records", "test_session")
	defer func() {
		// Clean up test records
		// In production, would properly clean up
	}()
	
	// Create config
	config := &models.AgentConfig{
		Name:     "TestAgent",
		MaxSteps: []int{5, 3},
		LLMConfig: models.LLMConfig{
			Model: "test-model",
		},
	}
	
	// Create planner
	planner := NewTaskPlanner(mockLLM, config, recordManager)
	
	// Create context
	ctx := context.Background()
	execContext := &models.ExecutionContext{
		AgentName: "TestAgent",
		Config:    config,
		Messages:  []models.Message{},
	}
	
	// Test planning
	result, err := planner.Plan(ctx, "Process the data and generate a report", execContext)
	
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	
	if len(result.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(result.Tasks))
	}
	
	if result.Summary != "Data processing pipeline" {
		t.Errorf("Expected summary 'Data processing pipeline', got %s", result.Summary)
	}
}

func BenchmarkTaskPlanner_NeedsPlan(b *testing.B) {
	planner := &TaskPlanner{}
	message := "Analyze the sales data and generate a comprehensive report with visualizations"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		planner.NeedsPlan(message)
	}
}

func BenchmarkTaskPlanner_OptimizePlan(b *testing.B) {
	planner := &TaskPlanner{}
	tasks := []models.SubTask{
		{ID: "1", Description: "Task 1"},
		{ID: "2", Description: "Task 2"},
		{ID: "3", Description: "Task 3"},
		{ID: "4", Description: "Task 4"},
		{ID: "5", Description: "Task 5"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		planner.OptimizePlan(tasks)
	}
}