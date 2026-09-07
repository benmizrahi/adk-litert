// Package litert provides a low-latency, in-memory LiteRT-LM model engine
// for the Google Agent Development Kit (ADK v2), built for scalable and real-time
// in-process inference in Go.
//
// Usage:
//
//	import "github.com/benmizrahi/adk-litert"
//
//	model, err := litert.NewModel(ctx, litert.DefaultConfig("/path/to/model.litertlm"))
package litert

import (
	"context"

	internallitert "github.com/benmizrahi/adk-litert/internal/model/litert"
)

// Hardware acceleration backend.
type Backend = internallitert.Backend

const (
	// BackendCPU runs inference entirely on the CPU.
	BackendCPU = internallitert.BackendCPU
	// BackendGPU runs inference using GPU acceleration (OpenCL / Vulkan / Metal).
	BackendGPU = internallitert.BackendGPU
)

// Config defines options for initializing the LiteRT model.
type Config = internallitert.Config

// Model represents an in-memory LiteRT-LM instance satisfying ADK model.LLM.
type Model = internallitert.Model

// NewModel loads a .litertlm model into memory and returns a model.LLM instance.
func NewModel(ctx context.Context, config Config) (*Model, error) {
	return internallitert.NewModel(ctx, config)
}

// DefaultConfig returns recommended baseline settings for LiteRT-LM.
func DefaultConfig(modelPath string) Config {
	return internallitert.DefaultConfig(modelPath)
}

// Register registers the LiteRT model loader with ADK's model registry.
// Users can resolve models via model.NewLLM(ctx, "litert:/path/to/model.litertlm").
func Register(pattern string, defaultCfg Config) {
	internallitert.Register(pattern, defaultCfg)
}
