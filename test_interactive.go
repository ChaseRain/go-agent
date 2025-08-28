package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"go-agent/pkg/agent"
	"go-agent/pkg/config"
	"go-agent/pkg/models"
)

func main() {
	fmt.Println("测试DeepSeek API交互式对话")
	fmt.Println(strings.Repeat("=", 40))

	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 配置LLM
	llmConfig := &models.LLMConfig{
		Provider:    cfg.LLM.Provider,
		Model:       cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
		APIKey:      cfg.LLM.APIKey,
		BaseURL:     cfg.LLM.BaseURL,
	}

	// 配置代理
	agentConfig := &models.AgentConfig{
		Name:            cfg.Agent.Name,
		RoleDescription: cfg.Agent.RoleDescription,
		MaxSteps:        cfg.Agent.MaxSteps,
		MaxRounds:       cfg.Agent.MaxRounds,
		Parallel:        cfg.Agent.Parallel,
		LLMConfig:       *llmConfig,
		Tools:           cfg.Tools.Enabled,
		OutputDir:       cfg.Execution.OutputDir,
	}

	// 创建代理
	myAgent := agent.NewDynAgent(agentConfig)
	ctx := context.Background()

	// 预定义的测试对话
	testQueries := []string{
		"你好，请简单介绍一下自己",
		"你能做什么？",
		"请帮我计算 25 * 16 + 100",
		"写一首关于春天的短诗",
		"解释一下什么是人工智能",
	}

	// 测试每个查询
	for i, query := range testQueries {
		fmt.Printf("\n[测试 %d] 用户: %s\n", i+1, query)
		fmt.Println(strings.Repeat("-", 50))

		result, err := myAgent.ProcessMessage(ctx, query)
		if err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			continue
		}

		fmt.Printf("🤖 助手: %s\n", result.Message)
		
		if len(result.AllFiles) > 0 {
			fmt.Printf("📁 输出文件: %v\n", result.AllFiles)
		}
	}

	fmt.Println("\n✅ 测试完成！")
}