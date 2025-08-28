package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-agent/pkg/agent"
	"go-agent/pkg/config"
	"go-agent/pkg/models"
)

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Message string `json:"message"`
}

// WebServer Web服务器结构
type WebServer struct {
	streamingAgent *agent.StreamingAgent
	clients        map[string]chan agent.StreamEvent
	mu             sync.RWMutex
}

// NewWebServer 创建Web服务器
func NewWebServer(streamingAgent *agent.StreamingAgent) *WebServer {
	ws := &WebServer{
		streamingAgent: streamingAgent,
		clients:        make(map[string]chan agent.StreamEvent),
	}
	
	// 启动事件分发器
	go ws.eventDispatcher()
	
	return ws
}

// eventDispatcher 事件分发器
func (ws *WebServer) eventDispatcher() {
	eventChan := ws.streamingAgent.GetEventChannel()
	for event := range eventChan {
		ws.mu.RLock()
		for _, clientChan := range ws.clients {
			select {
			case clientChan <- event:
			default:
				// 防止阻塞
			}
		}
		ws.mu.RUnlock()
	}
}

// handleSSE 处理SSE连接
func (ws *WebServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// 设置SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// 创建客户端通道
	clientID := uuid.New().String()
	clientChan := make(chan agent.StreamEvent, 100)
	
	// 注册客户端
	ws.mu.Lock()
	ws.clients[clientID] = clientChan
	ws.mu.Unlock()
	
	// 清理函数
	defer func() {
		ws.mu.Lock()
		delete(ws.clients, clientID)
		close(clientChan)
		ws.mu.Unlock()
	}()
	
	// 发送初始连接事件
	fmt.Fprintf(w, "event: connected\ndata: {\"message\":\"Connected to agent stream\"}\n\n")
	w.(http.Flusher).Flush()
	
	// 心跳定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	// 持续发送事件
	for {
		select {
		case event := <-clientChan:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			w.(http.Flusher).Flush()
			
		case <-ticker.C:
			// 发送心跳
			fmt.Fprintf(w, "event: ping\ndata: {\"time\":%d}\n\n", time.Now().Unix())
			w.(http.Flusher).Flush()
			
		case <-r.Context().Done():
			// 客户端断开连接
			return
		}
	}
}

// handleChat 处理聊天请求
func (ws *WebServer) handleChat(w http.ResponseWriter, r *http.Request) {
	// CORS处理
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	// 创建请求ID
	requestID := uuid.New().String()
	
	// 异步处理消息
	go func() {
		ctx := context.Background()
		
		// 调用流式处理方法
		result, err := ws.streamingAgent.ProcessMessageStream(ctx, req.Message, requestID)
		
		if err != nil {
			log.Printf("处理消息失败: %v", err)
		} else {
			log.Printf("处理完成: %+v", result)
		}
	}()
	
	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"request_id": requestID,
		"status":     "processing",
	})
}

// handleStatic 提供静态文件
func handleStatic(w http.ResponseWriter, r *http.Request) {
	// 如果是根路径，返回index.html
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "web/index.html")
		return
	}
	
	// 其他静态文件
	http.ServeFile(w, r, filepath.Join("web", r.URL.Path))
}

func main() {
	// 加载配置
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	
	// 转换配置
	agentConfig := &models.AgentConfig{
		Name:          cfg.Agent.Name,
		RoleDescription: cfg.Agent.RoleDescription,
		MaxSteps:      cfg.Agent.MaxSteps,
		MaxRounds:     cfg.Agent.MaxRounds,
		Parallel:      cfg.Agent.Parallel,
		LLMConfig:     models.LLMConfig{
			Provider:    cfg.LLM.Provider,
			Model:       cfg.LLM.Model,
			APIKey:      cfg.LLM.APIKey,
			BaseURL:     cfg.LLM.BaseURL,
			Temperature: cfg.LLM.Temperature,
			MaxTokens:   cfg.LLM.MaxTokens,
		},
		CustomConfig:  make(map[string]interface{}),
	}
	
	// 创建流式代理
	streamingAgent := agent.NewStreamingAgent(agentConfig)
	
	// 创建Web服务器
	webServer := NewWebServer(streamingAgent)
	
	// 设置路由
	http.HandleFunc("/", handleStatic)
	http.HandleFunc("/api/chat", webServer.handleChat)
	http.HandleFunc("/api/stream", webServer.handleSSE)
	
	// 启动服务器
	port := ":8080"
	fmt.Printf("🚀 Agent Web 服务器已启动: http://localhost%s\n", port)
	fmt.Println("📝 请在浏览器中打开 http://localhost:8080")
	fmt.Println("💡 使用 DeepSeek V3 模型进行智能对话")
	fmt.Println("🎯 支持任务规划、分解和流式执行")
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}