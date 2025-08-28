package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go-agent/pkg/agent"
	"go-agent/pkg/config"
	"go-agent/pkg/models"
	_ "go-agent/pkg/planning"
	_ "go-agent/pkg/record"
)

// DeepResearchFramework 深度研究框架演示
func DeepResearchFramework() {
	fmt.Println("=== 深度研究示例 ===")
	fmt.Println("本示例展示如何使用智能规划器进行深度研究报告生成")
	fmt.Println()

	// 1. 加载配置
	_, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("加载配置失败，使用默认配置: %v", err)
	}

	// 2. 初始化研究Agent
	_ = agent.NewResearchAgent(&models.AgentConfig{
		Name:            "ResearchAgent",
		RoleDescription: "深度研究分析专家",
		MaxRounds:       10,
		Stream:          true,
	})

	// 3. 定义研究主题
	researchTopics := []struct {
		Topic        string
		Description  string
		Requirements map[string]interface{}
	}{
		{
			Topic:       "人工智能在医疗领域的应用前景",
			Description: "深度研究AI在医疗诊断、药物研发、个性化治疗等方面的应用",
			Requirements: map[string]interface{}{
				"depth":       3,          // 研究深度：3层
				"branches":    3,          // 每层3个分支
				"framework":   "1.1.1xxx", // 使用三层框架
				"truth_level": "long",     // 需要长真理分析
				"tool_usage":  true,       // 使用工具辅助
			},
		},
		{
			Topic:       "全球供应链数字化转型趋势",
			Description: "分析供应链数字化的现状、挑战和未来发展方向",
			Requirements: map[string]interface{}{
				"depth":     2,        // 研究深度：2层
				"branches":  4,        // 每层4个分支
				"framework": "1.4xxx", // 使用两层框架
				"focus":     "trends", // 关注趋势
			},
		},
		{
			Topic:       "可持续能源技术发展路线图",
			Description: "研究太阳能、风能、氢能等可持续能源技术的发展",
			Requirements: map[string]interface{}{
				"depth":     3,           // 研究深度：3层
				"branches":  3,           // 每层3个分支
				"framework": "1.1.1xxx",  // 使用三层框架
				"timeframe": "2024-2030", // 时间框架
			},
		},
	}

	// 4. 执行研究示例
	for i, research := range researchTopics {
		fmt.Printf("\n--- 研究案例 %d: %s ---\n", i+1, research.Topic)
		fmt.Printf("描述: %s\n", research.Description)
		fmt.Printf("框架: %v\n", research.Requirements["framework"])
		fmt.Println()

		// 创建研究上下文
		_, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// 模拟深度研究过程
		fmt.Println("1. 智能规划器分析任务复杂度...")
		simulatePlanning(research.Requirements)

		fmt.Println("2. 执行多轮迭代研究...")
		simulateIterativeResearch(research.Requirements)

		fmt.Println("3. 生成研究洞察...")
		insights := simulateInsightGeneration()

		fmt.Println("4. 生成最终报告...")
		report := generateSimulatedReport(research.Topic, research.Requirements, insights)

		// 保存报告
		saveReport(report, fmt.Sprintf("research_report_%d.json", i+1))

		// 展示关键发现
		displayKeyFindings(report)

		fmt.Println("\n研究完成！")
		fmt.Println(strings.Repeat("-", 50))
	}
}

// 此处移除了未使用的函数以简化示例

// saveReport 保存报告
func saveReport(report *models.ResearchReport, filename string) {
	// 将报告序列化为JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("序列化报告失败: %v", err)
		return
	}

	// 确保输出目录存在
	os.MkdirAll("./research_reports", 0755)

	// 保存到文件
	filepath := fmt.Sprintf("./research_reports/%s", filename)
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		log.Printf("保存报告失败: %v", err)
		return
	}

	fmt.Printf("报告已保存到: %s\n", filepath)
}

