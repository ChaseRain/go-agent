package main

import (
	"context"
	"fmt"
	"go-agent/pkg/agent"
	"go-agent/pkg/config"
	"go-agent/pkg/llm"
	"go-agent/pkg/models"
	"log"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create LLM provider
	llmConfig := &models.LLMConfig{
		Provider:    cfg.LLM.Provider,
		Model:       cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
		APIKey:      cfg.LLM.APIKey,
		BaseURL:     cfg.LLM.BaseURL,
	}
	provider, err := llm.NewProvider(cfg.LLM.Provider, llmConfig)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	// Create agent configuration
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

	// Create and initialize agent
	myAgent := agent.NewAgentWithProvider(provider)
	if err := myAgent.Initialize(agentConfig); err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	// Process a simple query
	ctx := context.Background()
	query := "Calculate the sum of 15 and 27, then multiply by 3"

	fmt.Printf("Query: %s\n\n", query)

	result, err := myAgent.ProcessMessage(ctx, query)
	if err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}

	// Display result
	fmt.Printf("Result:\n%s\n", result.Message)
	if result.OutputFile != "" {
		fmt.Printf("Output saved to: %s\n", result.OutputFile)
	}
}
