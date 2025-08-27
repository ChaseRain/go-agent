package planning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// TaskPlanner implements the task planning logic
type TaskPlanner struct {
	llmProvider    interfaces.LLMProvider
	maxSteps       []int
	currentDepth   int
	recordManager  interfaces.RecordManager
}

// NewTaskPlanner creates a new TaskPlanner instance
func NewTaskPlanner(llmProvider interfaces.LLMProvider, config *models.AgentConfig, recordManager interfaces.RecordManager) *TaskPlanner {
	return &TaskPlanner{
		llmProvider:   llmProvider,
		maxSteps:      config.MaxSteps,
		recordManager: recordManager,
		currentDepth:  0,
	}
}

// Plan creates a task plan based on the input message
func (p *TaskPlanner) Plan(ctx context.Context, message string, execContext *models.ExecutionContext) (*models.PlanningResult, error) {
	// Check if we've reached max planning depth
	if p.currentDepth >= len(p.maxSteps) {
		return nil, fmt.Errorf("max planning depth reached")
	}
	
	maxStepsForLevel := p.maxSteps[p.currentDepth]
	
	// Record planning start
	planRecordID, err := p.recordManager.Record(
		interfaces.RecordTypePlanning,
		map[string]interface{}{
			"parent_id": execContext.ParentRecordID,
			"depth":     p.currentDepth,
			"message":   message,
			"status":    "started",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record planning start: %w", err)
	}
	
	// Build planning prompt
	prompt := p.buildPlanningPrompt(message, execContext, maxStepsForLevel)
	
	// Call LLM for planning
	messages := []models.Message{
		{Role: "system", Content: p.getSystemPrompt()},
		{Role: "user", Content: prompt},
	}
	
	response, err := p.llmProvider.Call(ctx, messages, &execContext.Config.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse planning result
	planResult, err := p.parsePlanningResult(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse planning result: %w", err)
	}
	
	// Validate and optimize plan
	planResult.Tasks = p.OptimizePlan(planResult.Tasks)
	
	// Build dependency map
	planResult.Dependencies = p.buildDependencyMap(planResult.Tasks)
	
	// Record planning completion
	p.recordManager.Record(
		interfaces.RecordTypePlanning,
		map[string]interface{}{
			"parent_id": planRecordID,
			"depth":     p.currentDepth,
			"status":    "completed",
			"plan":      planResult,
		},
	)
	
	return planResult, nil
}

// NeedsPlan determines if planning is needed for the message
func (p *TaskPlanner) NeedsPlan(message string) bool {
	// Simple heuristic - can be made more sophisticated
	message = strings.ToLower(message)
	
	// Keywords that indicate complex tasks needing planning
	complexKeywords := []string{
		"analyze", "compare", "generate", "create", "build",
		"research", "investigate", "multiple", "steps",
		"first", "then", "finally", "process",
	}
	
	for _, keyword := range complexKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	
	// Simple queries don't need planning
	simpleKeywords := []string{
		"what is", "who is", "when", "where", "define",
		"tell me", "show me", "list",
	}
	
	for _, keyword := range simpleKeywords {
		if strings.HasPrefix(message, keyword) {
			return false
		}
	}
	
	// Default to planning for safety
	return true
}

// OptimizePlan cleans and optimizes the task list
func (p *TaskPlanner) OptimizePlan(tasks []models.SubTask) []models.SubTask {
	optimized := make([]models.SubTask, 0, len(tasks))
	
	for _, task := range tasks {
		// Skip empty or duplicate tasks
		if task.Description == "" {
			continue
		}
		
		// Check for duplicates
		isDuplicate := false
		for _, existing := range optimized {
			if existing.Description == task.Description {
				isDuplicate = true
				break
			}
		}
		
		if !isDuplicate {
			// Ensure task has required fields
			if task.ID == "" {
				task.ID = uuid.New().String()
			}
			if task.State == "" {
				task.State = models.TaskStateWait
			}
			if task.Type == "" {
				task.Type = p.inferTaskType(task.Process)
			}
			task.CreatedAt = time.Now()
			task.UpdatedAt = time.Now()
			
			optimized = append(optimized, task)
		}
	}
	
	return optimized
}

// Private methods

func (p *TaskPlanner) getSystemPrompt() string {
	return `You are an intelligent task planner. Your role is to break down complex user requests into actionable subtasks.

Rules:
1. Create clear, specific subtasks
2. Identify dependencies between tasks
3. Use appropriate task types (task, function, agent_call)
4. Keep the number of tasks reasonable and within limits
5. Ensure tasks are logically ordered

Output your plan in JSON format with the following structure:
{
  "tasks": [
    {
      "sub_task_name": "Task name",
      "sub_task_describe": "Detailed description",
      "process": "How to execute this task",
      "sub_task_type": "task|function|agent_call",
      "dependent": "ID of dependent task or empty string"
    }
  ],
  "summary": "Brief summary of the plan"
}`
}

func (p *TaskPlanner) buildPlanningPrompt(message string, context *models.ExecutionContext, maxSteps int) string {
	prompt := fmt.Sprintf(`User Request: %s

Context:
- Agent: %s
- Max steps for this level: %d
- Current depth: %d

Previous messages:
`, message, context.AgentName, maxSteps, p.currentDepth)
	
	// Add recent message history for context
	messages := context.Messages
	if len(messages) > 5 {
		messages = messages[len(messages)-5:]
	}
	
	for _, msg := range messages {
		prompt += fmt.Sprintf("- %s: %s\n", msg.Role, msg.Content)
	}
	
	prompt += fmt.Sprintf(`
Please create a task plan to address this request. 
Break it down into no more than %d subtasks.
Consider what tools or functions might be needed.
Identify any dependencies between tasks.`, maxSteps)
	
	return prompt
}

func (p *TaskPlanner) parsePlanningResult(content string) (*models.PlanningResult, error) {
	// Extract JSON from LLM response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	jsonStr := content[jsonStart : jsonEnd+1]
	
	// Parse JSON
	var rawResult struct {
		Tasks []struct {
			Name        string `json:"sub_task_name"`
			Description string `json:"sub_task_describe"`
			Process     string `json:"process"`
			Type        string `json:"sub_task_type"`
			Dependent   string `json:"dependent"`
		} `json:"tasks"`
		Summary string `json:"summary"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &rawResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Convert to models.SubTask
	tasks := make([]models.SubTask, len(rawResult.Tasks))
	for i, rawTask := range rawResult.Tasks {
		tasks[i] = models.SubTask{
			ID:          fmt.Sprintf("task_%d_%s", i, uuid.New().String()[:8]),
			Name:        rawTask.Name,
			Description: rawTask.Description,
			Process:     rawTask.Process,
			Type:        rawTask.Type,
			Dependent:   rawTask.Dependent,
			State:       models.TaskStateWait,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}
	
	return &models.PlanningResult{
		Tasks:   tasks,
		Summary: rawResult.Summary,
	}, nil
}

func (p *TaskPlanner) inferTaskType(process string) string {
	process = strings.ToLower(process)
	
	if strings.Contains(process, "<function_call>") || strings.Contains(process, "function:") {
		return string(models.SubTaskTypeFunction)
	}
	if strings.Contains(process, "<agent_call>") || strings.Contains(process, "agent:") {
		return string(models.SubTaskTypeAgentCall)
	}
	if strings.Contains(process, "<agent_gen>") {
		return string(models.SubTaskTypeAgentGen)
	}
	
	return string(models.SubTaskTypeTask)
}

func (p *TaskPlanner) buildDependencyMap(tasks []models.SubTask) map[string][]string {
	deps := make(map[string][]string)
	
	for _, task := range tasks {
		if task.Dependent != "" {
			// Task depends on another task
			if _, ok := deps[task.ID]; !ok {
				deps[task.ID] = []string{}
			}
			deps[task.ID] = append(deps[task.ID], task.Dependent)
		} else {
			// Task has no dependencies
			deps[task.ID] = []string{}
		}
	}
	
	return deps
}