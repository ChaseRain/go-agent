package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-agent/pkg/interfaces"
)

// StateManager 实现生产级状态管理器
type StateManager struct {
	stateDir     string
	mu           sync.RWMutex
	cache        map[string]*AgentState
	maxCacheSize int
	persistence  StatePersistence
}

// AgentState 代理状态结构
type AgentState struct {
	AgentID     string                 `json:"agent_id"`
	SessionID   string                 `json:"session_id"`
	Status      interfaces.AgentState  `json:"status"`
	LastUpdated time.Time              `json:"last_updated"`
	Context     *ExecutionContext      `json:"context"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     int                    `json:"version"`
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	CurrentTask    string                 `json:"current_task,omitempty"`
	TaskQueue      []string               `json:"task_queue,omitempty"`
	Variables      map[string]interface{} `json:"variables,omitempty"`
	MessageHistory []interface{}          `json:"message_history,omitempty"`
	ToolStates     map[string]interface{} `json:"tool_states,omitempty"`
}

// StatePersistence 状态持久化接口
type StatePersistence interface {
	Save(agentID string, state *AgentState) error
	Load(agentID string) (*AgentState, error)
	Delete(agentID string) error
	List() ([]string, error)
}

// FileStatePersistence 基于文件的状态持久化
type FileStatePersistence struct {
	baseDir string
}

// NewStateManager 创建新的状态管理器
func NewStateManager(stateDir string, maxCacheSize int) *StateManager {
	if stateDir == "" {
		stateDir = "./state"
	}
	if maxCacheSize <= 0 {
		maxCacheSize = 100
	}

	// 确保状态目录存在
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create state directory: %v", err))
	}

	persistence := &FileStatePersistence{baseDir: stateDir}

	return &StateManager{
		stateDir:     stateDir,
		cache:        make(map[string]*AgentState),
		maxCacheSize: maxCacheSize,
		persistence:  persistence,
	}
}

// SaveState 保存代理状态
func (s *StateManager) SaveState(agentID string, state interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agentState, err := s.convertToAgentState(agentID, state)
	if err != nil {
		return fmt.Errorf("failed to convert state: %w", err)
	}

	// 更新版本号
	if existing, exists := s.cache[agentID]; exists {
		agentState.Version = existing.Version + 1
	} else {
		agentState.Version = 1
	}

	agentState.LastUpdated = time.Now().UTC()

	// 保存到缓存
	s.cache[agentID] = agentState
	s.evictIfNeeded()

	// 持久化到存储
	if err := s.persistence.Save(agentID, agentState); err != nil {
		return fmt.Errorf("failed to persist state: %w", err)
	}

	return nil
}

// LoadState 加载代理状态
func (s *StateManager) LoadState(agentID string) (interface{}, error) {
	s.mu.RLock()

	// 首先检查缓存
	if state, exists := s.cache[agentID]; exists {
		s.mu.RUnlock()
		return state, nil
	}
	s.mu.RUnlock()

	// 从持久化存储加载
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.persistence.Load(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	if state != nil {
		// 添加到缓存
		s.cache[agentID] = state
		s.evictIfNeeded()
	}

	return state, nil
}

// DeleteState 删除代理状态
func (s *StateManager) DeleteState(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从缓存中删除
	delete(s.cache, agentID)

	// 从持久化存储中删除
	if err := s.persistence.Delete(agentID); err != nil {
		return fmt.Errorf("failed to delete persisted state: %w", err)
	}

	return nil
}

// ListStates 列出所有状态
func (s *StateManager) ListStates() ([]string, error) {
	return s.persistence.List()
}

// GetStateInfo 获取状态信息
func (s *StateManager) GetStateInfo(agentID string) (*StateInfo, error) {
	state, err := s.LoadState(agentID)
	if err != nil {
		return nil, err
	}

	if agentState, ok := state.(*AgentState); ok {
		return &StateInfo{
			AgentID:     agentState.AgentID,
			SessionID:   agentState.SessionID,
			Status:      agentState.Status,
			LastUpdated: agentState.LastUpdated,
			Version:     agentState.Version,
			HasContext:  agentState.Context != nil,
		}, nil
	}

	return nil, fmt.Errorf("invalid state format")
}

// StateInfo 状态信息摘要
type StateInfo struct {
	AgentID     string                `json:"agent_id"`
	SessionID   string                `json:"session_id"`
	Status      interfaces.AgentState `json:"status"`
	LastUpdated time.Time             `json:"last_updated"`
	Version     int                   `json:"version"`
	HasContext  bool                  `json:"has_context"`
}

// UpdateStateStatus 更新状态
func (s *StateManager) UpdateStateStatus(agentID string, status interfaces.AgentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var state *AgentState
	var exists bool

	// 检查缓存
	if state, exists = s.cache[agentID]; !exists {
		// 从持久化存储加载
		loadedState, err := s.persistence.Load(agentID)
		if err != nil {
			// 如果不存在，创建新状态
			state = &AgentState{
				AgentID: agentID,
				Status:  status,
				Version: 1,
			}
		} else {
			state = loadedState
		}
	}

	// 更新状态
	state.Status = status
	state.LastUpdated = time.Now().UTC()
	state.Version++

	// 保存到缓存
	s.cache[agentID] = state
	s.evictIfNeeded()

	// 持久化
	return s.persistence.Save(agentID, state)
}

// SetContext 设置执行上下文
func (s *StateManager) SetContext(agentID string, ctx *ExecutionContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var state *AgentState
	var exists bool

	// 获取或创建状态
	if state, exists = s.cache[agentID]; !exists {
		loadedState, err := s.persistence.Load(agentID)
		if err != nil {
			state = &AgentState{
				AgentID: agentID,
				Version: 1,
			}
		} else {
			state = loadedState
		}
	}

	// 更新上下文
	state.Context = ctx
	state.LastUpdated = time.Now().UTC()
	state.Version++

	// 保存
	s.cache[agentID] = state
	s.evictIfNeeded()

	return s.persistence.Save(agentID, state)
}

// GetContext 获取执行上下文
func (s *StateManager) GetContext(agentID string) (*ExecutionContext, error) {
	state, err := s.LoadState(agentID)
	if err != nil {
		return nil, err
	}

	if agentState, ok := state.(*AgentState); ok {
		return agentState.Context, nil
	}

	return nil, fmt.Errorf("invalid state format")
}

// CleanupOldStates 清理过期状态
func (s *StateManager) CleanupOldStates(ctx context.Context, maxAge time.Duration) error {
	states, err := s.ListStates()
	if err != nil {
		return fmt.Errorf("failed to list states: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for _, agentID := range states {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, err := s.GetStateInfo(agentID)
		if err != nil {
			continue
		}

		if info.LastUpdated.Before(cutoff) {
			if err := s.DeleteState(agentID); err != nil {
				fmt.Printf("Warning: failed to delete old state %s: %v\n", agentID, err)
			} else {
				cleaned++
			}
		}
	}

	fmt.Printf("Cleaned up %d old states\n", cleaned)
	return nil
}

// 私有方法

func (s *StateManager) convertToAgentState(agentID string, state interface{}) (*AgentState, error) {
	if agentState, ok := state.(*AgentState); ok {
		agentState.AgentID = agentID
		return agentState, nil
	}

	// 尝试从通用格式转换
	agentState := &AgentState{
		AgentID:  agentID,
		Status:   interfaces.AgentStateIdle,
		Metadata: make(map[string]interface{}),
	}

	if stateMap, ok := state.(map[string]interface{}); ok {
		// 提取常用字段
		if sessionID, exists := stateMap["session_id"]; exists {
			if sid, ok := sessionID.(string); ok {
				agentState.SessionID = sid
			}
		}

		if status, exists := stateMap["status"]; exists {
			if statusStr, ok := status.(string); ok {
				agentState.Status = interfaces.AgentState(statusStr)
			}
		}

		// 其余作为元数据
		agentState.Metadata = stateMap
	} else {
		// 直接作为元数据存储
		agentState.Metadata["data"] = state
	}

	return agentState, nil
}

func (s *StateManager) evictIfNeeded() {
	if len(s.cache) <= s.maxCacheSize {
		return
	}

	// 简单的LRU淘汰策略：删除最旧的状态
	var oldestID string
	var oldestTime time.Time
	first := true

	for id, state := range s.cache {
		if first || state.LastUpdated.Before(oldestTime) {
			oldestID = id
			oldestTime = state.LastUpdated
			first = false
		}
	}

	if oldestID != "" {
		delete(s.cache, oldestID)
	}
}

// FileStatePersistence 实现

func (f *FileStatePersistence) Save(agentID string, state *AgentState) error {
	filePath := filepath.Join(f.baseDir, agentID+".json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return ioutil.WriteFile(filePath, data, 0644)
}

func (f *FileStatePersistence) Load(agentID string) (*AgentState, error) {
	filePath := filepath.Join(f.baseDir, agentID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // 状态不存在
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

func (f *FileStatePersistence) Delete(agentID string) error {
	filePath := filepath.Join(f.baseDir, agentID+".json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，认为删除成功
	}

	return os.Remove(filePath)
}

func (f *FileStatePersistence) List() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(f.baseDir, "*.json"))
	if err != nil {
		return nil, err
	}

	var agentIDs []string
	for _, file := range files {
		base := filepath.Base(file)
		agentID := base[:len(base)-5] // 移除 .json 后缀
		agentIDs = append(agentIDs, agentID)
	}

	return agentIDs, nil
}
