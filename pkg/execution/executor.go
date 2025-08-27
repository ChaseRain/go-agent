package execution

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// TaskExecutor implements task execution logic
type TaskExecutor struct {
	llmProvider   interfaces.LLMProvider
	recordManager interfaces.RecordManager
	tools         map[string]interfaces.Tool
	mu            sync.RWMutex
	parallel      bool
	maxWorkers    int
}

// NewTaskExecutor creates a new TaskExecutor instance
func NewTaskExecutor(
	llmProvider interfaces.LLMProvider,
	recordManager interfaces.RecordManager,
	config *models.AgentConfig,
) *TaskExecutor {
	maxWorkers := 5
	if workers, ok := config.CustomConfig["max_workers"].(int); ok {
		maxWorkers = workers
	}

	return &TaskExecutor{
		llmProvider:   llmProvider,
		recordManager: recordManager,
		tools:         make(map[string]interfaces.Tool),
		parallel:      config.Parallel,
		maxWorkers:    maxWorkers,
	}
}

// RegisterTool registers a tool for execution
func (e *TaskExecutor) RegisterTool(tool interfaces.Tool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tools[tool.GetName()] = tool
}

// ExecuteTask executes a single task
func (e *TaskExecutor) ExecuteTask(ctx context.Context, task *models.SubTask, execContext *models.ExecutionContext) error {
	// Update task state to running
	task.State = models.TaskStateRunning
	task.UpdatedAt = time.Now()

	// Record task execution start
	recordID, err := e.recordManager.Record(
		interfaces.RecordTypeSubtask,
		map[string]interface{}{
			"parent_id":   execContext.ParentRecordID,
			"task_id":     task.ID,
			"task_name":   task.Name,
			"task_type":   task.Type,
			"status":      "started",
			"agent_chain": execContext.AgentChain,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to record task start: %w", err)
	}
	task.RecordID = recordID

	// Execute based on task type
	var execErr error
	switch models.SubTaskType(task.Type) {
	case models.SubTaskTypeFunction:
		execErr = e.executeFunctionTask(ctx, task, execContext)
	case models.SubTaskTypeAgentCall:
		execErr = e.executeAgentCallTask(ctx, task, execContext)
	case models.SubTaskTypeAgentGen:
		execErr = e.executeAgentGenTask(ctx, task, execContext)
	default:
		execErr = e.executeNormalTask(ctx, task, execContext)
	}

	// Update task state based on execution result
	if execErr != nil {
		task.State = models.TaskStateFail
		task.StateMsg = execErr.Error()

		// Record error
		e.recordManager.Record(
			interfaces.RecordTypeError,
			map[string]interface{}{
				"parent_id": recordID,
				"task_id":   task.ID,
				"error":     execErr.Error(),
			},
		)
	} else {
		task.State = models.TaskStateSuccess
	}
	task.UpdatedAt = time.Now()

	// Record task completion
	e.recordManager.Record(
		interfaces.RecordTypeSubtask,
		map[string]interface{}{
			"parent_id": recordID,
			"task_id":   task.ID,
			"status":    string(task.State),
			"state_msg": task.StateMsg,
		},
	)

	return execErr
}

// ExecuteBatch executes multiple tasks with dependency management
func (e *TaskExecutor) ExecuteBatch(ctx context.Context, tasks []models.SubTask, execContext *models.ExecutionContext) error {
	if len(tasks) == 0 {
		return nil
	}

	// Build task map for quick lookup
	taskMap := make(map[string]*models.SubTask)
	for i := range tasks {
		taskMap[tasks[i].ID] = &tasks[i]
	}

	// Analyze dependencies and group tasks
	taskGroups := e.analyzeTaskDependencies(tasks)

	// Execute task groups in order
	for _, group := range taskGroups {
		if e.parallel && e.CanParallelize(group) {
			// Execute tasks in parallel
			if err := e.executeParallel(ctx, group, execContext); err != nil {
				return fmt.Errorf("parallel execution failed: %w", err)
			}
		} else {
			// Execute tasks sequentially
			for i := range group {
				if err := e.ExecuteTask(ctx, &group[i], execContext); err != nil {
					// Continue with other tasks even if one fails
					fmt.Printf("Task %s failed: %v\n", group[i].ID, err)
				}
			}
		}
	}

	return nil
}

// CanParallelize checks if tasks can be executed in parallel
func (e *TaskExecutor) CanParallelize(tasks []models.SubTask) bool {
	// Check if any task has dependencies within this group
	taskIDs := make(map[string]bool)
	for _, task := range tasks {
		taskIDs[task.ID] = true
	}

	for _, task := range tasks {
		if task.Dependent != "" && taskIDs[task.Dependent] {
			// Has dependency within group, cannot parallelize
			return false
		}
	}

	// Check task types - some types shouldn't be parallelized
	for _, task := range tasks {
		if task.Type == string(models.SubTaskTypeAgentGen) {
			// Agent generation tasks should be sequential
			return false
		}
	}

	return true
}

// Private methods

func (e *TaskExecutor) executeNormalTask(ctx context.Context, task *models.SubTask, execContext *models.ExecutionContext) error {
	// Build prompt for task execution
	prompt := fmt.Sprintf(`Execute the following task:
Task: %s
Description: %s
Process: %s

Context:
Agent: %s
Session: %s

Please complete this task and provide the result.`,
		task.Name, task.Description, task.Process,
		execContext.AgentName, execContext.SessionID)

	// Prepare messages for LLM
	messages := append(execContext.Messages, models.Message{
		Role:    "user",
		Content: prompt,
	})

	// Call LLM
	response, err := e.llmProvider.Call(ctx, messages, &execContext.Config.LLMConfig)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Record LLM call
	e.recordManager.Record(
		interfaces.RecordTypeLLMCall,
		map[string]interface{}{
			"parent_id": task.RecordID,
			"task_id":   task.ID,
			"prompt":    prompt,
			"response":  response.Content,
			"model":     response.Model,
			"tokens":    response.Usage,
		},
	)

	// Save result to output file
	task.OutputFile = fmt.Sprintf("%s_%s.md", task.Name, time.Now().Format("20060102_150405"))
	// In real implementation, would save to actual file

	return nil
}

func (e *TaskExecutor) executeFunctionTask(ctx context.Context, task *models.SubTask, execContext *models.ExecutionContext) error {
	// Parse function name and arguments from task process
	functionName, args := e.parseFunctionCall(task.Process)

	// Get tool
	e.mu.RLock()
	tool, exists := e.tools[functionName]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("function %s not found", functionName)
	}

	// Validate arguments
	if err := tool.ValidateArgs(args); err != nil {
		return fmt.Errorf("invalid arguments for %s: %w", functionName, err)
	}

	// Execute tool
	result, err := tool.Execute(ctx, args)
	if err != nil {
		return fmt.Errorf("function execution failed: %w", err)
	}

	// Record function call
	e.recordManager.Record(
		interfaces.RecordTypeFunctionCall,
		map[string]interface{}{
			"parent_id": task.RecordID,
			"task_id":   task.ID,
			"function":  functionName,
			"args":      args,
			"result":    result,
		},
	)

	// Store result
	task.OutputFile = fmt.Sprintf("function_%s_%s.json", functionName, time.Now().Format("20060102_150405"))

	return nil
}

