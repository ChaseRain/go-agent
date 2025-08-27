package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-agent/pkg/interfaces"
	"go-agent/pkg/models"
)

// ResultProcessor 实现生产级结果处理器
type ResultProcessor struct {
	outputDir     string
	templateDir   string
	enableBackup  bool
	maxFileSize   int64
	outputFormats []interfaces.OutputFormat
}

// ProcessorConfig 结果处理器配置
type ProcessorConfig struct {
	OutputDir     string                    `json:"output_dir"`
	TemplateDir   string                    `json:"template_dir"`
	EnableBackup  bool                      `json:"enable_backup"`
	MaxFileSize   int64                     `json:"max_file_size"`
	OutputFormats []interfaces.OutputFormat `json:"output_formats"`
}

// ProcessingResult 处理结果结构
type ProcessingResult struct {
	ID          string                  `json:"id"`
	Format      interfaces.OutputFormat `json:"format"`
	Content     string                  `json:"content"`
	FilePath    string                  `json:"file_path"`
	Size        int64                   `json:"size"`
	ProcessedAt time.Time               `json:"processed_at"`
	Metadata    map[string]interface{}  `json:"metadata"`
	TaskResults []TaskResultSummary     `json:"task_results"`
}

// TaskResultSummary 任务结果摘要
type TaskResultSummary struct {
	TaskID      string                 `json:"task_id"`
	TaskName    string                 `json:"task_name"`
	Status      models.TaskState       `json:"status"`
	Output      interface{}            `json:"output"`
	Duration    time.Duration          `json:"duration"`
	TokensUsed  int                    `json:"tokens_used"`
	ToolsCalled []string               `json:"tools_called"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewResultProcessor 创建新的结果处理器
func NewResultProcessor(config *ProcessorConfig) *ResultProcessor {
	if config == nil {
		config = &ProcessorConfig{
			OutputDir:     "./output",
			TemplateDir:   "./templates",
			EnableBackup:  true,
			MaxFileSize:   10 * 1024 * 1024, // 10MB
			OutputFormats: []interfaces.OutputFormat{interfaces.OutputFormatMarkdown},
		}
	}

	// 确保输出目录存在
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create output directory: %v", err))
	}

	return &ResultProcessor{
		outputDir:     config.OutputDir,
		templateDir:   config.TemplateDir,
		enableBackup:  config.EnableBackup,
		maxFileSize:   config.MaxFileSize,
		outputFormats: config.OutputFormats,
	}
}

// ProcessResults 处理任务执行结果
func (p *ResultProcessor) ProcessResults(results []interface{}, format interfaces.OutputFormat) (interface{}, error) {
	ctx := context.Background()
	return p.ProcessResultsWithContext(ctx, results, format)
}

// ProcessResultsWithContext 使用上下文处理任务执行结果
func (p *ResultProcessor) ProcessResultsWithContext(ctx context.Context, results []interface{}, format interfaces.OutputFormat) (interface{}, error) {
	processingID := fmt.Sprintf("proc_%d", time.Now().UnixNano())

	// 解析任务结果
	taskSummaries, err := p.parseTaskResults(results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task results: %w", err)
	}

	// 生成内容
	content, err := p.generateContent(ctx, taskSummaries, format)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// 创建处理结果
	result := &ProcessingResult{
		ID:          processingID,
		Format:      format,
		Content:     content,
		ProcessedAt: time.Now().UTC(),
		TaskResults: taskSummaries,
		Metadata: map[string]interface{}{
			"total_tasks":      len(taskSummaries),
			"successful_tasks": p.countSuccessfulTasks(taskSummaries),
			"failed_tasks":     p.countFailedTasks(taskSummaries),
		},
	}

	return result, nil
}

// GenerateSummary 生成执行摘要
func (p *ResultProcessor) GenerateSummary(results []interface{}) (string, error) {
	taskSummaries, err := p.parseTaskResults(results)
	if err != nil {
		return "", fmt.Errorf("failed to parse task results: %w", err)
	}

	totalTasks := len(taskSummaries)
	successfulTasks := p.countSuccessfulTasks(taskSummaries)
	failedTasks := p.countFailedTasks(taskSummaries)

	summary := fmt.Sprintf("执行完成: 总任务数 %d, 成功 %d, 失败 %d",
		totalTasks, successfulTasks, failedTasks)

	if totalTasks > 0 {
		successRate := float64(successfulTasks) / float64(totalTasks) * 100
		summary += fmt.Sprintf(" (成功率: %.1f%%)", successRate)
	}

	// 添加工具使用统计
	toolsUsed := p.collectToolsUsed(taskSummaries)
	if len(toolsUsed) > 0 {
		summary += fmt.Sprintf(", 使用工具: %s", strings.Join(toolsUsed, ", "))
	}

	return summary, nil
}

// SaveToFile 保存结果到文件
func (p *ResultProcessor) SaveToFile(results interface{}, filepath string) error {
	if !strings.HasPrefix(filepath, "/") {
		// 相对路径，加上输出目录前缀
		filepath = filepath_Join(p.outputDir, filepath)
	}

	// 确保目录存在
	dir := filepath_Dir(filepath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var content []byte
	var err error

	// 根据结果类型处理
	if processingResult, ok := results.(*ProcessingResult); ok {
		content = []byte(processingResult.Content)
		processingResult.FilePath = filepath
		processingResult.Size = int64(len(content))
	} else if str, ok := results.(string); ok {
		content = []byte(str)
	} else {
		// JSON序列化
		content, err = json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
	}

	// 检查文件大小
	if int64(len(content)) > p.maxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", len(content), p.maxFileSize)
	}

	// 备份现有文件
	if p.enableBackup && p.fileExists(filepath) {
		if err := p.backupFile(filepath); err != nil {
			// 记录错误但继续执行
			fmt.Printf("Warning: failed to backup file %s: %v\n", filepath, err)
		}
	}

	// 写入文件
	if err := ioutil.WriteFile(filepath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filepath, err)
	}

	return nil
}

// GenerateReport 生成详细报告
func (p *ResultProcessor) GenerateReport(results []interface{}, format interfaces.OutputFormat) (string, error) {
	taskSummaries, err := p.parseTaskResults(results)
	if err != nil {
		return "", fmt.Errorf("failed to parse task results: %w", err)
	}

	switch format {
	case interfaces.OutputFormatMarkdown:
		return p.generateMarkdownReport(taskSummaries), nil
	case interfaces.OutputFormatJSON:
		return p.generateJSONReport(taskSummaries)
	case interfaces.OutputFormatText:
		return p.generateTextReport(taskSummaries), nil
	default:
		return p.generateMarkdownReport(taskSummaries), nil
	}
}

// 私有方法

func (p *ResultProcessor) parseTaskResults(results []interface{}) ([]TaskResultSummary, error) {
	var summaries []TaskResultSummary

	for _, result := range results {
		if resultMap, ok := result.(map[string]interface{}); ok {
			summary := TaskResultSummary{
				TaskID:   getStringValue(resultMap, "task_id"),
				TaskName: getStringValue(resultMap, "task_name"),
				Output:   resultMap["output"],
				Metadata: make(map[string]interface{}),
			}

			// 解析任务状态
			if stateStr := getStringValue(resultMap, "state"); stateStr != "" {
				summary.Status = models.TaskState(stateStr)
			} else {
				summary.Status = models.TaskStateSuccess // 默认成功
			}

			// 提取工具调用信息
			if toolsInterface, exists := resultMap["tools_called"]; exists {
				if toolsSlice, ok := toolsInterface.([]interface{}); ok {
					for _, tool := range toolsSlice {
						if toolStr, ok := tool.(string); ok {
							summary.ToolsCalled = append(summary.ToolsCalled, toolStr)
						}
					}
				}
			}

			summaries = append(summaries, summary)
		}
	}

	return summaries, nil
}

func (p *ResultProcessor) generateContent(ctx context.Context, summaries []TaskResultSummary, format interfaces.OutputFormat) (string, error) {
	switch format {
	case interfaces.OutputFormatMarkdown:
		return p.generateMarkdownReport(summaries), nil
	case interfaces.OutputFormatJSON:
		return p.generateJSONReport(summaries)
	case interfaces.OutputFormatText:
		return p.generateTextReport(summaries), nil
	default:
		return p.generateMarkdownReport(summaries), nil
	}
}

func (p *ResultProcessor) generateMarkdownReport(summaries []TaskResultSummary) string {
	var content strings.Builder

	content.WriteString("# 任务执行报告\n\n")
	content.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 执行摘要
	total := len(summaries)
	successful := p.countSuccessfulTasks(summaries)
	failed := p.countFailedTasks(summaries)

	content.WriteString("## 执行摘要\n\n")
	content.WriteString(fmt.Sprintf("- **总任务数**: %d\n", total))
	content.WriteString(fmt.Sprintf("- **成功**: %d\n", successful))
	content.WriteString(fmt.Sprintf("- **失败**: %d\n", failed))
	if total > 0 {
		successRate := float64(successful) / float64(total) * 100
		content.WriteString(fmt.Sprintf("- **成功率**: %.1f%%\n", successRate))
	}
	content.WriteString("\n")

	// 任务详情
	content.WriteString("## 任务详情\n\n")
	for i, summary := range summaries {
		content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, summary.TaskName))
		content.WriteString(fmt.Sprintf("- **任务ID**: `%s`\n", summary.TaskID))
		content.WriteString(fmt.Sprintf("- **状态**: %s\n", p.getStatusEmoji(summary.Status)))

		if len(summary.ToolsCalled) > 0 {
			content.WriteString(fmt.Sprintf("- **使用工具**: %s\n", strings.Join(summary.ToolsCalled, ", ")))
		}

		if summary.Output != nil {
			content.WriteString("- **输出**:\n```\n")
			content.WriteString(fmt.Sprintf("%v", summary.Output))
			content.WriteString("\n```\n")
		}
		content.WriteString("\n")
	}

	return content.String()
}

