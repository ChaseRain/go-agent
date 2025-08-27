package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// SearchTool implements web search functionality
type SearchTool struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewSearchTool creates a new search tool
func NewSearchTool(apiKey string) *SearchTool {
	return &SearchTool{
		apiKey:  apiKey,
		baseURL: "https://api.search.example.com", // Replace with actual API
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetName returns the tool name
func (t *SearchTool) GetName() string {
	return "web_search"
}

// GetDescription returns the tool description
func (t *SearchTool) GetDescription() string {
	return "Search the web for information using keywords"
}

// Execute performs a web search
func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}
	
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	
	// For demo, return mock results
	// In production, would call actual search API
	results := t.mockSearch(query, limit)
	
	return map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	}, nil
}

// ValidateArgs validates the arguments
func (t *SearchTool) ValidateArgs(args map[string]interface{}) error {
	if _, ok := args["query"]; !ok {
		return fmt.Errorf("missing required parameter: query")
	}
	return nil
}

func (t *SearchTool) mockSearch(query string, limit int) []map[string]interface{} {
	// Mock search results
	results := make([]map[string]interface{}, 0, limit)
	
	for i := 0; i < limit && i < 5; i++ {
		results = append(results, map[string]interface{}{
			"title":       fmt.Sprintf("Result %d for '%s'", i+1, query),
			"url":         fmt.Sprintf("https://example.com/result%d", i+1),
			"snippet":     fmt.Sprintf("This is a snippet for search result %d related to %s...", i+1, query),
			"relevance":   1.0 - float64(i)*0.1,
		})
	}
	
	return results
}

// realSearch performs actual API search (example implementation)
func (t *SearchTool) realSearch(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	// Build request URL
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("api_key", t.apiKey)
	
	requestURL := fmt.Sprintf("%s/search?%s", t.baseURL, params.Encode())
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Execute request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse response
	var result struct {
		Results []map[string]interface{} `json:"results"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return result.Results, nil
}