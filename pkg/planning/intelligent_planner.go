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

// IntelligentTaskPlanner 智能任务规划器 - 实现真正的推理和规划能力
type IntelligentTaskPlanner struct {
	llmProvider   interfaces.LLMProvider
	config        *models.AgentConfig
	recordManager interfaces.RecordManager
}

// NewIntelligentTaskPlanner 创建智能任务规划器
func NewIntelligentTaskPlanner(
	llmProvider interfaces.LLMProvider,
	config *models.AgentConfig,
	recordManager interfaces.RecordManager,
) *IntelligentTaskPlanner {
	return &IntelligentTaskPlanner{
		llmProvider:   llmProvider,
		config:        config,
		recordManager: recordManager,
	}
}

// NeedsPlan 判断是否需要规划
func (p *IntelligentTaskPlanner) NeedsPlan(message string) bool {
	// 分析消息复杂度
	needsPlanKeywords := []string{
		"研究", "分析", "生成", "创建", "设计", "实现", "开发", "调查",
		"比较", "评估", "制作", "编写", "构建", "深度", "报告", "方案",
		"计划", "策略", "优化", "改进", "财报", "文档", "系统",
	}

	for _, keyword := range needsPlanKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	// 长消息通常需要规划
	return len(message) > 100
}

// Plan 执行智能规划
func (p *IntelligentTaskPlanner) Plan(ctx context.Context, message string, execContext *models.ExecutionContext) (*models.PlanningResult, error) {
	// 创建超时context
	planCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// 构建深度推理提示
	prompt := p.buildDeepReasoningPrompt(message)

	// 调用LLM进行深度推理和规划
	messages := []models.Message{
		{
			Role: "system",
			Content: `你是一个专业的任务规划专家，擅长将复杂任务分解为可执行的步骤。
你必须按照以下JSON格式返回规划结果，确保每个任务都有明确的：
1. 任务ID、名称和详细描述
2. 任务类型(research/analysis/generation/execution)
3. 任务依赖关系
4. 预期输出
5. 需要使用的工具或方法

请进行深度思考，生成详细的执行计划。`,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// 调用LLM
	response, err := p.llmProvider.Call(planCtx, messages, &p.config.LLMConfig)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析规划结果
	planResult, err := p.parsePlanningResponse(response.Content)
	if err != nil {
		// 如果解析失败，创建默认规划
		return p.createDefaultPlan(message), nil
	}

	// 记录规划结果
	p.recordManager.Record(
		interfaces.RecordTypePlanning,
		map[string]interface{}{
			"input":       message,
			"plan_result": planResult,
			"reasoning":   planResult.Summary,
		},
	)

	return planResult, nil
}

// buildDeepReasoningPrompt 构建深度推理提示
func (p *IntelligentTaskPlanner) buildDeepReasoningPrompt(message string) string {
	return fmt.Sprintf(`你是一个高级任务规划专家。请对以下请求进行深度分析和任务规划。

用户请求：%s

=== 规划框架 (基于Plan-Task-Action模型) ===

【第一步：理解与分析】
- 深入理解用户的真实需求和目标
- 识别任务类型(研究型/分析型/生成型/执行型)
- 评估任务复杂度和所需资源
- 确定关键成功因素

【第二步：智能分解】
根据任务性质，将其分解为3-7个核心子任务：
- 每个子任务必须有明确的输入和输出
- 任务之间的依赖关系必须清晰
- 支持并行执行的任务应当标注
- 需要迭代的任务应当说明迭代条件

【第三步：工具映射】
为每个子任务分配合适的执行方式：
- research: 调用ResearchAgent进行深度研究
- analysis: 使用分析工具进行数据处理
- generation: 调用生成型Agent创建内容
- execution: 执行具体操作任务
- synthesis: 综合多个来源的信息

【第四步：质量控制】
- 定义每个任务的验证标准
- 设置关键检查点
- 准备失败回滚方案

=== JSON输出格式 ===
{
  "plan_metadata": {
    "request_understanding": "对用户请求的理解",
    "main_objective": "主要目标",
    "complexity_level": "high|medium|low",
    "estimated_iterations": 1-5,
    "requires_deep_research": true/false
  },
  "tasks": [
    {
      "id": "task_xxx",
      "name": "任务名称",
      "description": "详细描述",
      "type": "research|analysis|generation|execution|synthesis",
      "execution_strategy": {
        "method": "具体执行方法",
        "iterations": "需要的迭代次数",
        "parallel_possible": true/false
      },
      "dependencies": ["依赖的任务ID列表"],
      "required_capabilities": ["search", "analyze", "generate", "calculate"],
      "input_requirements": "输入要求",
      "expected_output": {
        "format": "输出格式",
        "content": "预期内容描述",
        "quality_metrics": "质量指标"
      },
      "validation_criteria": "验证标准",
      "failure_handling": "失败处理策略"
    }
  ],
  "execution_flow": {
    "phases": [
      {
        "phase_name": "阶段名称",
        "tasks": ["task_ids"],
        "can_parallelize": true/false
      }
    ]
  },
  "reasoning": "详细的规划理由和思考过程",
  "risk_assessment": "风险评估",
  "success_criteria": "整体成功标准"
}

请基于以上框架，为用户请求生成一个智能、可执行的任务规划。`, message)
}

// parsePlanningResponse 解析LLM的规划响应
func (p *IntelligentTaskPlanner) parsePlanningResponse(response string) (*models.PlanningResult, error) {
	// 尝试提取JSON内容
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("响应中没有找到JSON内容")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	// 定义增强的结构体来解析响应
	var planData struct {
		PlanMetadata struct {
			RequestUnderstanding string `json:"request_understanding"`
			MainObjective        string `json:"main_objective"`
			ComplexityLevel      string `json:"complexity_level"`
			EstimatedIterations  int    `json:"estimated_iterations"`
			RequiresDeepResearch bool   `json:"requires_deep_research"`
		} `json:"plan_metadata"`
		Tasks []struct {
			ID                string `json:"id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			Type              string `json:"type"`
			ExecutionStrategy struct {
				Method           string `json:"method"`
				Iterations       string `json:"iterations"`
				ParallelPossible bool   `json:"parallel_possible"`
			} `json:"execution_strategy"`
			Dependencies         []string `json:"dependencies"`
			RequiredCapabilities []string `json:"required_capabilities"`
			InputRequirements    string   `json:"input_requirements"`
			ExpectedOutput       struct {
				Format         string `json:"format"`
				Content        string `json:"content"`
				QualityMetrics string `json:"quality_metrics"`
			} `json:"expected_output"`
			ValidationCriteria string `json:"validation_criteria"`
			FailureHandling    string `json:"failure_handling"`
		} `json:"tasks"`
		ExecutionFlow struct {
			Phases []struct {
				PhaseName      string   `json:"phase_name"`
				Tasks          []string `json:"tasks"`
				CanParallelize bool     `json:"can_parallelize"`
			} `json:"phases"`
		} `json:"execution_flow"`
		Reasoning       string `json:"reasoning"`
		RiskAssessment  string `json:"risk_assessment"`
		SuccessCriteria string `json:"success_criteria"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 转换为内部模型
	tasks := make([]models.SubTask, 0, len(planData.Tasks))
	dependencies := make(map[string][]string)

	for i, t := range planData.Tasks {
		// 确保有有效的ID
		taskID := t.ID
		if taskID == "" || taskID == "task_uuid" || taskID == "task_xxx" {
			taskID = fmt.Sprintf("task_%d_%s", i, uuid.New().String()[:8])
		}

		// 构建详细的任务描述
		fullDescription := fmt.Sprintf("%s\n\n执行策略: %s (迭代: %s)\n输入要求: %s\n预期输出: %s - %s\n质量指标: %s\n验证标准: %s\n失败处理: %s",
			t.Description,
			t.ExecutionStrategy.Method,
			t.ExecutionStrategy.Iterations,
			t.InputRequirements,
			t.ExpectedOutput.Format,
			t.ExpectedOutput.Content,
			t.ExpectedOutput.QualityMetrics,
			t.ValidationCriteria,
			t.FailureHandling)

		// 构建Process字段，包含执行细节
		processDetails := fmt.Sprintf("方法: %s | 能力需求: %s | 并行: %v",
			t.ExecutionStrategy.Method,
			strings.Join(t.RequiredCapabilities, ", "),
			t.ExecutionStrategy.ParallelPossible)

		task := models.SubTask{
			ID:          taskID,
			Name:        t.Name,
			Description: fullDescription,
			Type:        p.mapTaskType(t.Type),
			State:       models.TaskStateWait,
			Process:     processDetails,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			// 添加扩展属性
			Metadata: map[string]interface{}{
				"execution_strategy": t.ExecutionStrategy,
				"capabilities":       t.RequiredCapabilities,
				"expected_output":    t.ExpectedOutput,
				"validation":         t.ValidationCriteria,
				"failure_handling":   t.FailureHandling,
			},
		}

		// 处理依赖关系
		if len(t.Dependencies) > 0 {
			task.Dependent = t.Dependencies[0]    // 保留主要依赖
			dependencies[taskID] = t.Dependencies // 保存所有依赖
		}

		tasks = append(tasks, task)
	}

	// 构建详细的摘要
	summary := fmt.Sprintf("【规划摘要】\n目标: %s\n理解: %s\n复杂度: %s\n迭代次数: %d\n深度研究: %v\n风险: %s\n成功标准: %s\n理由: %s",
		planData.PlanMetadata.MainObjective,
		planData.PlanMetadata.RequestUnderstanding,
		planData.PlanMetadata.ComplexityLevel,
		planData.PlanMetadata.EstimatedIterations,
		planData.PlanMetadata.RequiresDeepResearch,
		planData.RiskAssessment,
		planData.SuccessCriteria,
		planData.Reasoning)

	return &models.PlanningResult{
		Tasks:        tasks,
		Dependencies: dependencies,
		Summary:      summary,
		Metadata: map[string]interface{}{
			"plan_metadata":  planData.PlanMetadata,
			"execution_flow": planData.ExecutionFlow,
			"risk":           planData.RiskAssessment,
			"success":        planData.SuccessCriteria,
		},
	}, nil
}

// mapTaskType 映射任务类型
func (p *IntelligentTaskPlanner) mapTaskType(taskType string) string {
	switch strings.ToLower(taskType) {
	case "research":
		return "agent_call" // 调用研究agent进行深度研究
	case "analysis":
		return "function" // 使用分析工具处理数据
	case "generation":
		return "agent_gen" // 生成新内容和报告
	case "execution":
		return "task" // 执行具体操作任务
	case "synthesis":
		return "agent_call" // 综合多源信息
	default:
		return "task"
	}
}

// createDefaultPlan 创建默认规划（当LLM规划失败时的后备方案）
func (p *IntelligentTaskPlanner) createDefaultPlan(message string) *models.PlanningResult {
	// 根据消息内容智能生成基础规划
	tasks := []models.SubTask{}

	// 判断任务类型
	if strings.Contains(message, "研究") || strings.Contains(message, "分析") {
		// 研究类任务
		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_0_%s", uuid.New().String()[:8]),
			Name:        "信息收集与研究",
			Description: "收集相关信息和数据，进行初步研究",
			Type:        "agent_call",
			State:       models.TaskStateWait,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})

		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_1_%s", uuid.New().String()[:8]),
			Name:        "深度分析",
			Description: "对收集的信息进行深度分析和整理",
			Type:        "task",
			State:       models.TaskStateWait,
			Dependent:   tasks[0].ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})

		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_2_%s", uuid.New().String()[:8]),
			Name:        "报告生成",
			Description: "基于分析结果生成详细报告",
			Type:        "agent_gen",
			State:       models.TaskStateWait,
			Dependent:   tasks[1].ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	} else if strings.Contains(message, "生成") || strings.Contains(message, "创建") {
		// 生成类任务
		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_0_%s", uuid.New().String()[:8]),
			Name:        "需求分析",
			Description: "分析生成需求和规范",
			Type:        "task",
			State:       models.TaskStateWait,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})

		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_1_%s", uuid.New().String()[:8]),
			Name:        "内容生成",
			Description: "根据需求生成内容",
			Type:        "agent_gen",
			State:       models.TaskStateWait,
			Dependent:   tasks[0].ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})

		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_2_%s", uuid.New().String()[:8]),
			Name:        "质量检查",
			Description: "检查和优化生成的内容",
			Type:        "task",
			State:       models.TaskStateWait,
			Dependent:   tasks[1].ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	} else {
		// 通用任务
		tasks = append(tasks, models.SubTask{
			ID:          fmt.Sprintf("task_0_%s", uuid.New().String()[:8]),
			Name:        "任务执行",
			Description: fmt.Sprintf("执行任务: %s", message),
			Type:        "task",
			State:       models.TaskStateWait,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &models.PlanningResult{
		Tasks:        tasks,
		Dependencies: make(map[string][]string),
		Summary:      fmt.Sprintf("为 '%s' 生成了%d个任务的执行计划", message, len(tasks)),
	}
}

// RevisePlan 修订现有规划
func (p *IntelligentTaskPlanner) RevisePlan(
	ctx context.Context,
	originalPlan *models.PlanningResult,
	feedback string,
) (*models.PlanningResult, error) {
	// 基于反馈修订规划
	prompt := fmt.Sprintf(`基于以下反馈修订执行计划：

原始计划：
%+v

反馈：%s

请生成修订后的计划，保持相同的JSON格式。`, originalPlan, feedback)

	messages := []models.Message{
		{
			Role:    "system",
			Content: "你是任务规划专家，擅长根据反馈优化执行计划。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := p.llmProvider.Call(ctx, messages, &p.config.LLMConfig)
	if err != nil {
		return originalPlan, err
	}

	return p.parsePlanningResponse(response.Content)
}
