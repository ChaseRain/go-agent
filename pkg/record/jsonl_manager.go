package record

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-agent/pkg/interfaces"
)

// JSONLRecordManager 实现基于JSONL文件的记录管理器
type JSONLRecordManager struct {
	baseDir   string
	mu        sync.RWMutex
	fileCache map[string]*os.File
	config    *RecordConfig
}

// RecordConfig 记录管理器配置
type RecordConfig struct {
	BaseDir    string `json:"base_dir"`
	MaxRecords int    `json:"max_records"`
	EnableSSE  bool   `json:"enable_sse"`
}

// RecordEntry 记录条目结构
type RecordEntry struct {
	ID        string                 `json:"id"`
	Type      interfaces.RecordType  `json:"type"`
	ParentID  string                 `json:"parent_id,omitempty"`
	SessionID string                 `json:"session_id"`
	AgentID   string                 `json:"agent_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewJSONLRecordManager 创建新的JSONL记录管理器
func NewJSONLRecordManager(config *RecordConfig) *JSONLRecordManager {
	if config == nil {
		config = &RecordConfig{
			BaseDir:    "./records",
			MaxRecords: 10000,
			EnableSSE:  false,
		}
	}

	// 确保记录目录存在
	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create records directory: %v", err))
	}

	return &JSONLRecordManager{
		baseDir:   config.BaseDir,
		fileCache: make(map[string]*os.File),
		config:    config,
	}
}

// Record 记录执行记录
func (r *JSONLRecordManager) Record(recordType interfaces.RecordType, data interface{}) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 生成记录ID
	recordID := fmt.Sprintf("%s_%d", string(recordType), time.Now().UnixNano())

	// 提取元数据
	var sessionID, agentID, parentID string
	var recordData map[string]interface{}

	if dataMap, ok := data.(map[string]interface{}); ok {
		recordData = dataMap
		if sid, exists := dataMap["session_id"]; exists {
			if sidStr, ok := sid.(string); ok {
				sessionID = sidStr
			}
		}
		if aid, exists := dataMap["agent_id"]; exists {
			if aidStr, ok := aid.(string); ok {
				agentID = aidStr
			}
		}
		if pid, exists := dataMap["parent_id"]; exists {
			if pidStr, ok := pid.(string); ok {
				parentID = pidStr
			}
		}
	} else {
		recordData = map[string]interface{}{
			"raw_data": data,
		}
	}

	// 创建记录条目
	entry := RecordEntry{
		ID:        recordID,
		Type:      recordType,
		ParentID:  parentID,
		SessionID: sessionID,
		AgentID:   agentID,
		Timestamp: time.Now().UTC(),
		Data:      recordData,
	}

	// 写入文件
	if err := r.writeRecord(&entry); err != nil {
		return "", fmt.Errorf("failed to write record: %w", err)
	}

	return recordID, nil
}

// GetRecord 获取单个记录
func (r *JSONLRecordManager) GetRecord(recordID string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 遍历所有记录文件查找记录
	files, err := filepath.Glob(filepath.Join(r.baseDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list record files: %w", err)
	}

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry RecordEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}
			if entry.ID == recordID {
				return &entry, nil
			}
		}
	}

	return nil, fmt.Errorf("record not found: %s", recordID)
}

// GetSessionRecords 获取会话相关的所有记录
func (r *JSONLRecordManager) GetSessionRecords(sessionID string) ([]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var records []interface{}

	files, err := filepath.Glob(filepath.Join(r.baseDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list record files: %w", err)
	}

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry RecordEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}
			if entry.SessionID == sessionID {
				records = append(records, &entry)
			}
		}
	}

	return records, nil
}

// SaveSession 保存会话状态
func (r *JSONLRecordManager) SaveSession(sessionID string, data interface{}) error {
	_, err := r.Record(interfaces.RecordTypeAgentExecution, map[string]interface{}{
		"session_id": sessionID,
		"data":       data,
		"action":     "save",
	})
	return err
}

// LoadSession 加载会话状态
func (r *JSONLRecordManager) LoadSession(sessionID string) (interface{}, error) {
	records, err := r.GetSessionRecords(sessionID)
	if err != nil {
		return nil, err
	}

	// 查找最新的会话保存记录
	var latestSession *RecordEntry
	for _, record := range records {
		if entry, ok := record.(*RecordEntry); ok {
			if entry.Type == interfaces.RecordTypeAgentExecution {
				if action, exists := entry.Data["action"]; exists && action == "save" {
					if latestSession == nil || entry.Timestamp.After(latestSession.Timestamp) {
						latestSession = entry
					}
				}
			}
		}
	}

	if latestSession != nil {
		return latestSession.Data["data"], nil
	}

	return nil, fmt.Errorf("no session data found for session: %s", sessionID)
}

// Close 关闭记录管理器，清理资源
func (r *JSONLRecordManager) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, file := range r.fileCache {
		if err := file.Close(); err != nil {
			return err
		}
	}
	r.fileCache = make(map[string]*os.File)

	return nil
}

// 私有方法

func (r *JSONLRecordManager) writeRecord(entry *RecordEntry) error {
	// 根据日期和记录类型确定文件名
	fileName := fmt.Sprintf("%s_%s.jsonl",
		entry.Timestamp.Format("2006-01-02"),
		string(entry.Type))
	filePath := filepath.Join(r.baseDir, fileName)

	// 获取或创建文件
	file, err := r.getOrCreateFile(filePath)
	if err != nil {
		return err
	}

	// 序列化记录
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// 写入文件
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write record to file: %w", err)
	}

	// 强制刷新到磁盘
	return file.Sync()
}

func (r *JSONLRecordManager) getOrCreateFile(filePath string) (*os.File, error) {
	// 检查缓存
	if file, exists := r.fileCache[filePath]; exists {
		return file, nil
	}

	// 创建或打开文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	// 缓存文件句柄
	r.fileCache[filePath] = file
	return file, nil
}

// QueryRecords 查询记录（支持按类型、时间范围等过滤）
func (r *JSONLRecordManager) QueryRecords(ctx context.Context, filter RecordFilter) ([]RecordEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []RecordEntry

	files, err := filepath.Glob(filepath.Join(r.baseDir, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list record files: %w", err)
	}

	for _, filePath := range files {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		file, err := os.Open(filePath)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry RecordEntry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}

			if r.matchesFilter(&entry, &filter) {
				results = append(results, entry)
			}
		}
	}

	return results, nil
}

// RecordFilter 记录查询过滤器
type RecordFilter struct {
	SessionID string                `json:"session_id,omitempty"`
	AgentID   string                `json:"agent_id,omitempty"`
	Type      interfaces.RecordType `json:"type,omitempty"`
	StartTime *time.Time            `json:"start_time,omitempty"`
	EndTime   *time.Time            `json:"end_time,omitempty"`
	Limit     int                   `json:"limit,omitempty"`
}

func (r *JSONLRecordManager) matchesFilter(entry *RecordEntry, filter *RecordFilter) bool {
	if filter.SessionID != "" && entry.SessionID != filter.SessionID {
		return false
	}
	if filter.AgentID != "" && entry.AgentID != filter.AgentID {
		return false
	}
	if filter.Type != "" && entry.Type != filter.Type {
		return false
	}
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}
