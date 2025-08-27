package main

import (
	"context"
	"fmt"
	"log"
	"go-agent/pkg/agent"
	"go-agent/pkg/config"
	"go-agent/pkg/llm"
	"go-agent/pkg/models"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create LLM provider
	provider, err := llm.NewProvider(cfg.LLM.Provider, &cfg.LLM)
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
		LLMConfig:       cfg.LLM,
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