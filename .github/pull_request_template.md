## Description

<!-- Provide a brief summary of what this PR introduces, fixes, or improves. -->

## Type of Change

- [ ] 🐛 Bug fix (non-breaking change fixing an issue)
- [ ] ✨ New feature (non-breaking change adding functionality)
- [ ] ⚡ Performance improvement (optimization of latency, throughput, or memory)
- [ ] 📝 Documentation update
- [ ] ♻️ Code refactoring (no functional changes)
- [ ] 🧪 Tests / Benchmarks

## Motivation & Context

<!-- Why is this change needed? What problem does it solve? Link any related issues with #issue_number. -->

## How Has This Been Tested?

- [ ] `go test -v ./...` passes
- [ ] `make build` successfully compiles `bin/agent`
- [ ] `make build-bench` successfully compiles `bin/benchmark`
- [ ] Tested on Linux (x86_64 / arm64)
- [ ] Tested on macOS (arm64)
- [ ] Performance benchmark run (attach TTFT / throughput numbers if relevant)

### Benchmark Results (if applicable)

```
<!-- Paste benchmark CLI output or table here -->
```

## Checklist

- [ ] My code follows the code style of this project (`gofmt -s -w .`)
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in Cgo and pointer-sensitive areas
- [ ] I have made corresponding changes to the documentation if applicable
- [ ] My changes generate no new warnings or compiler errors
