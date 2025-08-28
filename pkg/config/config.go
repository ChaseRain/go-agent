package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go-agent/pkg/models"
	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration
type Config struct {
	Agent     AgentConfig     `yaml:"agent" json:"agent"`
	LLM       LLMConfig       `yaml:"llm" json:"llm"`
	Execution ExecutionConfig `yaml:"execution" json:"execution"`
	Record    RecordConfig    `yaml:"record" json:"record"`
	Tools     ToolsConfig     `yaml:"tools" json:"tools"`
	Logging   LoggingConfig   `yaml:"logging" json:"logging"`
	Server    ServerConfig    `yaml:"server" json:"server"`
}

// AgentConfig represents agent-specific configuration
type AgentConfig struct {
	Name            string `yaml:"name" json:"name"`
	RoleDescription string `yaml:"role_description" json:"role_description"`
	MaxSteps        []int  `yaml:"max_steps" json:"max_steps"`
	MaxRounds       int    `yaml:"max_rounds" json:"max_rounds"`
	Parallel        bool   `yaml:"parallel" json:"parallel"`
	Stream          bool   `yaml:"stream" json:"stream"`
}

// LLMConfig represents LLM provider configuration
type LLMConfig struct {
	Provider    string  `yaml:"provider" json:"provider"`
	Model       string  `yaml:"model" json:"model"`
	Temperature float32 `yaml:"temperature" json:"temperature"`
	MaxTokens   int     `yaml:"max_tokens" json:"max_tokens"`
	APIKey      string  `yaml:"api_key" json:"api_key"`
	BaseURL     string  `yaml:"base_url" json:"base_url"`
}

// ExecutionConfig represents execution configuration
type ExecutionConfig struct {
	MaxWorkers  int    `yaml:"max_workers" json:"max_workers"`
	Timeout     int    `yaml:"timeout" json:"timeout"` // seconds
	SaveOutput  bool   `yaml:"save_output" json:"save_output"`
	OutputDir   string `yaml:"output_dir" json:"output_dir"`
}

// RecordConfig represents record management configuration
type RecordConfig struct {
	SaveDir     string `yaml:"save_dir" json:"save_dir"`
	EnableSSE   bool   `yaml:"enable_sse" json:"enable_sse"`
	MaxRecords  int    `yaml:"max_records" json:"max_records"`
}

// ToolsConfig represents tools configuration
type ToolsConfig struct {
	Enabled         []string `yaml:"enabled" json:"enabled"`
	CustomToolsDir  string   `yaml:"custom_tools_dir" json:"custom_tools_dir"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	File   string `yaml:"file" json:"file"`
	Format string `yaml:"format" json:"format"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Host           string `yaml:"host" json:"host"`
	Port           int    `yaml:"port" json:"port"`
	CORSEnabled    bool   `yaml:"cors_enabled" json:"cors_enabled"`
	MaxConnections int    `yaml:"max_connections" json:"max_connections"`
}

// ConfigManager manages configuration loading and validation
type ConfigManager struct {
	config     *Config
	configPath string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: DefaultConfig(),
	}
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:            "DefaultAgent",
			RoleDescription: "A helpful AI assistant",
			MaxSteps:        []int{5, 3, 2},
			MaxRounds:       10,
			Parallel:        false,
			Stream:          false,
		},
		LLM: LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			Temperature: 0.7,
			MaxTokens:   2000,
		},
		Execution: ExecutionConfig{
			MaxWorkers: 5,
			Timeout:    30,
			SaveOutput: true,
			OutputDir:  "./output",
		},
		Record: RecordConfig{
			SaveDir:    "./records",
			EnableSSE:  false,
			MaxRecords: 10000,
		},
		Tools: ToolsConfig{
			Enabled:        []string{"search", "calculate"},
			CustomToolsDir: "./tools",
		},
		Logging: LoggingConfig{
			Level:  "info",
			File:   "./logs/agent.log",
			Format: "json",
		},
		Server: ServerConfig{
			Enabled:        false,
			Host:           "localhost",
			Port:           8080,
			CORSEnabled:    true,
			MaxConnections: 100,
		},
	}
}

// LoadFromFile loads configuration from file
func (cm *ConfigManager) LoadFromFile(path string) error {
	cm.configPath = path
	
	// Read file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &cm.config)
	case ".json":
		err = json.Unmarshal(data, &cm.config)
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	cm.applyEnvOverrides()

	// Validate configuration
	return cm.Validate()
}

// LoadFromEnv loads configuration from environment variables
func (cm *ConfigManager) LoadFromEnv() {
	cm.applyEnvOverrides()
}