// displayKeyFindings 展示关键发现
func displayKeyFindings(report *models.ResearchReport) {
	fmt.Printf("\n=== 关键发现 ===\n")
	fmt.Printf("主题: %s\n", report.Topic)
	fmt.Printf("执行摘要:\n%s\n", report.ExecutiveSummary)

	// 展示每个章节的要点
	for i, section := range report.Sections {
		if i >= 3 {
			break // 只显示前3个章节
		}
		fmt.Printf("\n章节 %d: %s\n", i+1, section.Title)
		// 显示章节内容的前100个字符
		content := section.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		fmt.Printf("  %s\n", content)
	}

	// 展示研究元数据
	fmt.Printf("\n研究元数据:\n")
	fmt.Printf("- 作者: %s\n", report.Metadata.Author)
	fmt.Printf("- 版本: %s\n", report.Metadata.Version)
	fmt.Printf("- 方法论: %s\n", report.Metadata.Methodology)
	fmt.Printf("- 置信度: %.2f\n", report.Metadata.Confidence)
	fmt.Printf("- 审核状态: %s\n", report.Metadata.ReviewStatus)
}

// main 函数 - 演示完整的深度研究框架
func main() {
	fmt.Println("🚀 启动深度研究框架演示...")
	fmt.Println(strings.Repeat("=", 60))

	// 检查环境
	fmt.Println("📋 环境检查:")
	checkEnvironment()

	// 演示架构图中的多层次框架
	fmt.Println("\n📊 多层次研究框架演示:")
	demonstrateFrameworkLevels()

	// 运行深度研究示例
	fmt.Println("\n🔬 运行深度研究示例:")
	DeepResearchFramework()

	fmt.Println("\n✅ 所有演示完成！")
}

// checkEnvironment 检查环境配置
func checkEnvironment() {
	if os.Getenv("OPENAI_API_KEY") != "" {
		fmt.Println("✓ OPENAI_API_KEY 已配置")
	} else {
		fmt.Println("⚠ OPENAI_API_KEY 未配置，将使用模拟模式")
	}

	// 检查配置文件
	if _, err := os.Stat("config.yaml"); err == nil {
		fmt.Println("✓ config.yaml 配置文件存在")
	} else {
		fmt.Println("⚠ config.yaml 配置文件不存在，将使用默认配置")
	}
}

// demonstrateFrameworkLevels 演示不同框架层级
func demonstrateFrameworkLevels() {
	frameworks := map[string]string{
		"1.xxx":     "单层框架 - 3个主要研究方向",
		"1.1.xxx":   "双层框架 - 每个方向3个子研究",
		"1.1.1.xxx": "三层框架 - 深度递归研究",
	}

	fmt.Println("支持的研究框架类型:")
	for framework, description := range frameworks {
		fmt.Printf("• %s: %s\n", framework, description)
	}

	fmt.Println("\n核心特征:")
	features := []string{
		"完全依赖LLM推理能力",
		"自适应任务分解",
		"迭代质量优化",
		"多维度洞察生成",
		"递归深度研究",
	}

	for _, feature := range features {
		fmt.Printf("✓ %s\n", feature)
	}
}

// simulatePlanning 模拟智能规划过程
func simulatePlanning(requirements map[string]interface{}) {
	depth := requirements["depth"].(int)
	branches := requirements["branches"].(int)
	framework := requirements["framework"].(string)

	fmt.Printf("✓ 任务复杂度分析完成\n")
	fmt.Printf("  - 研究深度: %d层\n", depth)
	fmt.Printf("  - 分支数量: %d个/层\n", branches)
	fmt.Printf("  - 框架类型: %s\n", framework)
	fmt.Printf("  - 预估任务数: %d个\n", depth*branches)

	time.Sleep(300 * time.Millisecond)
}

// simulateIterativeResearch 模拟迭代研究过程
func simulateIterativeResearch(requirements map[string]interface{}) {
	depth := requirements["depth"].(int)

	for iteration := 1; iteration <= depth; iteration++ {
		fmt.Printf("✓ 第%d轮研究迭代\n", iteration)

		phases := []string{
			"数据收集与分析",
			"模式识别与归纳",
			"深度推理与验证",
			"洞察生成与整合",
		}

		for i, phase := range phases {
			if i >= iteration {
				break
			}
			fmt.Printf("  - %s\n", phase)
			time.Sleep(200 * time.Millisecond)
		}

		// 模拟质量评估
		quality := 0.70 + float64(iteration)*0.05
		fmt.Printf("  当前质量评分: %.2f\n", quality)

		if quality >= 0.85 {
			fmt.Printf("  ✓ 质量达标，可以进入下一轮\n")
		}
		fmt.Println()
	}
}

