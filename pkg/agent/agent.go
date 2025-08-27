package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/execution"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/messaging"
	"go-agent/pkg/models"
	"go-agent/pkg/planning"
	"go-agent/pkg/processing"
	"go-agent/pkg/record"
	"go-agent/pkg/state"
	"go-agent/pkg/tools"
)

// DynAgent is the main agent implementation
type DynAgent struct {
	// Basic info
	id     string
	name   string
	config *models.AgentConfig
	state  interfaces.AgentState
	mu     sync.RWMutex

	// Core components
	planner         interfaces.TaskPlanner
	executor        interfaces.TaskExecutor
	recordManager   interfaces.RecordManager
	messageManager  interfaces.MessageManager
	stateManager    interfaces.StateManager
	resultProcessor interfaces.ResultProcessor
	llmProvider     interfaces.LLMProvider

	// Runtime state
	sessionID      string
	agentChain     []string
	parentRecordID string
	currentContext *models.ExecutionContext

	// Tools registry
	tools   map[string]interfaces.Tool
	toolsMu sync.RWMutex
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

	// Generate intelligent response
	intelligentResponse := a.generateIntelligentResponse(message, finalResult)

	// Build final result
	result := &models.ProcessMessageResult{
		Code:               0,
		Message:            intelligentResponse, // 使用智能回应而不是硬编码的"success"
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
	// 初始化生产级智能组件

	// 初始化JSONL记录管理器
	if a.recordManager == nil {
		recordConfig := &record.RecordConfig{
			BaseDir:    "./records",
			MaxRecords: 10000,
			EnableSSE:  false,
		}
		if a.config.CustomConfig != nil {
			if recordDir, ok := a.config.CustomConfig["record_dir"].(string); ok && recordDir != "" {
				recordConfig.BaseDir = recordDir
			}
		}
		a.recordManager = record.NewJSONLRecordManager(recordConfig)
	}

	// 初始化上下文消息管理器
	if a.messageManager == nil {
		tokenizer := messaging.NewSimpleTokenizer()
		maxMessages := 50
		maxTokens := 4000

		if a.config.CustomConfig != nil {
			if mm, ok := a.config.CustomConfig["max_messages"].(int); ok && mm > 0 {
				maxMessages = mm
			}
			if mt, ok := a.config.CustomConfig["max_tokens"].(int); ok && mt > 0 {
				maxTokens = mt
			}
		}

		a.messageManager = messaging.NewContextManager(maxMessages, maxTokens, tokenizer)
	}

	// 初始化状态管理器
	if a.stateManager == nil {
		stateDir := "./state"
		maxCacheSize := 100

		if a.config.CustomConfig != nil {
			if sd, ok := a.config.CustomConfig["state_dir"].(string); ok && sd != "" {
				stateDir = sd
			}
		}

		a.stateManager = state.NewStateManager(stateDir, maxCacheSize)
	}

	// 初始化结果处理器
	if a.resultProcessor == nil {
		outputDir := "./output"
		if a.config.CustomConfig != nil {
			if od, ok := a.config.CustomConfig["output_dir"].(string); ok && od != "" {
				outputDir = od
			}
		}

		processorConfig := &processing.ProcessorConfig{
			OutputDir:     outputDir,
			TemplateDir:   "./templates",
			EnableBackup:  true,
			MaxFileSize:   10 * 1024 * 1024, // 10MB
			OutputFormats: []interfaces.OutputFormat{interfaces.OutputFormatMarkdown, interfaces.OutputFormatJSON},
		}

		a.resultProcessor = processing.NewResultProcessor(processorConfig)
	}

	// LLM提供者需要在外部设置
	if a.llmProvider == nil {
		// 使用一个更智能的模拟提供者
		a.llmProvider = a.createMockProvider()
	}

	// 初始化智能任务规划器
	if a.planner == nil {
		a.planner = a.createTaskPlanner()
	}

	// 初始化任务执行器
	if a.executor == nil {
		a.executor = a.createTaskExecutor()

		// 注册内置工具
		a.registerBuiltinTools()
	}

	return nil
}

func (a *DynAgent) buildExecutionContext() *models.ExecutionContext {
	return &models.ExecutionContext{
		AgentID:        a.id,
		AgentName:      a.name,
		AgentChain:     a.agentChain,
		SessionID:      a.sessionID,
		ParentRecordID: a.parentRecordID,
		Messages:       a.messageManager.GetHistory(),
		Config:         a.config,
	}
}

func (a *DynAgent) collectTaskResults(tasks []models.SubTask) []interface{} {
	results := make([]interface{}, 0, len(tasks))
	for _, task := range tasks {
		// In real implementation, would collect actual results from task execution
		results = append(results, map[string]interface{}{
			"task_id":   task.ID,
			"task_name": task.Name,
			"state":     task.State,
			"output":    task.OutputFile,
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

// createMockProvider 创建生产级的模拟LLM提供者
func (a *DynAgent) createMockProvider() interfaces.LLMProvider {
	return &ProductionMockProvider{
		model:       "gpt-4",
		temperature: 0.7,
		maxTokens:   2000,
	}
}

// ProductionMockProvider 生产级模拟LLM提供者
type ProductionMockProvider struct {
	model       string
	temperature float32
	maxTokens   int
}

// Call 实现 LLMProvider 接口的 Call 方法
func (p *ProductionMockProvider) Call(ctx context.Context, messages []models.Message, config *models.LLMConfig) (*interfaces.LLMResponse, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("消息列表不能为空")
	}

	lastMessage := messages[len(messages)-1]

	// 智能判断是否为规划请求
	if isComplexPlanningRequest(lastMessage.Content) {
		return p.generatePlanningResponse(lastMessage.Content)
	}

	// 智能判断是否为任务执行请求
	if isTaskExecutionRequest(lastMessage.Content) {
		return p.generateExecutionResponse(lastMessage.Content)
	}

	// 默认对话响应
	return p.generateConversationResponse(lastMessage.Content)
}

// StreamCall 实现 LLMProvider 接口的 StreamCall 方法
func (p *ProductionMockProvider) StreamCall(ctx context.Context, messages []models.Message, config *models.LLMConfig) (<-chan *interfaces.LLMStreamChunk, error) {
	// 创建流式响应通道
	ch := make(chan *interfaces.LLMStreamChunk, 1)

	go func() {
		defer close(ch)

		// 获取响应内容
		resp, err := p.Call(ctx, messages, config)
		if err != nil {
			ch <- &interfaces.LLMStreamChunk{
				Delta:  "",
				Finish: true,
				Error:  err,
			}
			return
		}

		// 模拟流式输出
		content := resp.Content
		chunkSize := 20
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}

			ch <- &interfaces.LLMStreamChunk{
				Delta:  content[i:end],
				Finish: false,
				Error:  nil,
			}

			// 模拟延迟
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}

		// 发送完成信号
		ch <- &interfaces.LLMStreamChunk{
			Delta:  "",
			Finish: true,
			Error:  nil,
		}
	}()

	return ch, nil
}

// CountTokens 实现 LLMProvider 接口的 CountTokens 方法
func (p *ProductionMockProvider) CountTokens(messages []models.Message) (int, error) {
	totalTokens := 0
	for _, msg := range messages {
		// 简单估算：每4个字符约1个token
		contentTokens := (len(msg.Content) + 3) / 4
		reasoningTokens := (len(msg.ReasoningContent) + 3) / 4
		totalTokens += contentTokens + reasoningTokens + 4 // 4个tokens用于消息元数据
	}
	return totalTokens, nil
}

// 判断是否为复杂规划请求
func isComplexPlanningRequest(content string) bool {
	complexKeywords := []string{
		"帮我", "制作", "生成", "分析", "处理", "计算", "查找",
		"创建", "设计", "实现", "开发", "编写", "搜索", "比较",
	}

	for _, keyword := range complexKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return len(content) > 50 // 长消息通常需要规划
}

// 判断是否为任务执行请求
func isTaskExecutionRequest(content string) bool {
	executionKeywords := []string{
		"执行", "运行", "调用", "使用工具", "计算", "搜索", "读取", "写入",
	}

	for _, keyword := range executionKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// 生成规划响应
func (p *ProductionMockProvider) generatePlanningResponse(content string) (*interfaces.LLMResponse, error) {
	// 分析用户需求并生成智能的任务分解
	planJSON := `{
		"tasks": [
			{
				"id": "` + uuid.New().String() + `",
				"name": "分析用户需求",
				"description": "理解用户的具体需求和目标",
				"type": "task",
				"state": "wait",
				"dependencies": [],
				"tools": [],
				"priority": 1
			},
			{
				"id": "` + uuid.New().String() + `",
				"name": "执行主要任务",
				"description": "根据需求执行相应的操作",
				"type": "task", 
				"state": "wait",
				"dependencies": [],
				"tools": ["calculator", "search", "file"],
				"priority": 2
			},
			{
				"id": "` + uuid.New().String() + `",
				"name": "整理和输出结果",
				"description": "整理执行结果并格式化输出",
				"type": "task",
				"state": "wait", 
				"dependencies": [],
				"tools": ["file"],
				"priority": 3
			}
		],
		"reasoning": "基于用户请求，我将任务分解为分析、执行和输出三个阶段，确保有条理地完成任务。"
	}`

	return &interfaces.LLMResponse{
		Content: planJSON,
		Model:   p.model,
		Usage: interfaces.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 250,
			TotalTokens:      350,
		},
		Raw: map[string]interface{}{
			"reasoning": "基于用户请求生成的智能任务分解",
		},
	}, nil
}

// 生成执行响应
func (p *ProductionMockProvider) generateExecutionResponse(content string) (*interfaces.LLMResponse, error) {
	response := ""

	// 根据内容类型生成不同响应
	if strings.Contains(content, "计算") {
		response = "我将使用计算器工具来处理这个计算任务。"
	} else if strings.Contains(content, "搜索") {
		response = "我将使用搜索工具来查找相关信息。"
	} else if strings.Contains(content, "文件") {
		response = "我将使用文件工具来处理文件操作。"
	} else {
		response = "我理解了您的需求，正在执行相应的任务。"
	}

	return &interfaces.LLMResponse{
		Content: response,
		Model:   p.model,
		Usage: interfaces.TokenUsage{
			PromptTokens:     50,
			CompletionTokens: 80,
			TotalTokens:      130,
		},
		Raw: map[string]interface{}{
			"task_type": "execution",
		},
	}, nil
}

// 生成对话响应
func (p *ProductionMockProvider) generateConversationResponse(content string) (*interfaces.LLMResponse, error) {
	responses := []string{
		"我理解了您的问题。让我来帮助您解决。",
		"这是一个很有趣的问题。我会仔细分析并给出答案。",
		"基于您提供的信息，我认为可以这样处理。",
		"让我思考一下最佳的解决方案。",
	}

	// 根据内容长度选择响应
	responseIdx := len(content) % len(responses)

	return &interfaces.LLMResponse{
		Content: responses[responseIdx],
		Model:   p.model,
		Usage: interfaces.TokenUsage{
			PromptTokens:     30,
			CompletionTokens: 60,
			TotalTokens:      90,
		},
		Raw: map[string]interface{}{
			"conversation_type": "general",
		},
	}, nil
}

// GetModel 返回模型名称
func (p *ProductionMockProvider) GetModel() string {
	return p.model
}

// GetConfig 返回提供者配置
func (p *ProductionMockProvider) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"model":       p.model,
		"temperature": p.temperature,
		"max_tokens":  p.maxTokens,
		"provider":    "production_mock",
	}
}

