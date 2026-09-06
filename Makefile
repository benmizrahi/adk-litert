# ==============================================================================
# LiteRT-LM In-Memory ADK Agent Makefile
# ==============================================================================

# Paths to LiteRT-LM prebuilt libraries and headers
LITERT_LM_LIB  ?= $(shell pwd)/lib
LITERT_LM_INC  ?= $(shell pwd)/include
MODEL_PATH     ?= $(shell pwd)/model.litertlm
BACKEND        ?= cpu
HF_TOKEN       ?=

# Output binary
BIN_DIR := bin
BINARY  := $(BIN_DIR)/agent

# Cgo compilation flags
export CGO_ENABLED ?= 1
export CGO_CXXFLAGS ?= -std=c++17 -I$(shell pwd)/internal/model/litert -I$(LITERT_LM_INC)
export CGO_LDFLAGS  ?= -L$(LITERT_LM_LIB) -Wl,-rpath,$(LITERT_LM_LIB) -llitert-lm -lstdc++
export LD_LIBRARY_PATH := $(LITERT_LM_LIB):$(LD_LIBRARY_PATH)

.PHONY: all setup deps build run download-model test tidy clean help

all: build

## setup: Clones and builds LiteRT-LM runtime dependency
setup: deps

## deps: Automatically fetches and compiles LiteRT-LM runtime if missing
deps:
	@./scripts/setup_deps.sh

## download-model: Downloads the sample FunctionGemma / tiny_garden model from HuggingFace
download-model:
	@echo "==> Downloading tiny_garden.litertlm from HuggingFace..."
	@TOKEN="$${HF_TOKEN:-$(HF_TOKEN)}"; \
	if [ -n "$$TOKEN" ]; then \
		echo "==> Authenticating download with HF_TOKEN..."; \
		curl -f -L -H "Authorization: Bearer $$TOKEN" -o tiny_garden.litertlm "https://huggingface.co/google/functiongemma-270m-it/resolve/main/tiny_garden.litertlm" || { rm -f tiny_garden.litertlm; echo "Error: Failed to download model. Please verify your HF_TOKEN."; exit 1; }; \
	else \
		echo "==> No HF_TOKEN provided; downloading without auth (gated repos may fail)..."; \
		curl -f -L -o tiny_garden.litertlm "https://huggingface.co/google/functiongemma-270m-it/resolve/main/tiny_garden.litertlm" || { rm -f tiny_garden.litertlm; echo "Error: Failed to download model. HuggingFace requires authentication for gated models. Please set HF_TOKEN (e.g. export HF_TOKEN=... or make download-model HF_TOKEN=...)."; exit 1; }; \
	fi
	@ln -sf tiny_garden.litertlm model.litertlm
	@echo "==> Model saved to tiny_garden.litertlm and linked to model.litertlm"

## build: Compiles the Go ADK application (cmd/agent)
build: deps
	@mkdir -p $(BIN_DIR)
	@echo "==> Building $(BINARY)..."
	go build -o $(BINARY) ./cmd/agent

## build-bench: Compiles the TTFT & throughput benchmark CLI
build-bench: deps
	@mkdir -p $(BIN_DIR)
	@echo "==> Building $(BIN_DIR)/benchmark..."
	go build -o $(BIN_DIR)/benchmark ./cmd/benchmark

## bench: Runs TTFT and performance benchmark tool
bench: build-bench
	@$(BIN_DIR)/benchmark -model="$(MODEL_PATH)" -backend="$(BACKEND)"

## benchmark: Alias for bench
benchmark: bench

## bench-suite: Runs multi-prompt comprehensive workload benchmark
bench-suite: build-bench
	@$(BIN_DIR)/benchmark -model="$(MODEL_PATH)" -backend="$(BACKEND)" -suite

## test-bench: Runs Go test benchmark suite measuring TTFT
test-bench: deps
	@echo "==> Running Go benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

## run: Runs the in-memory ADK agent with the specified model
run: build
	@echo "==> Running $(BINARY) with model: $(MODEL_PATH)..."
	LITERT_MODEL_PATH="$(MODEL_PATH)" $(BINARY) console

## tidy: Cleans and updates Go module dependencies
tidy:
	@echo "==> Updating go.mod dependencies..."
	go mod tidy

## test: Runs Go test suite
test:
	@echo "==> Running tests..."
	go test -v ./...

## clean: Removes build artifacts
clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(BIN_DIR)

## help: Shows available Makefile targets
help:
	@echo "Usage: make [target] [VARIABLES...]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':'
	@echo ""
	@echo "Environment / Override Variables:"
	@echo "  LITERT_LM_ROOT   Path to LiteRT-LM repository (default: ./LiteRT-LM)"
	@echo "  LITERT_LM_LIB    Path to compiled LiteRT-LM .so (default: \$$LITERT_LM_ROOT/bazel-bin/runtime)"
	@echo "  MODEL_PATH       Path to .litertlm model file (default: ./model.litertlm)"
	@echo "  BACKEND          Inference backend: cpu or gpu (default: cpu)"
	@echo "  HF_TOKEN         Hugging Face access token (default: \$$HF_TOKEN from environment)"
