# Benchmark CLI Tool (`cmd/benchmark`)

The `cmd/benchmark` directory contains the CLI entrypoint for running Time To First Token (TTFT), token throughput, and memory benchmarks for `adk-litert`.

For full benchmark results, metric definitions, and JSON schemas, see [**`pkg/benchmark/README.md`**](../../pkg/benchmark/README.md).

---

## Quick Start

```bash
# Build and run default benchmark
make bench MODEL_PATH="./gemma3-270m-it-q8.litertlm"

# Run multi-prompt workload suite
make bench-suite MODEL_PATH="./gemma3-270m-it-q8.litertlm"
```

## CLI Usage

```bash
# Compile binary to bin/benchmark
make build-bench

# Run with custom options
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -backend=cpu -runs=5 -warmup=1 -prompt="Your prompt here"

# Export structured JSON
./bin/benchmark -model="./gemma3-270m-it-q8.litertlm" -runs=5 -json -output="results.json"
```

## Benchmark Metrics Summary

- **TTFT (Time To First Token)**: Prefill latency (first token emission).
- **Decode Speed**: Output tokens per second (`tok/s`).
- **Inter-Token Latency (ITL / TPOT)**: Streaming latency between tokens.
- **Total Latency**: Full turn duration.
- **Resident Memory**: Total process RSS in MB.
