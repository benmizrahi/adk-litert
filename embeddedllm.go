// Package embeddedllm provides an in-memory LiteRT-LM model implementation
// for the Google Agent Development Kit (ADK v2).
//
// Usage:
//
//	import "adk-embedded-litert" // or "github.com/.../adk-embedded-litert"
//
//	model, err := embeddedllm.NewModel(ctx, embeddedllm.DefaultConfig("/path/to/model.litertlm"))
package embeddedllm

import (
	"context"

	"adk-embedded-litert/internal/model/litert"
)

// Hardware acceleration backend.
type Backend = litert.Backend

const (
	// BackendCPU runs inference entirely on the CPU.
	BackendCPU = litert.BackendCPU
	// BackendGPU runs inference using GPU acceleration (OpenCL / Vulkan / Metal).
	BackendGPU = litert.BackendGPU
)

// Config defines options for initializing the embedded LiteRT model.
type Config = litert.Config

// Model represents an embedded in-memory LiteRT-LM instance satisfying ADK model.LLM.
type Model = litert.Model

// NewModel loads a .litertlm model into memory and returns a model.LLM instance.
func NewModel(ctx context.Context, config Config) (*Model, error) {
	return litert.NewModel(ctx, config)
}

// DefaultConfig returns recommended baseline settings for LiteRT-LM.
func DefaultConfig(modelPath string) Config {
	return litert.DefaultConfig(modelPath)
}

// Register registers the LiteRT model loader with ADK's model registry.
// Users can resolve models via model.NewLLM(ctx, "litert:/path/to/model.litertlm").
func Register(pattern string, defaultCfg Config) {
	litert.Register(pattern, defaultCfg)
}
