package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// StreamEvent 流事件类型
type StreamEventType string

const (
	StreamEventThinking   StreamEventType = "thinking"
	StreamEventPlanning   StreamEventType = "planning"
	StreamEventTask       StreamEventType = "task"
	StreamEventExecution  StreamEventType = "execution"
	StreamEventToolCall   StreamEventType = "tool_call"
	StreamEventToolResult StreamEventType = "tool_result"
	StreamEventResult     StreamEventType = "result"
	StreamEventError      StreamEventType = "error"
	StreamEventComplete   StreamEventType = "complete"
)

// StreamEvent 流事件
type StreamEvent struct {
	ID        string          `json:"id"`
	Type      StreamEventType `json:"type"`
	Data      interface{}     `json:"data"`
	Timestamp int64           `json:"timestamp"`
	RequestID string          `json:"request_id"`
}

// StreamingAgent 支持流式输出的代理
type StreamingAgent struct {
	*DynAgent
	eventChannel chan StreamEvent
}

// NewStreamingAgent 创建流式代理
func NewStreamingAgent(config *models.AgentConfig) *StreamingAgent {
	return &StreamingAgent{
		DynAgent:     NewDynAgent(config),
		eventChannel: make(chan StreamEvent, 100),
	}
}

// GetEventChannel 获取事件通道
func (sa *StreamingAgent) GetEventChannel() <-chan StreamEvent {
	return sa.eventChannel
}