func (p *ResultProcessor) generateJSONReport(summaries []TaskResultSummary) (string, error) {
	report := map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"summary": map[string]interface{}{
			"total_tasks":      len(summaries),
			"successful_tasks": p.countSuccessfulTasks(summaries),
			"failed_tasks":     p.countFailedTasks(summaries),
		},
		"tasks": summaries,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	return string(data), nil
}

func (p *ResultProcessor) generateTextReport(summaries []TaskResultSummary) string {
	var content strings.Builder

	content.WriteString("任务执行报告\n")
	content.WriteString("=" + strings.Repeat("=", 50) + "\n\n")
	content.WriteString(fmt.Sprintf("生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 执行摘要
	total := len(summaries)
	successful := p.countSuccessfulTasks(summaries)
	failed := p.countFailedTasks(summaries)

	content.WriteString("执行摘要:\n")
	content.WriteString(fmt.Sprintf("  总任务数: %d\n", total))
	content.WriteString(fmt.Sprintf("  成功: %d\n", successful))
	content.WriteString(fmt.Sprintf("  失败: %d\n", failed))
	if total > 0 {
		successRate := float64(successful) / float64(total) * 100
		content.WriteString(fmt.Sprintf("  成功率: %.1f%%\n", successRate))
	}
	content.WriteString("\n")

	// 任务详情
	content.WriteString("任务详情:\n")
	content.WriteString("-" + strings.Repeat("-", 50) + "\n")
	for i, summary := range summaries {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, summary.TaskName))
		content.WriteString(fmt.Sprintf("   任务ID: %s\n", summary.TaskID))
		content.WriteString(fmt.Sprintf("   状态: %s\n", summary.Status))

		if len(summary.ToolsCalled) > 0 {
			content.WriteString(fmt.Sprintf("   使用工具: %s\n", strings.Join(summary.ToolsCalled, ", ")))
		}

		content.WriteString("\n")
	}

	return content.String()
}