// applyEnvOverrides applies environment variable overrides
func (cm *ConfigManager) applyEnvOverrides() {
	// LLM configuration - 只有在配置文件中没有设置API key时才使用环境变量
	if cm.config.LLM.APIKey == "" {
		if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
			cm.config.LLM.APIKey = apiKey
		} else if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
			cm.config.LLM.APIKey = apiKey
		} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			cm.config.LLM.APIKey = apiKey
		}
	}
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		cm.config.LLM.BaseURL = baseURL
	}
	if baseURL := os.Getenv("DEEPSEEK_BASE_URL"); baseURL != "" {
		cm.config.LLM.BaseURL = baseURL
	}
	if baseURL := os.Getenv("LLM_BASE_URL"); baseURL != "" {
		cm.config.LLM.BaseURL = baseURL
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		cm.config.LLM.Model = model
	}
	
	// Agent configuration
	if agentName := os.Getenv("AGENT_NAME"); agentName != "" {
		cm.config.Agent.Name = agentName
	}
	
	// Server configuration
	if port := os.Getenv("PORT"); port != "" {
		var p int
		fmt.Sscanf(port, "%d", &p)
		if p > 0 {
			cm.config.Server.Port = p
		}
	}
}

// Validate validates the configuration
func (cm *ConfigManager) Validate() error {
	// Validate LLM configuration
	if cm.config.LLM.Provider == "" {
		return fmt.Errorf("LLM provider is required")
	}
	if cm.config.LLM.Model == "" {
		return fmt.Errorf("LLM model is required")
	}
	if cm.config.LLM.Provider != "mock" && cm.config.LLM.APIKey == "" {
		return fmt.Errorf("API key is required for provider %s", cm.config.LLM.Provider)
	}
	
	// Validate agent configuration
	if cm.config.Agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if len(cm.config.Agent.MaxSteps) == 0 {
		cm.config.Agent.MaxSteps = []int{5}
	}
	
	// Validate paths
	if cm.config.Execution.OutputDir == "" {
		cm.config.Execution.OutputDir = "./output"
	}
	if cm.config.Record.SaveDir == "" {
		cm.config.Record.SaveDir = "./records"
	}
	
	// Create directories if they don't exist
	dirs := []string{
		cm.config.Execution.OutputDir,
		cm.config.Record.SaveDir,
		filepath.Dir(cm.config.Logging.File),
	}
	
	for _, dir := range dirs {
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}
	
	return nil
}

// SaveToFile saves configuration to file
func (cm *ConfigManager) SaveToFile(path string) error {
	var data []byte
	var err error
	
	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cm.config)
	case ".json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// ToAgentConfig converts to models.AgentConfig
func (cm *ConfigManager) ToAgentConfig() *models.AgentConfig {
	return &models.AgentConfig{
		Name:            cm.config.Agent.Name,
		RoleDescription: cm.config.Agent.RoleDescription,
		MaxSteps:        cm.config.Agent.MaxSteps,
		MaxRounds:       cm.config.Agent.MaxRounds,
		Stream:          cm.config.Agent.Stream,
		Parallel:        cm.config.Agent.Parallel,
		LLMConfig: models.LLMConfig{
			Provider:    cm.config.LLM.Provider,
			Model:       cm.config.LLM.Model,
			Temperature: cm.config.LLM.Temperature,
			MaxTokens:   cm.config.LLM.MaxTokens,
			APIKey:      cm.config.LLM.APIKey,
			BaseURL:     cm.config.LLM.BaseURL,
		},
		Tools: cm.config.Tools.Enabled,
		CustomConfig: map[string]interface{}{
			"save_output":  cm.config.Execution.SaveOutput,
			"output_dir":   cm.config.Execution.OutputDir,
			"max_workers":  cm.config.Execution.MaxWorkers,
			"timeout":      cm.config.Execution.Timeout,
			"enable_sse":   cm.config.Record.EnableSSE,
			"record_dir":   cm.config.Record.SaveDir,
		},
	}
}

// UpdateFromAgentConfig updates configuration from models.AgentConfig
func (cm *ConfigManager) UpdateFromAgentConfig(ac *models.AgentConfig) {
	cm.config.Agent.Name = ac.Name
	cm.config.Agent.RoleDescription = ac.RoleDescription
	cm.config.Agent.MaxSteps = ac.MaxSteps
	cm.config.Agent.MaxRounds = ac.MaxRounds
	cm.config.Agent.Stream = ac.Stream
	cm.config.Agent.Parallel = ac.Parallel
	
	cm.config.LLM.Provider = ac.LLMConfig.Provider
	cm.config.LLM.Model = ac.LLMConfig.Model
	cm.config.LLM.Temperature = ac.LLMConfig.Temperature
	cm.config.LLM.MaxTokens = ac.LLMConfig.MaxTokens
	cm.config.LLM.APIKey = ac.LLMConfig.APIKey
	cm.config.LLM.BaseURL = ac.LLMConfig.BaseURL
	
	cm.config.Tools.Enabled = ac.Tools
}

// Load loads configuration from file (convenience function)
func Load(path string) (*Config, error) {
	cm := NewConfigManager()
	if err := cm.LoadFromFile(path); err != nil {
		return nil, err
	}
	cm.LoadFromEnv()
	if err := cm.Validate(); err != nil {
		return nil, err
	}
	return cm.GetConfig(), nil
}