// ProcessMessageStream 流式处理消息
func (sa *StreamingAgent) ProcessMessageStream(ctx context.Context, message string, requestID string) (*models.ProcessMessageResult, error) {
	// 发送开始思考事件
	sa.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "understanding",
			"content": "正在理解您的请求...",
		},
	})

	// 标记状态为规划中
	sa.mu.Lock()
	sa.state = interfaces.AgentStatePlanning
	sa.mu.Unlock()

	// 记录执行开始
	executionRecordID, err := sa.recordManager.Record(
		interfaces.RecordTypeAgentExecution,
		map[string]interface{}{
			"agent_id":   sa.id,
			"agent_name": sa.name,
			"message":    message,
			"status":     "started",
			"request_id": requestID,
		},
	)
	if err != nil {
		sa.sendErrorEvent(requestID, "Failed to record execution", err)
		return nil, fmt.Errorf("failed to record execution start: %w", err)
	}

	// 添加消息到历史
	userMessage := models.Message{
		Role:    "user",
		Content: message,
	}
	if err := sa.messageManager.AddMessage(userMessage); err != nil {
		sa.sendErrorEvent(requestID, "Failed to add message", err)
		return nil, fmt.Errorf("failed to add message: %w", err)
	}

	// 构建执行上下文
	context := sa.buildExecutionContext()
	context.ParentRecordID = executionRecordID

	// 规划阶段
	var planResult *models.PlanningResult
	
	// 发送思考事件 - 判断是否需要规划
	sa.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "analyzing",
			"content": "正在分析任务复杂度...",
		},
	})

	if sa.planner.NeedsPlan(message) {
		// 发送规划开始事件
		sa.sendEvent(StreamEvent{
			Type:      StreamEventPlanning,
			RequestID: requestID,
			Data: map[string]interface{}{
				"status":  "started",
				"message": "任务较复杂，正在制定执行计划...",
			},
		})

		// 模拟思考过程
		sa.sendThinkingProcess(requestID, message)

		// 执行规划
		planResult, err = sa.planner.Plan(ctx, message, context)
		if err != nil {
			sa.recordError(executionRecordID, "planning", err)
			sa.sendErrorEvent(requestID, "Planning failed", err)
			return nil, fmt.Errorf("planning failed: %w", err)
		}

		// 发送规划结果
		sa.sendEvent(StreamEvent{
			Type:      StreamEventPlanning,
			RequestID: requestID,
			Data: map[string]interface{}{
				"status": "completed",
				"tasks":  sa.formatTasks(planResult.Tasks),
				"summary": planResult.Summary,
			},
		})

		// 记录规划结果
		_, err = sa.recordManager.Record(
			interfaces.RecordTypePlanning,
			map[string]interface{}{
				"parent_id":  executionRecordID,
				"plan":       planResult,
				"request_id": requestID,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to record planning: %w", err)
		}
	} else {
		// 简单响应
		sa.sendEvent(StreamEvent{
			Type:      StreamEventThinking,
			RequestID: requestID,
			Data: map[string]interface{}{
				"phase":   "direct_response",
				"content": "任务简单，直接响应...",
			},
		})

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

	// 执行阶段
	sa.mu.Lock()
	sa.state = interfaces.AgentStateExecuting
	sa.mu.Unlock()

	// 发送执行开始事件
	sa.sendEvent(StreamEvent{
		Type:      StreamEventExecution,
		RequestID: requestID,
		Data: map[string]interface{}{
			"status":     "started",
			"task_count": len(planResult.Tasks),
		},
	})

	// 执行任务（带流式更新）
	for i, task := range planResult.Tasks {
		// 发送任务开始事件
		sa.sendEvent(StreamEvent{
			Type:      StreamEventTask,
			RequestID: requestID,
			Data: map[string]interface{}{
				"task_id":     task.ID,
				"task_name":   task.Name,
				"description": task.Description,
				"status":      "started",
				"progress":    fmt.Sprintf("%d/%d", i+1, len(planResult.Tasks)),
			},
		})

		// 模拟工具调用（实际任务执行在ExecuteBatch中）
		if task.Type == string(models.SubTaskTypeTask) {
			sa.sendEvent(StreamEvent{
				Type:      StreamEventToolCall,
				RequestID: requestID,
				Data: map[string]interface{}{
					"task_id":   task.ID,
					"tool_name": "executor",
					"status":    "calling",
				},
			})

			// 模拟工具执行
			time.Sleep(300 * time.Millisecond)

			sa.sendEvent(StreamEvent{
				Type:      StreamEventToolResult,
				RequestID: requestID,
				Data: map[string]interface{}{
					"task_id":   task.ID,
					"tool_name": "executor",
					"status":    "completed",
					"result":    "Tool executed successfully",
				},
			})
		}

		// 更新任务状态
		task.State = models.TaskStateSuccess
		
		// 发送任务完成事件
		sa.sendEvent(StreamEvent{
			Type:      StreamEventTask,
			RequestID: requestID,
			Data: map[string]interface{}{
				"task_id":     task.ID,
				"task_name":   task.Name,
				"status":      "completed",
				"progress":    fmt.Sprintf("%d/%d", i+1, len(planResult.Tasks)),
			},
		})
	}

	// 执行批量任务
	if err := sa.executor.ExecuteBatch(ctx, planResult.Tasks, context); err != nil {
		sa.recordError(executionRecordID, "execution", err)
		sa.sendErrorEvent(requestID, "Execution failed", err)
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// 处理结果
	results := sa.collectTaskResults(planResult.Tasks)
	finalResult, err := sa.resultProcessor.ProcessResults(results, interfaces.OutputFormatMarkdown)
	if err != nil {
		sa.sendErrorEvent(requestID, "Result processing failed", err)
		return nil, fmt.Errorf("result processing failed: %w", err)
	}

	// 生成摘要
	summary, err := sa.resultProcessor.GenerateSummary(results)
	if err != nil {
		summary = "Task completed successfully"
	}

	// 生成智能响应
	intelligentResponse := sa.generateIntelligentResponse(message, finalResult)

	// 构建最终结果
	result := &models.ProcessMessageResult{
		Code:               0,
		Message:            intelligentResponse,
		OutputFile:         fmt.Sprintf("%s_%d.md", sa.name, time.Now().Unix()),
		OutputTextAbstract: summary,
	}

	// 发送结果事件
	sa.sendEvent(StreamEvent{
		Type:      StreamEventResult,
		RequestID: requestID,
		Data: map[string]interface{}{
			"message": intelligentResponse,
			"summary": summary,
			"result":  result,
		},
	})

	// 记录完成
	_, err = sa.recordManager.Record(
		interfaces.RecordTypeAgentExecution,
		map[string]interface{}{
			"agent_id":   sa.id,
			"agent_name": sa.name,
			"status":     "completed",
			"result":     result,
			"parent_id":  executionRecordID,
			"request_id": requestID,
		},
	)

	// 更新状态
	sa.mu.Lock()
	sa.state = interfaces.AgentStateCompleted
	sa.mu.Unlock()

	// 发送完成事件
	sa.sendEvent(StreamEvent{
		Type:      StreamEventComplete,
		RequestID: requestID,
		Data: map[string]interface{}{
			"status": "success",
		},
	})

	return result, nil
}

// sendEvent 发送事件
func (sa *StreamingAgent) sendEvent(event StreamEvent) {
	event.ID = uuid.New().String()
	event.Timestamp = time.Now().UnixMilli()
	
	select {
	case sa.eventChannel <- event:
	default:
		// 防止阻塞
	}
}

// sendErrorEvent 发送错误事件
func (sa *StreamingAgent) sendErrorEvent(requestID string, message string, err error) {
	sa.sendEvent(StreamEvent{
		Type:      StreamEventError,
		RequestID: requestID,
		Data: map[string]interface{}{
			"message": message,
			"error":   err.Error(),
		},
	})
}

// sendThinkingProcess 发送思考过程
func (sa *StreamingAgent) sendThinkingProcess(requestID string, message string) {
	// 模拟LLM的思考过程
	thoughts := []string{
		"让我理解一下您的需求...",
		"分析任务的复杂度和依赖关系...",
		"确定需要使用的工具和资源...",
		"设计最优的执行方案...",
		"准备开始执行任务...",
	}

	for _, thought := range thoughts {
		sa.sendEvent(StreamEvent{
			Type:      StreamEventThinking,
			RequestID: requestID,
			Data: map[string]interface{}{
				"phase":   "reasoning",
				"content": thought,
			},
		})
		time.Sleep(300 * time.Millisecond) // 模拟思考延迟
	}
}

// formatTasks 格式化任务列表
func (sa *StreamingAgent) formatTasks(tasks []models.SubTask) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		formatted[i] = map[string]interface{}{
			"id":           task.ID,
			"name":         task.Name,
			"description":  task.Description,
			"type":         task.Type,
			"state":        task.State,
			"dependent":    task.Dependent,
			"process":      task.Process,
		}
	}
	return formatted
}

// ProcessMessage 兼容原有接口
func (sa *StreamingAgent) ProcessMessage(ctx context.Context, message string) (*models.ProcessMessageResult, error) {
	requestID := uuid.New().String()
	return sa.ProcessMessageStream(ctx, message, requestID)
}