func (e *TaskExecutor) executeAgentCallTask(ctx context.Context, task *models.SubTask, execContext *models.ExecutionContext) error {
	// Parse agent name and task from process
	agentName, agentTask := e.parseAgentCall(task.Process)

	// In real implementation, would create and call sub-agent
	// For now, simulate with LLM call
	prompt := fmt.Sprintf(`Acting as agent '%s', execute: %s`, agentName, agentTask)

	messages := []models.Message{
		{Role: "system", Content: fmt.Sprintf("You are agent: %s", agentName)},
		{Role: "user", Content: prompt},
	}

	response, err := e.llmProvider.Call(ctx, messages, &execContext.Config.LLMConfig)
	if err != nil {
		return fmt.Errorf("agent call failed: %w", err)
	}

	// Record agent call
	e.recordManager.Record(
		interfaces.RecordTypeSubtask,
		map[string]interface{}{
			"parent_id":  task.RecordID,
			"task_id":    task.ID,
			"agent_name": agentName,
			"agent_task": agentTask,
			"result":     response.Content,
		},
	)

	return nil
}

func (e *TaskExecutor) executeAgentGenTask(ctx context.Context, task *models.SubTask, execContext *models.ExecutionContext) error {
	// Agent generation creates a new agent instance
	// In real implementation, would instantiate new agent
	// For now, simulate with enhanced context

	newAgentChain := append(execContext.AgentChain, task.Name)
	newContext := *execContext
	newContext.AgentChain = newAgentChain

	// Execute as sub-agent
	return e.executeNormalTask(ctx, task, &newContext)
}

