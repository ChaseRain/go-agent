package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// ResearchAgent 深度研究Agent - 专门用于生成深度研究报告
// 具备自主推理和迭代优化能力
type ResearchAgent struct {
	*StreamingAgent
	researchDepth    int
	iterationLimit   int                    // 最大迭代次数
	qualityThreshold float64                // 质量阈值
	researchHistory  []models.ReportSection // 研究历史，用于迭代改进
	insightEngine    *InsightEngine         // 洞察生成引擎
}

// NewResearchAgent 创建研究Agent
func NewResearchAgent(config *models.AgentConfig) *ResearchAgent {
	// 设置研究专用配置
	config.Name = "ResearchAgent"
	config.RoleDescription = "专业的深度研究分析专家，具备自主推理和迭代优化能力"

	return &ResearchAgent{
		StreamingAgent:   NewStreamingAgent(config),
		researchDepth:    3,    // 默认研究深度
		iterationLimit:   5,    // 最多迭代5次
		qualityThreshold: 0.85, // 质量阈值85%
		researchHistory:  make([]models.ReportSection, 0),
		insightEngine:    NewInsightEngine(config),
	}
}

// ProcessResearchRequest 处理研究请求
func (ra *ResearchAgent) ProcessResearchRequest(ctx context.Context, topic string, requestID string) (*models.ResearchReport, error) {
	// 发送开始事件
	ra.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "research_init",
			"content": fmt.Sprintf("开始深度研究: %s", topic),
		},
	})

	// 执行多阶段研究
	report := &models.ResearchReport{
		ID:        uuid.New().String(),
		Topic:     topic,
		CreatedAt: time.Now(),
		Sections:  []models.ReportSection{},
	}

	// 阶段1: 背景研究
	ra.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "background_research",
			"content": "正在收集背景信息和相关资料...",
		},
	})

	backgroundSection, err := ra.researchBackground(ctx, topic)
	if err == nil {
		report.Sections = append(report.Sections, *backgroundSection)
		ra.sendStreamingContent(requestID, "背景研究", backgroundSection.Content)
	}

	// 阶段2: 深度分析
	ra.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "deep_analysis",
			"content": "正在进行深度分析...",
		},
	})

	analysisSection, err := ra.performDeepAnalysis(ctx, topic)
	if err == nil {
		report.Sections = append(report.Sections, *analysisSection)
		ra.sendStreamingContent(requestID, "深度分析", analysisSection.Content)
	}

	// 阶段3: 数据收集与处理
	ra.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "data_collection",
			"content": "正在收集和处理相关数据...",
		},
	})

	dataSection, err := ra.collectAndProcessData(ctx, topic)
	if err == nil {
		report.Sections = append(report.Sections, *dataSection)
		ra.sendStreamingContent(requestID, "数据分析", dataSection.Content)
	}

	// 阶段4: 综合分析与结论
	ra.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "synthesis",
			"content": "正在综合分析并生成结论...",
		},
	})

	conclusionSection, err := ra.synthesizeConclusion(ctx, topic, report.Sections)
	if err == nil {
		report.Sections = append(report.Sections, *conclusionSection)
		ra.sendStreamingContent(requestID, "结论与建议", conclusionSection.Content)
	}

	// 生成执行摘要
	report.ExecutiveSummary = ra.generateExecutiveSummary(report.Sections)

	// 发送完成事件
	ra.sendEvent(StreamEvent{
		Type:      StreamEventResult,
		RequestID: requestID,
		Data: map[string]interface{}{
			"message": fmt.Sprintf("研究报告已完成，共%d个章节", len(report.Sections)),
			"report":  report,
		},
	})

	return report, nil
}