func (p *ResultProcessor) countSuccessfulTasks(summaries []TaskResultSummary) int {
	count := 0
	for _, summary := range summaries {
		if summary.Status == models.TaskStateSuccess {
			count++
		}
	}
	return count
}

func (p *ResultProcessor) countFailedTasks(summaries []TaskResultSummary) int {
	count := 0
	for _, summary := range summaries {
		if summary.Status == models.TaskStateFail {
			count++
		}
	}
	return count
}

func (p *ResultProcessor) collectToolsUsed(summaries []TaskResultSummary) []string {
	toolSet := make(map[string]bool)
	for _, summary := range summaries {
		for _, tool := range summary.ToolsCalled {
			toolSet[tool] = true
		}
	}

	var tools []string
	for tool := range toolSet {
		tools = append(tools, tool)
	}
	return tools
}

func (p *ResultProcessor) getStatusEmoji(status models.TaskState) string {
	switch status {
	case models.TaskStateSuccess:
		return "✅ 成功"
	case models.TaskStateFail:
		return "❌ 失败"
	case models.TaskStateRunning:
		return "🔄 运行中"
	case models.TaskStateWait:
		return "⏳ 等待"
	default:
		return "❓ 未知"
	}
}

func (p *ResultProcessor) fileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}

func (p *ResultProcessor) backupFile(filepath string) error {
	backupPath := filepath + ".backup." + time.Now().Format("20060102_150405")

	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(backupPath, content, 0644)
}

// 辅助函数
func getStringValue(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func filepath_Join(parts ...string) string {
	return filepath.Join(parts...)
}

func filepath_Dir(path string) string {
	return filepath.Dir(path)
}