func (e *TaskExecutor) executeParallel(ctx context.Context, tasks []models.SubTask, execContext *models.ExecutionContext) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))

	// Create worker pool
	semaphore := make(chan struct{}, e.maxWorkers)

	for i := range tasks {
		wg.Add(1)
		go func(task *models.SubTask) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute task
			if err := e.ExecuteTask(ctx, task, execContext); err != nil {
				errChan <- fmt.Errorf("task %s failed: %w", task.ID, err)
			}
		}(&tasks[i])
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("parallel execution had %d errors: %v", len(errs), errs[0])
	}

	return nil
}

func (e *TaskExecutor) analyzeTaskDependencies(tasks []models.SubTask) [][]models.SubTask {
	// Build dependency graph
	dependencyMap := make(map[string][]string)
	taskMap := make(map[string]*models.SubTask)

	for i := range tasks {
		task := &tasks[i]
		taskMap[task.ID] = task
		if task.Dependent != "" {
			dependencyMap[task.ID] = []string{task.Dependent}
		}
	}

	// Topological sort to determine execution order
	var groups [][]models.SubTask
	processed := make(map[string]bool)

	for len(processed) < len(tasks) {
		var currentGroup []models.SubTask

		for _, task := range tasks {
			if processed[task.ID] {
				continue
			}

			// Check if all dependencies are processed
			canExecute := true
			if deps, hasDeps := dependencyMap[task.ID]; hasDeps {
				for _, dep := range deps {
					if !processed[dep] {
						canExecute = false
						break
					}
				}
			}

			if canExecute {
				currentGroup = append(currentGroup, task)
			}
		}

		// Mark current group as processed
		for _, task := range currentGroup {
			processed[task.ID] = true
		}

		if len(currentGroup) > 0 {
			groups = append(groups, currentGroup)
		} else {
			// Circular dependency detected
			break
		}
	}

	return groups
}

func (e *TaskExecutor) parseFunctionCall(process string) (string, map[string]interface{}) {
	// 解析函数调用格式: <function_call>functionName(arg1=value1, arg2=value2)</function_call>
	// 或者简化格式: calculator(operation=basic, operator=+, a=15, b=27)

	// 查找函数调用标签
	start := strings.Index(process, "<function_call>")
	end := strings.Index(process, "</function_call>")

	var funcCall string
	if start != -1 && end != -1 {
		funcCall = process[start+15 : end] // 15 = len("<function_call>")
	} else {
		// 尝试直接解析，假设整个process就是函数调用
		funcCall = strings.TrimSpace(process)
	}

	// 解析函数名和参数
	parenIndex := strings.Index(funcCall, "(")
	if parenIndex == -1 {
		// 没有参数的函数调用
		return strings.TrimSpace(funcCall), map[string]interface{}{}
	}

	functionName := strings.TrimSpace(funcCall[:parenIndex])
	argsStr := funcCall[parenIndex+1:]

	// 移除末尾的 )
	if strings.HasSuffix(argsStr, ")") {
		argsStr = argsStr[:len(argsStr)-1]
	}

	// 解析参数
	args := make(map[string]interface{})
	if argsStr != "" {
		// 简单的参数解析：arg1=value1, arg2=value2
		pairs := strings.Split(argsStr, ",")
		for _, pair := range pairs {
			pair = strings.TrimSpace(pair)
			if strings.Contains(pair, "=") {
				parts := strings.SplitN(pair, "=", 2)
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// 尝试转换数值
				if intVal, err := strconv.Atoi(value); err == nil {
					args[key] = intVal
				} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					args[key] = floatVal
				} else {
					// 移除引号
					value = strings.Trim(value, "\"'")
					args[key] = value
				}
			}
		}
	}

	return functionName, args
}

func (e *TaskExecutor) parseAgentCall(process string) (string, string) {
	// Simple parsing - in real implementation would be more robust
	// Format: <agent_call>agentName: task description</agent_call>

	// For now, return mock data
	return "ResearchAgent", "Research the topic"
}
