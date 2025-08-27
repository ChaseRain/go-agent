package record

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/interfaces"
)

// RecordManager implements record management with file-based storage
type RecordManager struct {
	saveDir       string
	sessionID     string
	mu            sync.RWMutex
	records       []Record
	recordIndex   map[string]*Record
	agentChain    []string
	parentStack   []string
}

// Record represents a single record entry
type Record struct {
	RecordID       string                 `json:"record_id"`
	RecordType     interfaces.RecordType  `json:"record_type"`
	ParentRecordID string                 `json:"parent_record_id,omitempty"`
	AgentChain     []string               `json:"agent_chain,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Data           map[string]interface{} `json:"data"`
}

// NewRecordManager creates a new RecordManager instance
func NewRecordManager(saveDir string, sessionID string) *RecordManager {
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().Unix())
	}
	
	manager := &RecordManager{
		saveDir:     saveDir,
		sessionID:   sessionID,
		records:     make([]Record, 0),
		recordIndex: make(map[string]*Record),
		agentChain:  make([]string, 0),
		parentStack: make([]string, 0),
	}
	
	// Create save directory if it doesn't exist
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create save directory: %v\n", err)
	}
	
	// Load existing records if any
	manager.loadRecords()
	
	return manager
}

// Record creates a new record
func (rm *RecordManager) Record(recordType interfaces.RecordType, data interface{}) (string, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	recordID := uuid.New().String()
	
	// Convert data to map if necessary
	dataMap, err := rm.convertToMap(data)
	if err != nil {
		return "", fmt.Errorf("failed to convert data: %w", err)
	}
	
	// Get parent ID from stack or data
	parentID := ""
	if len(rm.parentStack) > 0 {
		parentID = rm.parentStack[len(rm.parentStack)-1]
	}
	if pid, ok := dataMap["parent_id"].(string); ok && pid != "" {
		parentID = pid
	}
	
	// Create record
	record := Record{
		RecordID:       recordID,
		RecordType:     recordType,
		ParentRecordID: parentID,
		AgentChain:     append([]string{}, rm.agentChain...), // Copy slice
		Timestamp:      time.Now(),
		Data:           dataMap,
	}
	
	// Add to records
	rm.records = append(rm.records, record)
	rm.recordIndex[recordID] = &record
	
	// Update parent stack for hierarchical recording
	if recordType == interfaces.RecordTypeAgentExecution || 
	   recordType == interfaces.RecordTypePlanning ||
	   recordType == interfaces.RecordTypeSubtask {
		if status, ok := dataMap["status"].(string); ok && status == "started" {
			rm.parentStack = append(rm.parentStack, recordID)
		} else if status == "completed" || status == "failed" {
			// Pop from stack
			if len(rm.parentStack) > 0 && rm.parentStack[len(rm.parentStack)-1] == recordID {
				rm.parentStack = rm.parentStack[:len(rm.parentStack)-1]
			}
		}
	}
	
	// Save to file (async to avoid blocking)
	go rm.saveRecord(record)
	
	return recordID, nil
}

// GetRecord retrieves a record by ID
func (rm *RecordManager) GetRecord(recordID string) (interface{}, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	if record, ok := rm.recordIndex[recordID]; ok {
		return record, nil
	}
	
	return nil, fmt.Errorf("record not found: %s", recordID)
}

// GetSessionRecords retrieves all records for a session
func (rm *RecordManager) GetSessionRecords(sessionID string) ([]interface{}, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	results := make([]interface{}, 0)
	for _, record := range rm.records {
		// Filter by session if needed (would need session tracking in records)
		results = append(results, record)
	}
	
	return results, nil
}

// SaveSession saves the current session state
func (rm *RecordManager) SaveSession(sessionID string, data interface{}) error {
	sessionFile := filepath.Join(rm.saveDir, fmt.Sprintf("%s_session.json", sessionID))
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}
	
	if err := os.WriteFile(sessionFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	
	return nil
}

// LoadSession loads a session state
func (rm *RecordManager) LoadSession(sessionID string) (interface{}, error) {
	sessionFile := filepath.Join(rm.saveDir, fmt.Sprintf("%s_session.json", sessionID))
	
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No session file exists
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}
	
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}
	
	return result, nil
}

// SetAgentChain sets the current agent chain
func (rm *RecordManager) SetAgentChain(chain []string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.agentChain = chain
}

// GetHierarchicalRecords returns records organized hierarchically
func (rm *RecordManager) GetHierarchicalRecords() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// Build hierarchy
	hierarchy := make(map[string]interface{})
	rootRecords := make([]Record, 0)
	childrenMap := make(map[string][]Record)
	
	// Group records by parent
	for _, record := range rm.records {
		if record.ParentRecordID == "" {
			rootRecords = append(rootRecords, record)
		} else {
			childrenMap[record.ParentRecordID] = append(childrenMap[record.ParentRecordID], record)
		}
	}
	
	// Build tree structure
	hierarchy["root"] = rm.buildRecordTree(rootRecords, childrenMap)
	hierarchy["total_records"] = len(rm.records)
	hierarchy["session_id"] = rm.sessionID
	
	return hierarchy
}

// Private methods

func (rm *RecordManager) convertToMap(data interface{}) (map[string]interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		return v, nil
	case Record:
		// Convert Record to map
		jsonData, _ := json.Marshal(v)
		var result map[string]interface{}
		json.Unmarshal(jsonData, &result)
		return result, nil
	default:
		// Try JSON marshal/unmarshal for other types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		var result map[string]interface{}
		if err := json.Unmarshal(jsonData, &result); err != nil {
			// If unmarshal fails, wrap in a map
			return map[string]interface{}{
				"value": data,
			}, nil
		}
		return result, nil
	}
}

func (rm *RecordManager) saveRecord(record Record) {
	// Save to JSONL file
	recordFile := filepath.Join(rm.saveDir, "records.jsonl")
	
	file, err := os.OpenFile(recordFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: failed to open record file: %v\n", err)
		return
	}
	defer file.Close()
	
	jsonData, err := json.Marshal(record)
	if err != nil {
		fmt.Printf("Warning: failed to marshal record: %v\n", err)
		return
	}
	
	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		fmt.Printf("Warning: failed to write record: %v\n", err)
	}
}

func (rm *RecordManager) loadRecords() {
	recordFile := filepath.Join(rm.saveDir, "records.jsonl")
	
	file, err := os.Open(recordFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to open record file: %v\n", err)
		}
		return
	}
	defer file.Close()
	
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var record Record
		if err := decoder.Decode(&record); err != nil {
			fmt.Printf("Warning: failed to decode record: %v\n", err)
			continue
		}
		rm.records = append(rm.records, record)
		rm.recordIndex[record.RecordID] = &record
	}
}

func (rm *RecordManager) buildRecordTree(records []Record, childrenMap map[string][]Record) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	
	for _, record := range records {
		node := map[string]interface{}{
			"record":  record,
			"children": []map[string]interface{}{},
		}
		
		if children, ok := childrenMap[record.RecordID]; ok {
			node["children"] = rm.buildRecordTree(children, childrenMap)
		}
		
		result = append(result, node)
	}
	
	return result
}