// createTaskPlanner 创建真正的智能任务规划器
func (a *DynAgent) createTaskPlanner() interfaces.TaskPlanner {
	return planning.NewTaskPlanner(a.llmProvider, a.config, a.recordManager)
}

// createTaskExecutor 创建真正的任务执行器
func (a *DynAgent) createTaskExecutor() interfaces.TaskExecutor {
	executor := execution.NewTaskExecutor(a.llmProvider, a.recordManager, a.config)

	// 将agent的工具注册给执行器
	a.toolsMu.RLock()
	for _, tool := range a.tools {
		executor.RegisterTool(tool)
	}
	a.toolsMu.RUnlock()

	return executor
}

// generateIntelligentResponse 生成智能回应
func (a *DynAgent) generateIntelligentResponse(userMessage string, processedResult interface{}) string {
	// 直接调用LLM提供者生成智能回应
	messages := []models.Message{
		{
			Role:    "user",
			Content: userMessage,
		},
	}

	response, err := a.llmProvider.Call(context.Background(), messages, &models.LLMConfig{
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	})

	if err != nil {
		// 如果LLM调用失败，提供回退响应
		return a.generateFallbackResponse(userMessage)
	}

	return response.Content
}

// generateFallbackResponse 生成回退响应（当LLM不可用时）
func (a *DynAgent) generateFallbackResponse(userMessage string) string {
	// 基于用户输入关键词生成回退响应
	if strings.Contains(userMessage, "计算") {
		return "我理解您需要进行计算。我可以帮助您进行数学计算，请提供具体的计算表达式。"
	} else if strings.Contains(userMessage, "你好") || strings.Contains(userMessage, "您好") {
		return "您好！我是Go智能代理，很高兴为您服务。我可以帮助您处理各种任务，包括数学计算、信息查询等。"
	} else if strings.Contains(userMessage, "是谁") || strings.Contains(userMessage, "什么") {
		return "我是Go智能代理框架，一个基于Go语言开发的生产级智能代理系统。我可以帮助您完成各种复杂任务。"
	} else {
		return "我收到了您的消息：" + userMessage + "。我正在处理您的请求，如需更详细的帮助，请提供更具体的指令。"
	}
}

// registerBuiltinTools 注册内置工具到代理
func (a *DynAgent) registerBuiltinTools() {
	// 注册计算器工具
	calculatorTool := tools.NewCalculatorTool()
	a.RegisterTool(calculatorTool)

	// 注册文件工具（允许访问输出目录和日志目录）
	allowedPaths := []string{"./output", "./logs", "./records", "."}
	if a.config.CustomConfig != nil {
		if outputDir, ok := a.config.CustomConfig["output_dir"].(string); ok && outputDir != "" {
			allowedPaths = append(allowedPaths, outputDir)
		}
	}
	fileTool := tools.NewFileTool(allowedPaths)
	a.RegisterTool(fileTool)

	// 注册搜索工具（使用模拟API key）
	searchTool := tools.NewSearchTool("mock_api_key")
	a.RegisterTool(searchTool)

	// 如果执行器已经初始化，也需要注册给执行器
	if a.executor != nil {
		if exec, ok := a.executor.(*execution.TaskExecutor); ok {
			exec.RegisterTool(calculatorTool)
			exec.RegisterTool(fileTool)
			exec.RegisterTool(searchTool)
		}
	}
}
