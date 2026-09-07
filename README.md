# adk-litert

[![Go Reference](https://pkg.go.dev/badge/github.com/benmizrahi/adk-litert.svg)](https://pkg.go.dev/github.com/benmizrahi/adk-litert)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.22-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Google ADK](https://img.shields.io/badge/Google_ADK-v2-4285F4?logo=google&logoColor=white)](https://github.com/google/adk)
[![LiteRT-LM](https://img.shields.io/badge/LiteRT--LM-v0.16.0-34A853)](https://github.com/google-ai-edge/LiteRT-LM)
[![Model](https://img.shields.io/badge/Model-Gemma_3_270M-FF6F00?logo=huggingface&logoColor=white)](https://huggingface.co/litert-community/gemma-3-270m-it)
[![Platform](https://img.shields.io/badge/Platform-Linux_%7C_macOS-lightgrey)]()

**adk-litert** brings **low-latency, in-process LLM inference** directly into Go applications and [Google Agent Development Kit (ADK v2)](https://github.com/google/adk) agents.

While Google originally designed [LiteRT-LM](https://github.com/google-ai-edge/LiteRT-LM) for on-device mobile and edge computing, **adk-litert** repurposes its lightweight, self-contained C++ runtime for **scalable, real-time backend and server-side inference**—eliminating external daemon processes, network hops, and operational overhead.

---

## 💡 Motivation: Real-Time, In-Process Inference at Scale

### The Problem with Traditional Model Serving
Most LLM production setups rely on separate model servers (e.g., vLLM, Ollama, Triton, or cloud APIs). While effective for massive foundation models, this architecture creates severe friction for real-time, high-concurrency workloads:
1. **Network & IPC Latency**: Even local loopback HTTP/gRPC calls introduce serialization overhead, socket churn, and connection pooling latency before token generation begins.
2. **Operational Debt**: Managing standalone daemons, sidecars, and process supervisors adds fragility to distributed microservices.
3. **Resource Overkill**: Calling heavy remote models for fast-path tasks (intent routing, classification, structured extraction, PII filtering) wastes cloud budgets and adds 200ms–1000ms roundtrips.

### The In-Memory Advantage
**adk-litert** embeds LiteRT-LM directly into your Go process address space via Cgo:
- **Zero-Hop Execution**: Direct memory calls straight into C++ with no network, sockets, or JSON serialization.
- **Sub-100ms TTFT**: Delivers **~90ms Time To First Token (TTFT)** and **~45 tok/s** on standard commodity CPU cores without requiring GPUs.
- **Horizontal Scalability**: Scale out by simply deploying standard Go microservices or Kubernetes pods. Each instance carries its own self-contained in-memory intelligence (~470MB RAM footprint with Gemma 3 270M).
- **Native ADK v2 Integration**: Fully satisfies Google ADK's `model.LLM` contract, enabling composable, multi-agent workflows with zero boilerplate.

---

## Features

- **In-Memory & Zero-Daemon**: Runs inference in-process via Cgo and the LiteRT C API with no separate service, proxy, or daemon required.
- **ADK v2 Native**: First-class Google ADK `model.LLM` implementation with full support for unary and streaming generation (`iter.Seq2`).
- **Real-Time Responsiveness**: Ultra-low TTFT (~90ms) optimized for real-time conversational agents, voice pipelines, and fast-path triage.
- **Automated Prebuilts**: Uses official Google LiteRT C API prebuilts (`liblitert-lm.so`) fetched automatically—no Bazel or C++ builds required.
- **Hardware Acceleration**: Supports both CPU (XNNPACK) and GPU execution backends.
- **Model Registry**: Supports declarative resolution using ADK's registry format: `model.NewLLM(ctx, "litert:/path/to/model.litertlm")`.

---

## Project Structure

```
├── cmd/
│   └── agent/
│       └── main.go          # Example ADK agent using embedded LiteRT-LM
├── internal/
│   └── model/
│       └── litert/          # Cgo wrapper and ADK model.LLM implementation
├── scripts/
│   └── setup_deps.sh        # Automated fetcher for LiteRT prebuilts
├── litert.go                # Public Go API surface
├── Makefile                 # Build & run automation
├── go.mod                   # Go module definitions
└── README.md
```

---

## Quick Start

### 1. Prerequisites

- **Go**: Version 1.22+ (configured with Go 1.26+)
- **GCC / G++**: For Cgo linking (`sudo apt install build-essential`)
- **curl & unzip**: For downloading official prebuilt runtime libraries

### 2. Download a Model

You can use the official quantized model [`gemma3-270m-it-q8.litertlm`](https://huggingface.co/litert-community/gemma-3-270m-it/blob/main/gemma3-270m-it-q8.litertlm) from the [litert-community/gemma-3-270m-it](https://huggingface.co/litert-community/gemma-3-270m-it/) repository on Hugging Face:

**Via Make:**
```bash
export HF_TOKEN="your_huggingface_token"
make download-model
```

**Or via curl:**
```bash
# Hugging Face authentication token for gated model access:
export HF_TOKEN="your_huggingface_token"

curl -L -H "Authorization: Bearer $HF_TOKEN" -o gemma3-270m-it-q8.litertlm "https://huggingface.co/litert-community/gemma-3-270m-it/resolve/main/gemma3-270m-it-q8.litertlm"
```

---

### 3. Build

Run `make build` to automatically fetch LiteRT runtime dependencies and compile the binary:

```bash
make build
```

The output binary will be generated at `bin/agent`.

---

### 4. Run & Interact with the Agent

#### Interactive Console Mode

Run the agent in console chat mode:

```bash
make run MODEL_PATH="./gemma3-270m-it-q8.litertlm"
```

Or run directly:

```bash
LITERT_MODEL_PATH="./gemma3-270m-it-q8.litertlm" ./bin/agent console
```

#### Web UI Mode

Launch the ADK Web Chat UI on `http://localhost:8080`:

```bash
LITERT_MODEL_PATH="./gemma3-270m-it-q8.litertlm" ./bin/agent web webui api
```

---

## Go API Usage

### Direct Initialization

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/benmizrahi/adk-litert"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	// 1. Configure the model
	cfg := litert.DefaultConfig("/path/to/model.litertlm")
	cfg.ModelName = "gemma-2b-local"
	cfg.Backend = litert.BackendCPU // or litert.BackendGPU
	cfg.MaxSeqLen = 2048

	// 2. Load model into memory
	llm, err := litert.NewModel(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to load model: %v", err)
	}
	defer llm.Close()

	// 3. Generate response (streaming)
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: "Explain quantum computing in one short sentence."},
				},
			},
		},
	}

	for resp, err := range llm.GenerateContent(ctx, req, true) {
		if err != nil {
			log.Fatalf("Generation error: %v", err)
		}
		for _, part := range resp.Content.Parts {
			fmt.Print(part.Text)
		}
	}
	fmt.Println()
}
```

### Using with Google ADK Agents

```go
package main

import (
	"context"
	"log"

	"github.com/benmizrahi/adk-litert"
	"google.golang.org/adk/v2/agent/llmagent"
)

func main() {
	ctx := context.Background()

	cfg := litert.DefaultConfig("/path/to/model.litertlm")
	modelInstance, err := litert.NewModel(ctx, cfg)
	if err != nil {
		log.Fatalf("Model load failed: %v", err)
	}
	defer modelInstance.Close()

	agent, err := llmagent.New(llmagent.Config{
		Name:        "on_device_assistant",
		Model:       modelInstance,
		Description: "Local on-device agent powered by LiteRT-LM.",
		Instruction: "You are an intelligent assistant running on-device.",
	})
	if err != nil {
		log.Fatalf("Failed to create ADK agent: %v", err)
	}

	_ = agent
}
```

### Using ADK Model Registry

```go
// Register litert model scheme
litert.Register("^(?i)litert:.*", litert.DefaultConfig(""))

// Anywhere in your application:
llm, err := model.NewLLM(ctx, "litert:/models/gemma-2b.litertlm")
```

---

## Using in Your Own Go Projects

### 1. Install the Native Runtime Dependency
Because this package interfaces with Google's native LiteRT-LM C runtime, install the prebuilt shared library (`liblitert-lm.so` on Linux, `liblitert-lm.dylib` on macOS) using our one-line installer:

```bash
curl -sSL https://raw.githubusercontent.com/benmizrahi/adk-litert/main/scripts/install.sh | sudo bash
```

*(Or from a local clone: `sudo make install-deps`)*

### 2. Add the Go Module
In your Go application directory:

```bash
go get github.com/benmizrahi/adk-litert
```

### 3. Build Your Application
Because the native runtime and C++ headers are resolved automatically from standard paths (`/usr/local/lib`), you can build your application with standard Go commands:

```bash
go build .
```

---

---

## Performance & TTFT Benchmarking

The project includes an inference performance benchmarking suite designed to measure critical LLM latency and throughput metrics:

- **TTFT (Time To First Token)**: Latency from prompt submission to the first generated token chunk (measures prefill speed and initial response responsiveness).
- **Decode Speed**: Generation throughput in tokens per second (`(Tokens - 1) / Decode Duration`).
- **Inter-Token Latency (ITL / TPOT)**: Time between consecutive streamed tokens (Mean, P50, P90, P95, P99).
- **Total Latency**: End-to-end duration from prompt to final turn token.
- **Memory Footprint**: Process Resident RSS and Go Heap usage before and during generation.

See [**`pkg/benchmark/README.md`**](pkg/benchmark/README.md) for detailed metrics and JSON schemas.

### Sample Measured Results (`gemma3-270m-it-q8.litertlm` on CPU)

| Metric | Min | Mean | Median (P50) | P90 | P95 | Max |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **TTFT (Time To First Token)** | **90.58 ms** | **127.77 ms** | **125.99 ms** | **152.86 ms** | **161.20 ms** | **169.53 ms** |
| **Decode Speed (TPS)** | **42.83 tok/s** | **45.09 tok/s** | **45.91 tok/s** | **46.31 tok/s** | **46.33 tok/s** | **46.36 tok/s** |
| **Total Latency** | 1,831.98 ms | 1,881.48 ms | 1,881.13 ms | 1,938.30 ms | 1,954.30 ms | 1,970.31 ms |
| **Inter-Token Latency (ITL)** | 21.31 ms | 21.88 ms | 21.46 ms | 22.66 ms | 22.76 ms | 22.86 ms |
| **Overall Throughput** | 40.60 tok/s | 42.55 tok/s | 42.53 tok/s | 43.65 tok/s | 43.66 tok/s | 43.67 tok/s |
| **Resident RSS Memory** | — | **471.73 MB** | — | — | — | — |

### Running Benchmarks via CLI

Run the standard benchmark against your model:

```bash
# Run benchmark on CPU
make bench MODEL_PATH="./gemma3-270m-it-q8.litertlm"

# Run benchmark with GPU acceleration
make bench MODEL_PATH="./gemma3-270m-it-q8.litertlm" BACKEND=gpu

# Or run the benchmark binary directly with custom options
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -backend=gpu -runs=5 -warmup=1
```

#### Multi-Prompt Workload Suite
Benchmark across different workload scenarios (short query, medium conversational response, and reasoning / code generation):

```bash
# Multi-prompt suite on GPU
make bench-suite MODEL_PATH="./gemma3-270m-it-q8.litertlm" BACKEND=gpu
```

#### Custom Prompts & JSON Export
```bash
# Benchmark a custom prompt
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -prompt="Summarize the history of AI in 3 bullet points."

# Export structured JSON results (for CI/CD regression tracking or plotting)
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -runs=5 -json -output=bench_results.json
```

### Running Go Standard Benchmarks

You can also run native Go benchmarks via `testing.B`:

```bash
make test-bench
# Or directly:
go test -bench=. -benchmem -v ./...
```

---

## Makefile Commands

| Command | Description |
| :--- | :--- |
| `make build` | Automatically fetches runtime dependencies and builds `bin/agent` |
| `make build-bench` | Compiles the TTFT & inference benchmark CLI (`bin/benchmark`) |
| `make bench` | Runs TTFT and performance benchmarks with default workload |
| `make bench-suite` | Runs multi-prompt comprehensive workload benchmark suite |
| `make test-bench` | Runs standard Go test benchmark suite (`go test -bench`) |
| `make setup` | Explicitly downloads and extracts LiteRT-LM prebuilts |
| `make download-model` | Downloads Gemma 3 270M IT (q8) model from Hugging Face |
| `make run` | Runs `bin/agent` in console mode with `MODEL_PATH` |
| `make test` | Runs Go test suite |
| `make tidy` | Runs `go mod tidy` |
| `make clean` | Cleans build artifacts (`bin/`) |
| `make help` | Displays available make targets and override options |

---

## 🤝 Contributing

Contributions, bug reports, and performance optimizations are warmly welcomed! Please review our [**Contributing Guide**](CONTRIBUTING.md) and PR guidelines before submitting changes.

---

## Disclaimer

> **Disclaimer**: This code is provided "as-is" as a demonstration only to illustrate a potential solution. The code does not constitute a Google product or service of any kind, and Google offers no support, warranties, or liability of any kind with its regard. Whoever chooses to use this code accepts all responsibility related to it, including for its implementation, use, and ongoing maintenance. For the avoidance of doubt, this code is not eligible for the Google Open Source Software Vulnerability Rewards Program.


