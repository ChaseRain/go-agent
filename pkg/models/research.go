package models

import (
	"time"
)

// ResearchReport 深度研究报告
type ResearchReport struct {
	ID               string          `json:"id"`
	Topic            string          `json:"topic"`
	ExecutiveSummary string          `json:"executive_summary"`
	Sections         []ReportSection `json:"sections"`
	Metadata         ReportMetadata  `json:"metadata"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// ReportSection 报告章节
type ReportSection struct {
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Type        string                 `json:"type"` // background, analysis, data, conclusion
	SubSections []ReportSubSection     `json:"sub_sections,omitempty"`
	Charts      []ChartData            `json:"charts,omitempty"`
	Tables      []TableData            `json:"tables,omitempty"`
	References  []string               `json:"references,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReportSubSection 报告子章节
type ReportSubSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   int    `json:"level"` // 1, 2, 3 等级
}

// ChartData 图表数据
type ChartData struct {
	ID      string                 `json:"id"`
	Title   string                 `json:"title"`
	Type    string                 `json:"type"` // line, bar, pie, scatter
	Data    map[string]interface{} `json:"data"`
	Options map[string]interface{} `json:"options"`
}

// TableData 表格数据
type TableData struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
	Footer  []string   `json:"footer,omitempty"`
}

// ReportMetadata 报告元数据
type ReportMetadata struct {
	Author       string   `json:"author"`
	Version      string   `json:"version"`
	Tags         []string `json:"tags"`
	Confidence   float64  `json:"confidence"` // 0-1 置信度
	DataSources  []string `json:"data_sources"`
	Methodology  string   `json:"methodology"`
	Limitations  string   `json:"limitations"`
	ReviewStatus string   `json:"review_status"` // draft, reviewed, final
}

// ResearchTask 研究任务
type ResearchTask struct {
	ID          string              `json:"id"`
	Type        ResearchTaskType    `json:"type"`
	Topic       string              `json:"topic"`
	Depth       int                 `json:"depth"`       // 1-5 研究深度
	Scope       string              `json:"scope"`       // narrow, medium, broad
	Constraints []string            `json:"constraints"` // 研究约束条件
	Deliverable ResearchDeliverable `json:"deliverable"`
	Status      string              `json:"status"`
	StartedAt   time.Time           `json:"started_at"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
}

// ResearchTaskType 研究任务类型
type ResearchTaskType string

const (
	ResearchTaskTypeMarket      ResearchTaskType = "market"      // 市场研究
	ResearchTaskTypeCompetitive ResearchTaskType = "competitive" // 竞争分析
	ResearchTaskTypeTechnical   ResearchTaskType = "technical"   // 技术研究
	ResearchTaskTypeFinancial   ResearchTaskType = "financial"   // 财务分析
	ResearchTaskTypeStrategic   ResearchTaskType = "strategic"   // 战略研究
	ResearchTaskTypeCustom      ResearchTaskType = "custom"      // 自定义研究
)

// ResearchDeliverable 研究交付物
type ResearchDeliverable struct {
	Format       string     `json:"format"`    // report, presentation, dashboard
	Length       string     `json:"length"`    // brief, standard, comprehensive
	Audiences    []string   `json:"audiences"` // executive, technical, general
	DueDate      *time.Time `json:"due_date,omitempty"`
	Requirements []string   `json:"requirements"`
}

// ResearchMethod 研究方法
type ResearchMethod struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Tools       []string `json:"tools"`
	DataSources []string `json:"data_sources"`
}

// ResearchFramework 研究框架
type ResearchFramework struct {
	Name        string         `json:"name"` // SWOT, Porter's Five Forces, etc
	Description string         `json:"description"`
	Components  []string       `json:"components"`
	Application ResearchMethod `json:"application"`
}

// DataPoint 数据点
type DataPoint struct {
	Label      string      `json:"label"`
	Value      interface{} `json:"value"`
	Unit       string      `json:"unit,omitempty"`
	Timestamp  *time.Time  `json:"timestamp,omitempty"`
	Source     string      `json:"source,omitempty"`
	Confidence float64     `json:"confidence"` // 0-1
}

// InsightItem 洞察项
type InsightItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // trend, anomaly, correlation, prediction
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"` // high, medium, low
	Evidence    []string  `json:"evidence"`
	Actions     []string  `json:"recommended_actions"`
	CreatedAt   time.Time `json:"created_at"`
}
