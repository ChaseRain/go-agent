package main

import (
	"fmt"
	"go-agent/pkg/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("API Key: %s\n", cfg.LLM.APIKey)
	fmt.Printf("API Key length: %d\n", len(cfg.LLM.APIKey))
}
