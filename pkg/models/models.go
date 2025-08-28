package models

import (
	"time"
)

// Message represents a conversation message
type Message struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
	Model            string `json:"model,omitempty"`
	Type             string `json:"type,omitempty"`
}

// SubTask represents a task to be executed
type SubTask struct {
	ID          string                 `json:"sub_task_id"`
	Name        string                 `json:"sub_task_name"`
	Description string                 `json:"sub_task_describe"`
	Process     string                 `json:"process"`
	Type        string                 `json:"sub_task_type"`
	Dependent   string                 `json:"dependent"`
	OutputFile  string                 `json:"output_md_file"`
	State       TaskState              `json:"state"`
	StateMsg    string                 `json:"state_msg"`
	GenAgentID  string                 `json:"gen_agent_id,omitempty"`
	RecordID    string                 `json:"record_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // 扩展属性，支持灵活的任务配置
}

// TaskState represents the state of a task
type TaskState string

const (
	TaskStateWait    TaskState = "wait"
	TaskStateRunning TaskState = "running"
	TaskStateSuccess TaskState = "success"
	TaskStateFail    TaskState = "fail"
)

// SubTaskType represents different types of subtasks
type SubTaskType string

const (
	SubTaskTypeTask      SubTaskType = "task"
	SubTaskTypeFunction  SubTaskType = "function"
	SubTaskTypeAgentCall SubTaskType = "agent_call"
	SubTaskTypeAgentGen  SubTaskType = "agent_gen"
)

// ProcessMessageResult represents the result of processing a message
type ProcessMessageResult struct {
	Code               int      `json:"code"`
	Message            string   `json:"msg"`
	OutputFile         string   `json:"output_md_file"`
	OutputTextAbstract string   `json:"output_text_abstract"`
	AllFiles           []string `json:"all_files"`
	Data               any      `json:"data,omitempty"`
}

// PlanningResult represents the result of task planning
type PlanningResult struct {
	Tasks        []SubTask              `json:"tasks"`
	Dependencies map[string][]string    `json:"dependencies"`
	Summary      string                 `json:"summary"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"` // 规划元数据，如执行流程、风险评估等
}

// ExecutionContext represents the context for task execution
type ExecutionContext struct {
	AgentID        string
	AgentName      string
	AgentChain     []string
	SessionID      string
	ParentRecordID string
	Messages       []Message
	Config         *AgentConfig
}

// AgentConfig represents agent configuration
type AgentConfig struct {
	Name            string                 `json:"name"`
	RoleDescription string                 `json:"role_description"`
	MaxSteps        []int                  `json:"max_steps"`
	MaxRounds       int                    `json:"max_rounds"`
	Stream          bool                   `json:"stream"`
	Parallel        bool                   `json:"parallel"`
	LLMConfig       LLMConfig              `json:"llm_config"`
	Tools           []string               `json:"tools"`
	SaveOutput      bool                   `json:"save_output"`
	OutputDir       string                 `json:"output_dir"`
	CustomConfig    map[string]interface{} `json:"custom_config"`
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	APIKey      string  `json:"api_key,omitempty"`
	BaseURL     string  `json:"base_url,omitempty"`
}
