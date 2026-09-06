package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/full"

	embeddedllm "github.com/benmizrahi/adk-embedded-litert"
)

func main() {
	ctx := context.Background()

	modelPath := os.Getenv("LITERT_MODEL_PATH")
	if modelPath == "" {
		if _, err := os.Stat("gemma3-270m-it-q8.litertlm"); err == nil {
			modelPath = "gemma3-270m-it-q8.litertlm"
		} else {
			modelPath = "model.litertlm"
		}
	}

	// 1. Initialize Embedded In-Memory LiteRT-LM Model
	cfg := embeddedllm.DefaultConfig(modelPath)
	cfg.ModelName = "gemma-litert-local"
	cfg.Backend = embeddedllm.BackendCPU

	model, err := embeddedllm.NewModel(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize embedded LiteRT-LM model: %v", err)
	}
	defer model.Close()

	// 2. Build ADK Agent
	agentInstance, err := llmagent.New(llmagent.Config{
		Name:        "embedded_agent",
		Model:       model,
		Description: "Assistant running on an embedded in-memory LiteRT-LM engine.",
		Instruction: "You are a helpful on-device assistant running locally.",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// 3. Launch via ADK Runner
	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(agentInstance),
	}

	l := full.NewLauncher()
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