// researchBackground 背景研究
func (ra *ResearchAgent) researchBackground(ctx context.Context, topic string) (*models.ReportSection, error) {
	prompt := fmt.Sprintf(`请对以下主题进行背景研究：

主题：%s

请提供：
1. 主题概述和定义
2. 历史背景和发展
3. 当前状况
4. 关键参与者或利益相关方
5. 重要性和影响

请提供详细、专业的分析。`, topic)

	messages := []models.Message{
		{
			Role:    "system",
			Content: "你是一位专业的研究分析师，擅长深度背景研究。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ra.llmProvider.Call(ctx, messages, &ra.config.LLMConfig)
	if err != nil {
		return nil, err
	}

	return &models.ReportSection{
		Title:   "背景研究",
		Content: response.Content,
		Type:    "background",
	}, nil
}

// performDeepAnalysis 深度分析
func (ra *ResearchAgent) performDeepAnalysis(ctx context.Context, topic string) (*models.ReportSection, error) {
	prompt := fmt.Sprintf(`对以下主题进行深度分析：

主题：%s

请从以下维度分析：
1. 核心要素分解
2. 关键挑战和问题
3. 机会和潜力
4. 风险评估
5. 趋势预测
6. 竞争格局（如适用）
7. 技术或方法论考虑

提供深入、批判性的分析。`, topic)

	messages := []models.Message{
		{
			Role:    "system",
			Content: "你是一位资深分析专家，擅长多维度深度分析。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ra.llmProvider.Call(ctx, messages, &ra.config.LLMConfig)
	if err != nil {
		return nil, err
	}

	return &models.ReportSection{
		Title:   "深度分析",
		Content: response.Content,
		Type:    "analysis",
	}, nil
}

// collectAndProcessData 数据收集与处理
func (ra *ResearchAgent) collectAndProcessData(ctx context.Context, topic string) (*models.ReportSection, error) {
	prompt := fmt.Sprintf(`为以下主题收集和分析相关数据：

主题：%s

请提供：
1. 关键数据点和统计
2. 数据来源和可靠性评估
3. 数据趋势分析
4. 对比分析（如有基准数据）
5. 数据洞察和发现
6. 数据限制和注意事项

如果是金融相关，请包含财务数据分析。`, topic)

	messages := []models.Message{
		{
			Role:    "system",
			Content: "你是一位数据分析专家，擅长处理和解释复杂数据。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ra.llmProvider.Call(ctx, messages, &ra.config.LLMConfig)
	if err != nil {
		return nil, err
	}

	return &models.ReportSection{
		Title:   "数据分析",
		Content: response.Content,
		Type:    "data",
	}, nil
}

// synthesizeConclusion 综合结论
func (ra *ResearchAgent) synthesizeConclusion(ctx context.Context, topic string, previousSections []models.ReportSection) (*models.ReportSection, error) {
	// 构建之前章节的摘要
	var sectionSummaries []string
	for _, section := range previousSections {
		// 取每个章节的前500字符作为摘要
		summary := section.Content
		if len(summary) > 500 {
			summary = summary[:500] + "..."
		}
		sectionSummaries = append(sectionSummaries, fmt.Sprintf("%s: %s", section.Title, summary))
	}

	prompt := fmt.Sprintf(`基于以下研究内容，生成综合结论和建议：

主题：%s

已完成的研究章节摘要：
%s

请提供：
1. 关键发现总结
2. 主要结论
3. 具体建议和行动项
4. 未来展望
5. 需要进一步研究的领域

确保结论基于前述分析，逻辑清晰，建议可操作。`, topic, strings.Join(sectionSummaries, "\n\n"))

	messages := []models.Message{
		{
			Role:    "system",
			Content: "你是一位资深战略顾问，擅长综合分析和提供可行建议。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ra.llmProvider.Call(ctx, messages, &ra.config.LLMConfig)
	if err != nil {
		return nil, err
	}

	return &models.ReportSection{
		Title:   "结论与建议",
		Content: response.Content,
		Type:    "conclusion",
	}, nil
}

// generateExecutiveSummary 生成执行摘要
func (ra *ResearchAgent) generateExecutiveSummary(sections []models.ReportSection) string {
	var summary strings.Builder
	summary.WriteString("【执行摘要】\n\n")

	for _, section := range sections {
		// 提取每个章节的关键点
		keyPoints := ra.extractKeyPoints(section.Content)
		summary.WriteString(fmt.Sprintf("◆ %s\n%s\n\n", section.Title, keyPoints))
	}

	return summary.String()
}

// extractKeyPoints 提取关键点
func (ra *ResearchAgent) extractKeyPoints(content string) string {
	// 简化版：取前200字符
	// 实际应该使用NLP技术提取关键句
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}

// sendStreamingContent 发送流式内容
func (ra *ResearchAgent) sendStreamingContent(requestID string, title string, content string) {
	// 分块发送内容
	chunkSize := 100
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}

		ra.sendEvent(StreamEvent{
			Type:      StreamEventResult,
			RequestID: requestID,
			Data: map[string]interface{}{
				"section":   title,
				"content":   content[i:end],
				"streaming": true,
				"progress":  fmt.Sprintf("%d/%d", end, len(content)),
			},
		})

		time.Sleep(50 * time.Millisecond) // 模拟流式延迟
	}
}

// InsightEngine 洞察生成引擎 - 负责从数据中提取深度洞察
type InsightEngine struct {
	llmProvider interfaces.LLMProvider
	config      *models.AgentConfig
}

// NewInsightEngine 创建洞察引擎
func NewInsightEngine(config *models.AgentConfig) *InsightEngine {
	return &InsightEngine{
		config: config,
	}
}

// SetLLMProvider 设置LLM提供者
func (ie *InsightEngine) SetLLMProvider(provider interfaces.LLMProvider) {
	ie.llmProvider = provider
}

// GenerateInsights 从研究内容生成深度洞察
func (ie *InsightEngine) GenerateInsights(ctx context.Context, sections []models.ReportSection, topic string) ([]models.InsightItem, error) {
	// 构建洞察生成提示
	sectionSummaries := make([]string, 0, len(sections))
	for _, section := range sections {
		sectionSummaries = append(sectionSummaries, fmt.Sprintf("【%s】: %s", section.Title, ie.extractKeyFindings(section.Content)))
	}

	prompt := fmt.Sprintf(`基于以下研究内容，生成深度洞察和发现。

主题：%s

研究内容摘要：
%s

请识别以下类型的洞察：
1. **趋势洞察** - 识别重要趋势和发展方向
2. **异常洞察** - 发现异常模式或意外发现 
3. **关联洞察** - 识别不同因素之间的关联关系
4. **预测洞察** - 基于现有数据的合理预测

对每个洞察提供：
- 明确的洞察描述
- 支撑证据
- 影响程度评估（high/medium/low）
- 建议的行动方案

请以JSON数组格式返回洞察，每个洞察包含：
{
  "type": "trend|anomaly|correlation|prediction",
  "title": "洞察标题",
  "description": "详细描述",
  "impact": "high|medium|low",
  "evidence": ["证据1", "证据2"],
  "recommended_actions": ["行动1", "行动2"]
}`, topic, strings.Join(sectionSummaries, "\n"))

	messages := []models.Message{
		{
			Role:    "system",
			Content: "你是一位顶级的数据洞察专家，擅长从复杂信息中提取深度洞察和模式。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ie.llmProvider.Call(ctx, messages, &ie.config.LLMConfig)
	if err != nil {
		return nil, err
	}

	return ie.parseInsights(response.Content)
}

// parseInsights 解析洞察响应
func (ie *InsightEngine) parseInsights(response string) ([]models.InsightItem, error) {
	// 提取JSON数组
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("响应中没有找到JSON数组")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var rawInsights []struct {
		Type               string   `json:"type"`
		Title              string   `json:"title"`
		Description        string   `json:"description"`
		Impact             string   `json:"impact"`
		Evidence           []string `json:"evidence"`
		RecommendedActions []string `json:"recommended_actions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawInsights); err != nil {
		return nil, fmt.Errorf("解析洞察JSON失败: %w", err)
	}

	insights := make([]models.InsightItem, 0, len(rawInsights))
	for _, raw := range rawInsights {
		insight := models.InsightItem{
			ID:          uuid.New().String(),
			Type:        raw.Type,
			Title:       raw.Title,
			Description: raw.Description,
			Impact:      raw.Impact,
			Evidence:    raw.Evidence,
			Actions:     raw.RecommendedActions,
			CreatedAt:   time.Now(),
		}
		insights = append(insights, insight)
	}

	return insights, nil
}

// extractKeyFindings 提取关键发现
func (ie *InsightEngine) extractKeyFindings(content string) string {
	// 简化版：取前300字符并尝试找到关键句
	if len(content) <= 300 {
		return content
	}

	// 查找关键词句
	sentences := strings.Split(content, "。")
	keyFindings := make([]string, 0)

	keywords := []string{"关键", "重要", "显示", "表明", "发现", "结果", "趋势", "影响"}

	for _, sentence := range sentences {
		if len(keyFindings) >= 3 {
			break
		}
		for _, keyword := range keywords {
			if strings.Contains(sentence, keyword) {
				keyFindings = append(keyFindings, strings.TrimSpace(sentence)+"。")
				break
			}
		}
	}

	if len(keyFindings) == 0 {
		return content[:300] + "..."
	}

	return strings.Join(keyFindings, " ")
}

// AutonomousResearch 自主研究方法 - 使用迭代改进的方式进行深度研究
func (ra *ResearchAgent) AutonomousResearch(ctx context.Context, task *models.ResearchTask, requestID string) (*models.ResearchReport, error) {
	ra.sendEvent(StreamEvent{
		Type:      StreamEventThinking,
		RequestID: requestID,
		Data: map[string]interface{}{
			"phase":   "autonomous_init",
			"content": fmt.Sprintf("开始自主深度研究: %s (深度: %d)", task.Topic, task.Depth),
		},
	})

	// 设置洞察引擎的LLM提供者
	ra.insightEngine.SetLLMProvider(ra.llmProvider)

	report := &models.ResearchReport{
		ID:        uuid.New().String(),
		Topic:     task.Topic,
		CreatedAt: time.Now(),
		Sections:  []models.ReportSection{},
		Metadata: models.ReportMetadata{
			Author:      "ResearchAgent",
			Version:     "1.0",
			Methodology: "自主迭代深度研究",
		},
	}

	// 多轮迭代研究
	for iteration := 1; iteration <= ra.iterationLimit; iteration++ {
		ra.sendEvent(StreamEvent{
			Type:      StreamEventThinking,
			RequestID: requestID,
			Data: map[string]interface{}{
				"phase":     "iteration",
				"iteration": iteration,
				"content":   fmt.Sprintf("开始第%d轮研究迭代...", iteration),
			},
		})

		// 根据当前研究状态动态调整研究策略
		researchPlan := ra.generateIterativeResearchPlan(ctx, task, report.Sections, iteration)

		// 执行当前轮次的研究
		newSections, err := ra.executeIterativeResearch(ctx, researchPlan, task.Topic, requestID)
		if err != nil {
			continue // 继续下一次迭代
		}

		// 合并新发现到报告中
		report.Sections = append(report.Sections, newSections...)

		// 生成当前轮次的洞察
		insights, err := ra.insightEngine.GenerateInsights(ctx, report.Sections, task.Topic)
		if err == nil {
			// 将洞察集成到报告元数据中
			report.Metadata.Tags = append(report.Metadata.Tags, fmt.Sprintf("iteration_%d_insights_%d", iteration, len(insights)))
		}

		// 评估研究质量
		quality := ra.evaluateResearchQuality(report.Sections)
		ra.sendEvent(StreamEvent{
			Type:      StreamEventThinking,
			RequestID: requestID,
			Data: map[string]interface{}{
				"phase":     "quality_check",
				"iteration": iteration,
				"quality":   quality,
				"content":   fmt.Sprintf("第%d轮研究质量: %.2f", iteration, quality),
			},
		})

		// 如果质量达到阈值，提前结束
		if quality >= ra.qualityThreshold {
			ra.sendEvent(StreamEvent{
				Type:      StreamEventThinking,
				RequestID: requestID,
				Data: map[string]interface{}{
					"phase":   "early_termination",
					"content": fmt.Sprintf("研究质量达标(%.2f >= %.2f)，提前结束迭代", quality, ra.qualityThreshold),
				},
			})
			break
		}
	}

	// 生成最终的综合洞察
	finalInsights, err := ra.insightEngine.GenerateInsights(ctx, report.Sections, task.Topic)
	if err == nil {
		// 将洞察转换为报告章节
		insightSection := ra.convertInsightsToSection(finalInsights)
		report.Sections = append(report.Sections, *insightSection)
	}

	// 生成执行摘要
	report.ExecutiveSummary = ra.generateAdvancedExecutiveSummary(report.Sections, finalInsights)
	report.Metadata.Confidence = ra.calculateConfidenceScore(report.Sections)
	report.Metadata.ReviewStatus = "final"

	return report, nil
}

// generateIterativeResearchPlan 生成迭代研究计划
func (ra *ResearchAgent) generateIterativeResearchPlan(ctx context.Context, task *models.ResearchTask, existingSections []models.ReportSection, iteration int) []string {
	// 分析已有研究内容的覆盖度
	coveredAspects := make([]string, 0)
	for _, section := range existingSections {
		coveredAspects = append(coveredAspects, section.Title)
	}

	// 根据迭代次数和已有内容，动态生成研究计划
	switch iteration {
	case 1:
		return []string{"基础背景研究", "核心概念定义", "当前状况分析"}
	case 2:
		return []string{"深度技术分析", "市场环境研究", "竞争态势评估"}
	case 3:
		return []string{"趋势预测分析", "风险机会识别", "案例研究"}
	case 4:
		return []string{"量化数据分析", "专家观点收集", "跨领域关联"}
	default:
		return []string{"综合验证研究", "遗漏点补充", "质量提升优化"}
	}
}

// executeIterativeResearch 执行迭代研究
func (ra *ResearchAgent) executeIterativeResearch(ctx context.Context, researchPlan []string, topic string, requestID string) ([]models.ReportSection, error) {
	sections := make([]models.ReportSection, 0)

	for _, planItem := range researchPlan {
		section, err := ra.conductDetailedResearch(ctx, topic, planItem)
		if err != nil {
			continue // 跳过失败的研究项
		}

		sections = append(sections, *section)
		ra.sendStreamingContent(requestID, planItem, section.Content)
	}

	return sections, nil
}

// conductDetailedResearch 执行详细研究
func (ra *ResearchAgent) conductDetailedResearch(ctx context.Context, topic string, aspect string) (*models.ReportSection, error) {
	prompt := fmt.Sprintf(`请对以下主题的特定方面进行深入研究：

主题：%s
研究方面：%s

请提供：
1. 详细的事实性信息和数据
2. 多角度的分析和解释
3. 相关的案例或实例
4. 可能的影响和意义
5. 与其他因素的关联性

要求：
- 信息准确、客观
- 分析深入、全面
- 逻辑清晰、结构化
- 避免重复已知的基础信息`, topic, aspect)

	messages := []models.Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("你是%s领域的资深研究专家，擅长%s的深度分析。", topic, aspect),
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	response, err := ra.llmProvider.Call(ctx, messages, &ra.config.LLMConfig)
	if err != nil {
		return nil, err
	}

	return &models.ReportSection{
		Title:   aspect,
		Content: response.Content,
		Type:    "detailed_research",
	}, nil
}

// evaluateResearchQuality 评估研究质量
func (ra *ResearchAgent) evaluateResearchQuality(sections []models.ReportSection) float64 {
	if len(sections) == 0 {
		return 0.0
	}

	totalScore := 0.0

	for _, section := range sections {
		// 基于内容长度评分 (30%)
		lengthScore := math.Min(float64(len(section.Content))/2000.0, 1.0) * 0.3

		// 基于关键词覆盖度评分 (40%)
		keywordScore := ra.evaluateKeywordCoverage(section.Content) * 0.4

		// 基于结构化程度评分 (30%)
		structureScore := ra.evaluateContentStructure(section.Content) * 0.3

		totalScore += lengthScore + keywordScore + structureScore
	}

	return totalScore / float64(len(sections))
}

// evaluateKeywordCoverage 评估关键词覆盖度
func (ra *ResearchAgent) evaluateKeywordCoverage(content string) float64 {
	qualityKeywords := []string{
		"分析", "研究", "数据", "趋势", "影响", "发展", "技术", "市场",
		"挑战", "机会", "风险", "策略", "案例", "结果", "结论", "建议",
	}

	foundKeywords := 0
	for _, keyword := range qualityKeywords {
		if strings.Contains(content, keyword) {
			foundKeywords++
		}
	}

	return float64(foundKeywords) / float64(len(qualityKeywords))
}

// evaluateContentStructure 评估内容结构化程度
func (ra *ResearchAgent) evaluateContentStructure(content string) float64 {
	structureIndicators := []string{
		"1.", "2.", "3.", "•", "-", "：", "：", "【", "】", "（", "）",
	}

	foundIndicators := 0
	for _, indicator := range structureIndicators {
		if strings.Contains(content, indicator) {
			foundIndicators++
		}
	}

	return math.Min(float64(foundIndicators)/5.0, 1.0) // 最多5个指标就算满分
}

// convertInsightsToSection 将洞察转换为报告章节
func (ra *ResearchAgent) convertInsightsToSection(insights []models.InsightItem) *models.ReportSection {
	var content strings.Builder
	content.WriteString("# 深度洞察与发现\n\n")

	// 按影响程度分组
	highImpact := []models.InsightItem{}
	mediumImpact := []models.InsightItem{}
	lowImpact := []models.InsightItem{}

	for _, insight := range insights {
		switch insight.Impact {
		case "high":
			highImpact = append(highImpact, insight)
		case "medium":
			mediumImpact = append(mediumImpact, insight)
		default:
			lowImpact = append(lowImpact, insight)
		}
	}

	// 输出高影响洞察
	if len(highImpact) > 0 {
		content.WriteString("## 🔥 高影响洞察\n\n")
		for i, insight := range highImpact {
			content.WriteString(fmt.Sprintf("### %d. %s\n", i+1, insight.Title))
			content.WriteString(fmt.Sprintf("**类型**: %s\n\n", insight.Type))
			content.WriteString(fmt.Sprintf("%s\n\n", insight.Description))
			if len(insight.Evidence) > 0 {
				content.WriteString("**支撑证据**:\n")
				for _, evidence := range insight.Evidence {
					content.WriteString(fmt.Sprintf("- %s\n", evidence))
				}
				content.WriteString("\n")
			}
			if len(insight.Actions) > 0 {
				content.WriteString("**建议行动**:\n")
				for _, action := range insight.Actions {
					content.WriteString(fmt.Sprintf("- %s\n", action))
				}
				content.WriteString("\n")
			}
		}
	}

	// 输出中等影响洞察
	if len(mediumImpact) > 0 {
		content.WriteString("## 📊 中等影响洞察\n\n")
		for i, insight := range mediumImpact {
			content.WriteString(fmt.Sprintf("### %d. %s\n", i+1, insight.Title))
			content.WriteString(fmt.Sprintf("%s\n\n", insight.Description))
		}
	}

	// 输出低影响洞察（简化显示）
	if len(lowImpact) > 0 {
		content.WriteString("## 💡 其他发现\n\n")
		for _, insight := range lowImpact {
			content.WriteString(fmt.Sprintf("- **%s**: %s\n", insight.Title, insight.Description))
		}
		content.WriteString("\n")
	}

	return &models.ReportSection{
		Title:   "深度洞察与发现",
		Content: content.String(),
		Type:    "insights",
	}
}

// generateAdvancedExecutiveSummary 生成高级执行摘要
func (ra *ResearchAgent) generateAdvancedExecutiveSummary(sections []models.ReportSection, insights []models.InsightItem) string {
	var summary strings.Builder
	summary.WriteString("# 执行摘要\n\n")

	// 研究概述
	summary.WriteString("## 研究概述\n")
	summary.WriteString(fmt.Sprintf("本研究包含%d个主要章节，", len(sections)))

	highImpactInsights := 0
	for _, insight := range insights {
		if insight.Impact == "high" {
			highImpactInsights++
		}
	}
	summary.WriteString(fmt.Sprintf("识别出%d个高影响洞察，", highImpactInsights))
	summary.WriteString(fmt.Sprintf("总计%d项研究发现。\n\n", len(insights)))

	// 关键发现
	summary.WriteString("## 关键发现\n")
	for i, insight := range insights {
		if insight.Impact == "high" && i < 3 { // 只显示前3个高影响洞察
			summary.WriteString(fmt.Sprintf("- **%s**: %s\n", insight.Title, insight.Description))
		}
	}
	summary.WriteString("\n")

	// 章节概要
	summary.WriteString("## 研究章节概要\n")
	for _, section := range sections {
		keyFindings := ra.insightEngine.extractKeyFindings(section.Content)
		summary.WriteString(fmt.Sprintf("- **%s**: %s\n", section.Title, keyFindings))
	}

	return summary.String()
}

// calculateConfidenceScore 计算置信度分数
func (ra *ResearchAgent) calculateConfidenceScore(sections []models.ReportSection) float64 {
	if len(sections) == 0 {
		return 0.0
	}

	totalConfidence := 0.0

	for _, section := range sections {
		// 基于内容质量计算置信度
		contentLength := float64(len(section.Content))
		confidence := math.Min(contentLength/1000.0, 1.0) * 0.9 // 长度因子最高90%

		// 基于结构化程度调整
		structureBonus := ra.evaluateContentStructure(section.Content) * 0.1
		confidence += structureBonus

		totalConfidence += confidence
	}

	return math.Min(totalConfidence/float64(len(sections)), 1.0)
}
