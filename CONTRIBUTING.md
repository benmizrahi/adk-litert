# Contributing to adk-litert

Thank you for your interest in contributing to **adk-litert**! We welcome contributions, bug reports, feature suggestions, and performance optimizations.

---

## Code of Conduct

We are committed to providing a welcoming, inclusive, and harassment-free experience for everyone. Please be respectful, constructive, and collaborative in all interactions across issues, pull requests, and discussions.

---

## Branch Protection & PR Policy

To ensure repository stability and high code quality, **direct pushes to the `main` branch are protected and disabled**.

- All changes must be submitted via a **Pull Request (PR)**.
- Each PR requires at least **one approved code review** before merging.
- Stale review approvals are automatically dismissed when new commits are pushed to the PR branch.
- PRs must pass all automated build and test checks before merge.

---

## Development Setup

### 1. Prerequisites

- **Go**: Version 1.22+ (configured with Go 1.26+)
- **C/C++ Compiler**: `gcc` / `g++` (C++17 support)
  - Ubuntu/Debian: `sudo apt install build-essential curl unzip`
  - macOS: `xcode-select --install`
- **Cgo**: Must be enabled (`export CGO_ENABLED=1`)

### 2. Fork and Clone

```bash
# Clone your fork
git clone https://github.com/<your-username>/adk-litert.git
cd adk-litert

# Add upstream remote
git remote add upstream https://github.com/benmizrahi/adk-litert.git
```

### 3. Fetch Runtime Dependencies

`adk-litert` uses Google's official prebuilt LiteRT-LM C runtime libraries. Fetch them automatically:

```bash
make setup
```

This downloads and extracts `liblitert-lm.so` (Linux) or `liblitert-lm.dylib` (macOS) and headers into `lib/` and `include/`.

### 4. (Optional) Download a Model for Local Testing

If you are testing end-to-end inference or running benchmarks, download Gemma 3 270M IT:

```bash
export HF_TOKEN="your_huggingface_token"
make download-model
```

---

## Development Workflow

### 1. Create a Topic Branch

Always create a descriptive branch off of the latest `main`:

```bash
git checkout main
git pull upstream main
git checkout -b feat/your-feature-name
# or: git checkout -b fix/bug-description
```

Branch naming suggestions:
- `feat/...` for new features or enhancements
- `fix/...` for bug fixes
- `perf/...` for latency, throughput, or memory optimizations
- `docs/...` for documentation improvements
- `refactor/...` for code refactoring without behavior change

### 2. Verify Your Changes Locally

Before submitting your PR, ensure the entire test and build suite passes:

```bash
# 1. Format code according to standard Go style
gofmt -s -w .

# 2. Ensure go.mod and go.sum are tidy
go mod tidy

# 3. Run all tests
make test
# or: go test -v ./...

# 4. Verify compilation of example agent and benchmark CLI
make build
make build-bench

# 5. (If touching inference code) Run benchmark to check for regressions
make bench MODEL_PATH="./gemma3-270m-it-q8.litertlm"
```

---

## Engineering Guidelines

### Cgo & Native Interoperability

Because `adk-litert` bridges Go with Google's native LiteRT C++ engine via Cgo:

1. **Memory Ownership**: Always pair every `C.CString` allocation with `defer C.free(unsafe.Pointer(cStr))`.
2. **Pointer Safety**: Do not pass Go pointers into C memory across goroutines without proper pinning or copying.
3. **Thread Safety**: The underlying LiteRT engine instance must remain thread-safe. Model inference calls are guarded by mutexes where appropriate.
4. **Cross-Platform Support**: Keep builds clean across Linux (x86_64, arm64) and macOS Apple Silicon (arm64). Do not introduce Linux-only or macOS-only dependencies without build tags (`//go:build`).

### Google ADK Integration

- Maintain compatibility with [`google.golang.org/adk/v2`](https://github.com/google/adk).
- Implementations must satisfy the `model.LLM` interface, including streaming support via Go 1.23+ range-over-function iterators (`iter.Seq2[*model.LLMResponse, error]`).

### Commit Messages

Write clear, imperative commit messages following the Conventional Commits style:

```
feat: add GPU acceleration support for Apple Silicon Metal
fix: prevent memory leak in token callback gateway
perf: reduce prompt prefill allocation in Cgo bridge
docs: update quick start instructions in README
```

---

## Submitting a Pull Request

1. Push your branch to your fork:
   ```bash
   git push origin feat/your-feature-name
   ```
2. Open a Pull Request against `benmizrahi/adk-litert` on GitHub (`main` branch).
3. Fill out the Pull Request template detailing:
   - What changed and why.
   - How the change was tested.
   - Any benchmark results (for performance-sensitive changes).
4. Respond to code review comments. If updates are pushed, reviewers will be notified.
5. Once approved and checks pass, your PR will be squash-merged into `main`.

---

## Reporting Issues & Feature Requests

- **Bug Reports**: Open an issue including your OS, Go version, hardware (CPU/GPU), model name/size, LiteRT-LM release version, and clear steps to reproduce.
- **Feature Requests**: Open an issue describing the proposed functionality, the motivating use case, and potential implementation approach.