// simulateInsightGeneration 模拟洞察生成
func simulateInsightGeneration() []string {
	insights := []string{
		"识别出3个关键趋势和发展方向",
		"发现2个重要的跨领域关联关系",
		"生成4个高置信度的预测性洞察",
		"提出5个可行性建议和行动方案",
	}

	for _, insight := range insights {
		fmt.Printf("✓ %s\n", insight)
		time.Sleep(150 * time.Millisecond)
	}

	return insights
}

// generateSimulatedReport 生成模拟报告
func generateSimulatedReport(topic string, requirements map[string]interface{}, insights []string) *models.ResearchReport {
	depth := requirements["depth"].(int)
	framework := requirements["framework"].(string)

	report := &models.ResearchReport{
		ID:    fmt.Sprintf("report_%s", time.Now().Format("20060102150405")),
		Topic: topic,
		ExecutiveSummary: fmt.Sprintf("本研究采用%s框架对'%s'进行了%d层深度分析。通过多轮迭代研究和智能洞察生成，我们获得了%d项关键发现。",
			framework, topic, depth, len(insights)),
		Sections:  generateSimulatedSections(topic, depth, insights),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: models.ReportMetadata{
			Author:       "DeepResearchAgent",
			Version:      "1.0",
			Tags:         []string{"AI", "deep-research", framework},
			Confidence:   0.87,
			DataSources:  []string{"LLM推理", "智能分析", "迭代优化"},
			Methodology:  "基于Plan-Task-Action模型的递归深度研究",
			ReviewStatus: "final",
		},
	}

	return report
}

// generateSimulatedSections 生成模拟章节
func generateSimulatedSections(topic string, depth int, insights []string) []models.ReportSection {
	sections := []models.ReportSection{}

	sectionTitles := []string{
		"背景研究与概念框架",
		"深度分析与关键发现",
		"数据驱动的洞察分析",
		"趋势预测与风险评估",
		"综合结论与建议",
	}

	for i := 0; i < depth && i < len(sectionTitles); i++ {
		section := models.ReportSection{
			Title:   sectionTitles[i],
			Content: generateSectionContent(sectionTitles[i], topic, insights),
			Type:    determineSectionType(i),
		}

		// 添加子章节
		subSections := []models.ReportSubSection{
			{
				Title:   fmt.Sprintf("%s - 核心要点", sectionTitles[i]),
				Content: fmt.Sprintf("关于%s在%s方面的核心要点分析", topic, sectionTitles[i]),
				Level:   1,
			},
			{
				Title:   fmt.Sprintf("%s - 深度解析", sectionTitles[i]),
				Content: fmt.Sprintf("对%s的深度解析和专业见解", sectionTitles[i]),
				Level:   2,
			},
		}

		section.SubSections = subSections
		sections = append(sections, section)
	}

	return sections
}

// generateSectionContent 生成章节内容
func generateSectionContent(title, topic string, insights []string) string {
	content := fmt.Sprintf("## %s\n\n", title)
	content += fmt.Sprintf("本章节针对'%s'进行%s。\n\n", topic, title)

	content += "### 关键发现\n"
	for i, insight := range insights {
		if i >= 2 {
			break
		}
		content += fmt.Sprintf("- %s\n", insight)
	}

	content += "\n### 深度分析\n"
	content += "基于多轮迭代研究和智能推理，我们识别出以下关键模式和趋势：\n\n"
	content += "1. 技术发展呈现加速态势\n"
	content += "2. 跨领域融合成为新的增长点\n"
	content += "3. 系统性风险需要提前预防\n\n"

	content += "### 洞察与建议\n"
	content += "基于上述分析，建议关注以下几个方面：\n"
	content += "- 持续跟踪技术演进\n"
	content += "- 加强跨领域合作\n"
	content += "- 建立风险预警机制\n"

	return content
}

// determineSectionType 确定章节类型
func determineSectionType(index int) string {
	types := []string{"background", "analysis", "data", "prediction", "conclusion"}
	if index < len(types) {
		return types[index]
	}
	return "general"
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *config.Config {
	return &config.Config{
		Agent: config.AgentConfig{
			MaxSteps:  []int{5, 3, 2},
			MaxRounds: 10,
			Parallel:  false,
		},
		LLM: config.LLMConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   4000,
		},
		Execution: config.ExecutionConfig{
			Timeout: 30,
		},
		Logging: config.LoggingConfig{
			Level: "info",
			File:  "logs/agent.log",
		},
	}
}
