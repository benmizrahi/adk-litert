# LiteRT-LM Performance & TTFT Benchmarking

This package provides an end-to-end inference benchmarking suite for on-device [LiteRT-LM](https://github.com/google-ai-edge/LiteRT-LM) models integrated with Go and the [Google Agent Development Kit (ADK v2)](https://github.com/google/adk).

---

## 📊 Measured Benchmark Results

The following metrics were collected on an **AMD EPYC 7B12 CPU** using the [`gemma3-270m-it-q8.litertlm`](https://huggingface.co/litert-community/gemma-3-270m-it/blob/main/gemma3-270m-it-q8.litertlm) model from [litert-community/gemma-3-270m-it](https://huggingface.co/litert-community/gemma-3-270m-it/) on Linux (x86_64).

### Standard Prompt Benchmark

- **Prompt**: `"Explain the theory of general relativity in 3 concise bullet points."`
- **Runs**: 1 warmup run, 5 measured runs
- **Backend**: CPU (XNNPACK delegate)
- **Model Size**: ~270M Parameters (`gemma3-270m-it-q8.litertlm`)

| Metric | Min | Mean | Median (P50) | P90 | P95 | Max |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **TTFT (Time To First Token)** | **90.58 ms** | **127.77 ms** | **125.99 ms** | **152.86 ms** | **161.20 ms** | **169.53 ms** |
| **Decode Speed (TPS)** | **42.83 tok/s** | **45.09 tok/s** | **45.91 tok/s** | **46.31 tok/s** | **46.33 tok/s** | **46.36 tok/s** |
| **Total Latency** | 1,831.98 ms | 1,881.48 ms | 1,881.13 ms | 1,938.30 ms | 1,954.30 ms | 1,970.31 ms |
| **Inter-Token Latency (ITL)** | 21.31 ms | 21.88 ms | 21.46 ms | 22.66 ms | 22.76 ms | 22.86 ms |
| **Overall Throughput** | 40.60 tok/s | 42.55 tok/s | 42.53 tok/s | 43.65 tok/s | 43.66 tok/s | 43.67 tok/s |

### Memory Footprint
- **Resident Set Size (RSS)**: `~471.73 MB` (Total memory including native C++ LiteRT engine & model weights)
- **Go Heap Allocated**: `~1.04 MB`

---

## 🎯 Key Metrics Explained

| Metric | Definition | Importance |
| :--- | :--- | :--- |
| **TTFT (Time To First Token)** | Time elapsed between submitting the prompt and receiving the first generated token chunk. | Primary metric for UI responsiveness and user-perceived interactivity. Reflects prompt ingestion / prefill speed. |
| **Decode Speed (tok/s)** | Output tokens generated per second during the auto-regressive generation phase: `(Output Tokens - 1) / (Total Time - TTFT)`. | Measures pure decoding throughput excluding prompt prefill time. |
| **Inter-Token Latency (ITL / TPOT)** | Time gap between consecutive emitted tokens during streaming. | Determines streaming smoothness; low jitter ensures natural reading speed for real-time assistants. |
| **Total Latency** | Wall-clock duration from start of request to completion. | Critical for batch processing and synchronous agent tool calls. |
| **Resident RSS** | Memory allocated by the OS for process pages in RAM. | Measures total in-memory footprint of embedded model weights and C++ KV cache. |

---

## 🚀 How to Run Benchmarks

### 1. Quick Run via Makefile

```bash
# Run standard benchmark (1 warmup + 5 measured runs)
make bench MODEL_PATH="./gemma3-270m-it-q8.litertlm"

# Run multi-prompt workload suite (Short, Medium, Code/Reasoning)
make bench-suite MODEL_PATH="./gemma3-270m-it-q8.litertlm"

# Run native Go testing benchmarks (testing.B)
make test-bench
```

### 2. Custom CLI Options (`bin/benchmark`)

Compile the benchmark CLI:
```bash
make build-bench
```

Execute with custom flags:
```bash
# Benchmark custom prompt on CPU with 10 iterations
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -backend=cpu -runs=10 -warmup=2 -prompt="List the 7 wonders of the ancient world."

# Benchmark using prompt from a file
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -prompt-file="prompt.txt"

# Export structured JSON report for CI/CD tracking
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -runs=5 -json -output="benchmark_results.json"
```

### CLI Flag Reference

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-model` | `$LITERT_MODEL_PATH` | Path to `.litertlm` model file |
| `-backend` | `cpu` | Acceleration backend (`cpu` or `gpu`) |
| `-runs` | `5` | Number of measured iterations |
| `-warmup` | `1` | Number of unmeasured warmup iterations |
| `-prompt` | *(built-in)* | Custom prompt string |
| `-prompt-file` | `""` | File containing prompt text |
| `-suite` | `false` | Run multi-scenario workload suite |
| `-json` | `false` | Output results in JSON format |
| `-output` | `""` | Save output report to specified file path |
| `-max-seq-len`| `2048` | Maximum sequence context length |

---

## 📋 JSON Output Schema

When `-json` is specified, the benchmark outputs structured telemetry suitable for CI/CD regression tracking, automated dashboards, or plotting:

```json
{
  "model_name": "LiteRT-LM",
  "backend": "cpu",
  "model_path": "./gemma3-270m-it-q8.litertlm",
  "warmup_runs": 1,
  "measured_runs": 5,
  "prompt_summary": "Explain the theory of general relativity in 3 concise bullet points.",
  "ttft_ms": {
    "min": 90.58,
    "mean": 127.77,
    "median": 125.99,
    "p90": 152.86,
    "p95": 161.20,
    "max": 169.53,
    "stddev": 28.14
  },
  "decode_tokens_per_sec": {
    "min": 42.83,
    "mean": 45.09,
    "median": 45.91,
    "p90": 46.31,
    "p95": 46.33,
    "max": 46.36,
    "stddev": 1.35
  },
  "total_duration_ms": {
    "min": 1831.98,
    "mean": 1881.48,
    "median": 1881.13,
    "p90": 1938.30,
    "p95": 1954.30,
    "max": 1970.31,
    "stddev": 48.22
  },
  "itl_ms": {
    "min": 21.31,
    "mean": 21.88,
    "median": 21.46,
    "p90": 22.66,
    "p95": 22.76,
    "max": 22.86,
    "stddev": 0.61
  },
  "memory_rss_mb": 471.73,
  "go_heap_alloc_mb": 1.04
}
```
