package tools

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FileTool implements file operations
type FileTool struct {
	allowedPaths []string
}

// NewFileTool creates a new file tool
func NewFileTool(allowedPaths []string) *FileTool {
	return &FileTool{
		allowedPaths: allowedPaths,
	}
}

// GetName returns the tool name
func (t *FileTool) GetName() string {
	return "file_operations"
}

// GetDescription returns the tool description
func (t *FileTool) GetDescription() string {
	return "Read, write, and manipulate files (text, JSON, CSV)"
}

// Execute performs file operations
func (t *FileTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}
	
	switch operation {
	case "read":
		return t.readFile(args)
	case "write":
		return t.writeFile(args)
	case "list":
		return t.listFiles(args)
	case "delete":
		return t.deleteFile(args)
	case "parse_csv":
		return t.parseCSV(args)
	case "parse_json":
		return t.parseJSON(args)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// ValidateArgs validates the arguments
func (t *FileTool) ValidateArgs(args map[string]interface{}) error {
	if _, ok := args["operation"]; !ok {
		return fmt.Errorf("missing required parameter: operation")
	}
	
	operation := args["operation"].(string)
	switch operation {
	case "read", "write", "delete", "parse_csv", "parse_json":
		if _, ok := args["path"]; !ok {
			return fmt.Errorf("missing required parameter: path")
		}
	case "list":
		if _, ok := args["directory"]; !ok {
			return fmt.Errorf("missing required parameter: directory")
		}
	}
	
	return nil
}

func (t *FileTool) readFile(args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}
	
	if !t.isPathAllowed(path) {
		return nil, fmt.Errorf("access to path %s is not allowed", path)
	}
	
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}, nil
}

func (t *FileTool) writeFile(args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}
	
	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required")
	}
	
	if !t.isPathAllowed(path) {
		return nil, fmt.Errorf("access to path %s is not allowed", path)
	}
	
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	
	return map[string]interface{}{
		"path":    path,
		"size":    len(content),
		"message": "File written successfully",
	}, nil
}

func (t *FileTool) listFiles(args map[string]interface{}) (interface{}, error) {
	directory, ok := args["directory"].(string)
	if !ok {
		return nil, fmt.Errorf("directory parameter is required")
	}
	
	if !t.isPathAllowed(directory) {
		return nil, fmt.Errorf("access to directory %s is not allowed", directory)
	}
	
	pattern := "*"
	if p, ok := args["pattern"].(string); ok {
		pattern = p
	}
	
	files, err := filepath.Glob(filepath.Join(directory, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	fileInfo := make([]map[string]interface{}, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		fileInfo = append(fileInfo, map[string]interface{}{
			"path":     file,
			"name":     info.Name(),
			"size":     info.Size(),
			"is_dir":   info.IsDir(),
			"modified": info.ModTime(),
		})
	}
	
	return map[string]interface{}{
		"directory": directory,
		"files":     fileInfo,
		"count":     len(fileInfo),
	}, nil
}

func (t *FileTool) deleteFile(args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}
	
	if !t.isPathAllowed(path) {
		return nil, fmt.Errorf("access to path %s is not allowed", path)
	}
	
	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}
	
	return map[string]interface{}{
		"path":    path,
		"message": "File deleted successfully",
	}, nil
}

func (t *FileTool) parseCSV(args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}
	
	if !t.isPathAllowed(path) {
		return nil, fmt.Errorf("access to path %s is not allowed", path)
	}
	
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	
	// Convert to structured data
	var headers []string
	var data []map[string]interface{}
	
	if len(records) > 0 {
		headers = records[0]
		
		for i := 1; i < len(records); i++ {
			row := make(map[string]interface{})
			for j, value := range records[i] {
				if j < len(headers) {
					row[headers[j]] = value
				}
			}
			data = append(data, row)
		}
	}
	
	return map[string]interface{}{
		"path":    path,
		"headers": headers,
		"data":    data,
		"rows":    len(data),
	}, nil
}

func (t *FileTool) parseJSON(args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path parameter is required")
	}
	
	if !t.isPathAllowed(path) {
		return nil, fmt.Errorf("access to path %s is not allowed", path)
	}
	
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}
	
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return map[string]interface{}{
		"path": path,
		"data": data,
	}, nil
}

func (t *FileTool) isPathAllowed(path string) bool {
	if len(t.allowedPaths) == 0 {
		// No restrictions if no allowed paths specified
		return true
	}
	
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	
	for _, allowed := range t.allowedPaths {
		absAllowed, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(absPath, absAllowed) {
			return true
		}
	}
	
	return false
}