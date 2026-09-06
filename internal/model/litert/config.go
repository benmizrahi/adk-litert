package litert

// Backend specifies the hardware acceleration backend for LiteRT-LM.
type Backend int

const (
	// BackendCPU runs inference entirely on the CPU.
	BackendCPU Backend = 0
	// BackendGPU runs inference using GPU acceleration (e.g. OpenCL / Vulkan / Metal).
	BackendGPU Backend = 1
)

func (b Backend) String() string {
	switch b {
	case BackendGPU:
		return "GPU"
	default:
		return "CPU"
	}
}

// Config defines options for initializing the in-memory LiteRT model.
type Config struct {
	// ModelPath is the file path to the .litertlm model assets.
	ModelPath string
	// ModelName is the user-facing name for this model (e.g. "gemma-2b-it").
	ModelName string
	// Backend configures CPU or GPU inference.
	Backend Backend
	// MaxSeqLen is the maximum context length in tokens (default 2048).
	MaxSeqLen int
	// Temperature controls randomness in generation (default 0.7).
	Temperature float32
	// TopP is nucleus sampling parameter (default 0.9).
	TopP float32
	// TopK is top-k sampling parameter (default 40).
	TopK int
}

// DefaultConfig returns recommended baseline settings for LiteRT-LM.
func DefaultConfig(modelPath string) Config {
	return Config{
		ModelPath:   modelPath,
		ModelName:   "litert-inmemory",
		Backend:     BackendCPU,
		MaxSeqLen:   2048,
		Temperature: 0.7,
		TopP:        0.9,
		TopK:        40,
	}
}
