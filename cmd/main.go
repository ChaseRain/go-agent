package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"go-agent/pkg/agent"
	"go-agent/pkg/config"
	"go-agent/pkg/llm"
	"go-agent/pkg/models"
)

func main() {
	// Command line flags
	configFile := flag.String("config", "config.yaml", "Configuration file path")
	interactive := flag.Bool("i", true, "Interactive mode")
	query := flag.String("q", "", "Query to process (non-interactive mode)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(cfg.Execution.OutputDir, 0755); err != nil {
		log.Printf("Failed to create output directory: %v", err)
	}

	// Create logs directory
	if logDir := "logs"; cfg.Logging.File != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
		}
	}

	// Create LLM config for models package
	llmConfig := &models.LLMConfig{
		Provider:    cfg.LLM.Provider,
		Model:       cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
		APIKey:      cfg.LLM.APIKey,
		BaseURL:     cfg.LLM.BaseURL,
	}

	// Create LLM provider
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

	ctx := context.Background()

	// Process based on mode
	if *query != "" {
		// Non-interactive mode: process single query
		processQuery(ctx, myAgent, *query)
	} else if *interactive {
		// Interactive mode
		runInteractive(ctx, myAgent)
	} else {
		fmt.Println("Use -q for single query or -i for interactive mode")
		flag.Usage()
	}
}

func runInteractive(ctx context.Context, agent *agent.AgentWithProvider) {
	fmt.Println("╭─────────────────────────────────────╮")
	fmt.Println("│        Go Agent Framework           │")
	fmt.Println("├─────────────────────────────────────┤")
	fmt.Println("│  Commands:                          │")
	fmt.Println("│  • exit/quit - Exit the program    │")
	fmt.Println("│  • clear - Clear screen            │")
	fmt.Println("│  • help - Show this message        │")
	fmt.Println("╰─────────────────────────────────────╯")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("\nGoodbye!")
			return
		case "clear":
			clearScreen()
			continue
		case "help":
			showHelp()
			continue
		}

		// Process the query
		processQuery(ctx, agent, input)
		fmt.Println()
	}
}

func processQuery(ctx context.Context, agent *agent.AgentWithProvider, query string) {
	fmt.Printf("\nProcessing: %s\n", query)
	fmt.Println(strings.Repeat("-", 50))

	result, err := agent.ProcessMessage(ctx, query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nAgent> %s\n", result.Message)

	if result.OutputFile != "" {
		fmt.Printf("\nOutput saved to: %s\n", result.OutputFile)
	}
}

func clearScreen() {
	// Clear screen command for Unix-like systems
	fmt.Print("\033[H\033[2J")
	fmt.Println("Screen cleared.")
}

func showHelp() {
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("                  HELP                 ")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  exit, quit - Exit the program")
	fmt.Println("  clear      - Clear the screen")
	fmt.Println("  help       - Show this help message")
	fmt.Println()
	fmt.Println("Just type your question or task to interact with the agent.")
	fmt.Println()
}
