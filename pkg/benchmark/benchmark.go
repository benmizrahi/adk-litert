package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

// IterationResult stores metrics for a single generation run.
type IterationResult struct {
	IterationIndex int           `json:"iteration"`
	Prompt         string        `json:"prompt"`
	ResponseText   string        `json:"response_text"`
	PromptChars    int           `json:"prompt_chars"`
	OutputTokens   int           `json:"output_tokens"`
	OutputChars    int           `json:"output_chars"`

	// Timing metrics
	TTFT           time.Duration `json:"ttft_ns"`            // Time To First Token
	TotalDuration  time.Duration `json:"total_duration_ns"`   // Total generation duration
	DecodeDuration time.Duration `json:"decode_duration_ns"`  // TotalDuration - TTFT

	// Throughput
	TokensPerSec       float64       `json:"tokens_per_sec"`        // OutputTokens / TotalDuration (in sec)
	DecodeTokensPerSec float64       `json:"decode_tokens_per_sec"` // (OutputTokens - 1) / DecodeDuration (in sec)

	// Inter-Token Latencies
	MeanITL time.Duration   `json:"mean_itl_ns"` // Mean Inter-Token Latency
	ITLs    []time.Duration `json:"itls_ns,omitempty"`
}

// PercentileStats holds distribution statistics for a metric.
type PercentileStats struct {
	Min    float64 `json:"min"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"` // P50
	P90    float64 `json:"p90"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	Max    float64 `json:"max"`
	StdDev float64 `json:"stddev"`
}

// SuiteReport contains aggregated benchmark results across multiple iterations.
type SuiteReport struct {
	ModelName     string `json:"model_name"`
	Backend       string `json:"backend"`
	ModelPath     string `json:"model_path"`
	WarmupRuns    int    `json:"warmup_runs"`
	MeasuredRuns  int    `json:"measured_runs"`
	PromptSummary string `json:"prompt_summary"`

	// Aggregated Statistics (times in milliseconds for readability)
	TTFTMs             PercentileStats `json:"ttft_ms"`
	TotalDurationMs    PercentileStats `json:"total_duration_ms"`
	DecodeDurationMs   PercentileStats `json:"decode_duration_ms"`
	ITLMs              PercentileStats `json:"itl_ms"`
	TokensPerSec       PercentileStats `json:"tokens_per_sec"`
	DecodeTokensPerSec PercentileStats `json:"decode_tokens_per_sec"`

	// Memory footprint
	MemoryRSSMB   float64 `json:"memory_rss_mb"`
	GoHeapAllocMB float64 `json:"go_heap_alloc_mb"`

	// Raw iteration data
	Iterations []IterationResult `json:"iterations"`
}

// Config defines options for benchmark runs.
type Config struct {
	WarmupRuns   int
	MeasuredRuns int
	Prompt       string
	Label        string
}

// DefaultConfig returns baseline benchmark options.
func DefaultConfig() Config {
	return Config{
		WarmupRuns:   1,
		MeasuredRuns: 5,
		Prompt:       "Explain the theory of general relativity in 3 concise bullet points.",
		Label:        "Standard Benchmark",
	}
}

// Run executes the benchmark against a model.LLM instance.
func Run(ctx context.Context, llm model.LLM, cfg Config, modelName, backend, modelPath string) (*SuiteReport, error) {
	if cfg.WarmupRuns < 0 {
		cfg.WarmupRuns = 0
	}
	if cfg.MeasuredRuns <= 0 {
		cfg.MeasuredRuns = 1
	}

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: cfg.Prompt},
				},
			},
		},
	}

	// 1. Warmup runs
	for i := 0; i < cfg.WarmupRuns; i++ {
		if err := runSingleIteration(ctx, llm, req, nil); err != nil {
			return nil, fmt.Errorf("warmup run %d failed: %w", i+1, err)
		}
	}

	// 2. Measured runs
	var results []IterationResult
	for i := 0; i < cfg.MeasuredRuns; i++ {
		res := &IterationResult{
			IterationIndex: i + 1,
			Prompt:         cfg.Prompt,
			PromptChars:    len(cfg.Prompt),
		}
		if err := runSingleIteration(ctx, llm, req, res); err != nil {
			return nil, fmt.Errorf("measured run %d failed: %w", i+1, err)
		}
		results = append(results, *res)
	}

	// 3. Compute Aggregated Stats
	report := computeReport(results, cfg, modelName, backend, modelPath)
	return report, nil
}

func runSingleIteration(ctx context.Context, llm model.LLM, req *model.LLMRequest, out *IterationResult) error {
	startTime := time.Now()
	var firstTokenTime time.Time
	var lastTokenTime time.Time
	var tokenCount int
	var itls []time.Duration
	var respBuilder strings.Builder

	streamSeq := llm.GenerateContent(ctx, req, true)
	for resp, err := range streamSeq {
		if err != nil {
			return err
		}
		now := time.Now()
		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				tokenCount++
				respBuilder.WriteString(part.Text)
				if tokenCount == 1 {
					firstTokenTime = now
				} else {
					itls = append(itls, now.Sub(lastTokenTime))
				}
				lastTokenTime = now
			}
		}
	}

	endTime := time.Now()

	if out != nil {
		out.OutputTokens = tokenCount
		out.OutputChars = respBuilder.Len()
		out.ResponseText = respBuilder.String()
		out.TotalDuration = endTime.Sub(startTime)

		if tokenCount > 0 {
			out.TTFT = firstTokenTime.Sub(startTime)
			out.DecodeDuration = endTime.Sub(firstTokenTime)
		} else {
			out.TTFT = out.TotalDuration
			out.DecodeDuration = 0
		}

		totalSec := out.TotalDuration.Seconds()
		if totalSec > 0 {
			out.TokensPerSec = float64(tokenCount) / totalSec
		}

		decodeSec := out.DecodeDuration.Seconds()
		if decodeSec > 0 && tokenCount > 1 {
			out.DecodeTokensPerSec = float64(tokenCount-1) / decodeSec
		} else if decodeSec > 0 {
			out.DecodeTokensPerSec = float64(tokenCount) / decodeSec
		}

		out.ITLs = itls
		if len(itls) > 0 {
			var sum time.Duration
			for _, itl := range itls {
				sum += itl
			}
			out.MeanITL = sum / time.Duration(len(itls))
		}
	}

	return nil
}

func computeReport(results []IterationResult, cfg Config, modelName, backend, modelPath string) *SuiteReport {
	report := &SuiteReport{
		ModelName:     modelName,
		Backend:       backend,
		ModelPath:     modelPath,
		WarmupRuns:    cfg.WarmupRuns,
		MeasuredRuns:  cfg.MeasuredRuns,
		PromptSummary: cfg.Prompt,
		Iterations:    results,
	}

	var ttftValues []float64
	var totalDurValues []float64
	var decodeDurValues []float64
	var itlValues []float64
	var tpsValues []float64
	var decodeTpsValues []float64

	for _, res := range results {
		ttftValues = append(ttftValues, float64(res.TTFT.Microseconds())/1000.0) // ms
		totalDurValues = append(totalDurValues, float64(res.TotalDuration.Microseconds())/1000.0)
		decodeDurValues = append(decodeDurValues, float64(res.DecodeDuration.Microseconds())/1000.0)
		if res.MeanITL > 0 {
			itlValues = append(itlValues, float64(res.MeanITL.Microseconds())/1000.0)
		}
		tpsValues = append(tpsValues, res.TokensPerSec)
		decodeTpsValues = append(decodeTpsValues, res.DecodeTokensPerSec)
	}

	report.TTFTMs = calcStats(ttftValues)
	report.TotalDurationMs = calcStats(totalDurValues)
	report.DecodeDurationMs = calcStats(decodeDurValues)
	report.ITLMs = calcStats(itlValues)
	report.TokensPerSec = calcStats(tpsValues)
	report.DecodeTokensPerSec = calcStats(decodeTpsValues)

	// Memory stats
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
		// Linux: Maxrss is in Kilobytes
		report.MemoryRSSMB = float64(rusage.Maxrss) / 1024.0
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	report.GoHeapAllocMB = float64(m.Alloc) / (1024.0 * 1024.0)

	return report
}

func calcStats(values []float64) PercentileStats {
	if len(values) == 0 {
		return PercentileStats{}
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	minVal := sorted[0]
	maxVal := sorted[len(sorted)-1]

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	meanVal := sum / float64(len(sorted))

	var varianceSum float64
	for _, v := range sorted {
		varianceSum += (v - meanVal) * (v - meanVal)
	}
	stdDev := math.Sqrt(varianceSum / float64(len(sorted)))

	return PercentileStats{
		Min:    minVal,
		Mean:   meanVal,
		Median: percentile(sorted, 50),
		P90:    percentile(sorted, 90),
		P95:    percentile(sorted, 95),
		P99:    percentile(sorted, 99),
		Max:    maxVal,
		StdDev: stdDev,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	weight := idx - float64(lower)
	return sorted[lower]*(1.0-weight) + sorted[upper]*weight
}

// FormatTable returns a nicely styled ASCII report for CLI printing.
func (r *SuiteReport) FormatTable() string {
	var b strings.Builder

	b.WriteString("\n================================================================================\n")
	b.WriteString("                     LiteRT-LM Inference Performance Benchmark                  \n")
	b.WriteString("================================================================================\n")
	b.WriteString(fmt.Sprintf(" Model Name:        %s\n", r.ModelName))
	b.WriteString(fmt.Sprintf(" Backend:           %s\n", r.Backend))
	b.WriteString(fmt.Sprintf(" Model Path:        %s\n", r.ModelPath))
	b.WriteString(fmt.Sprintf(" Warmup Iterations: %d\n", r.WarmupRuns))
	b.WriteString(fmt.Sprintf(" Measured Runs:     %d\n", r.MeasuredRuns))
	b.WriteString(fmt.Sprintf(" Resident RSS:      %.2f MB (Heap: %.2f MB)\n", r.MemoryRSSMB, r.GoHeapAllocMB))
	b.WriteString("--------------------------------------------------------------------------------\n")
	b.WriteString(fmt.Sprintf(" Prompt: %q\n", truncateString(r.PromptSummary, 70)))
	b.WriteString("--------------------------------------------------------------------------------\n")

	b.WriteString(fmt.Sprintf("%-24s | %-8s | %-8s | %-8s | %-8s | %-8s | %-8s\n",
		"Metric", "Min", "Mean", "Median", "P90", "P95", "Max"))
	b.WriteString("-------------------------+----------+----------+----------+----------+----------+----------\n")

	b.WriteString(fmt.Sprintf("%-24s | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f\n",
		"TTFT (ms)",
		r.TTFTMs.Min, r.TTFTMs.Mean, r.TTFTMs.Median, r.TTFTMs.P90, r.TTFTMs.P95, r.TTFTMs.Max))

	b.WriteString(fmt.Sprintf("%-24s | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f\n",
		"Decode Speed (tok/s)",
		r.DecodeTokensPerSec.Min, r.DecodeTokensPerSec.Mean, r.DecodeTokensPerSec.Median,
		r.DecodeTokensPerSec.P90, r.DecodeTokensPerSec.P95, r.DecodeTokensPerSec.Max))

	b.WriteString(fmt.Sprintf("%-24s | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f\n",
		"Total Latency (ms)",
		r.TotalDurationMs.Min, r.TotalDurationMs.Mean, r.TotalDurationMs.Median,
		r.TotalDurationMs.P90, r.TotalDurationMs.P95, r.TotalDurationMs.Max))

	b.WriteString(fmt.Sprintf("%-24s | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f\n",
		"Inter-Token Latency (ms)",
		r.ITLMs.Min, r.ITLMs.Mean, r.ITLMs.Median, r.ITLMs.P90, r.ITLMs.P95, r.ITLMs.Max))

	b.WriteString(fmt.Sprintf("%-24s | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f | %8.2f\n",
		"Overall Speed (tok/s)",
		r.TokensPerSec.Min, r.TokensPerSec.Mean, r.TokensPerSec.Median,
		r.TokensPerSec.P90, r.TokensPerSec.P95, r.TokensPerSec.Max))

	b.WriteString("================================================================================\n")
	if len(r.Iterations) > 0 {
		b.WriteString(fmt.Sprintf(" Sample Output (%d tokens):\n", r.Iterations[0].OutputTokens))
		b.WriteString(fmt.Sprintf("   %q\n", truncateString(r.Iterations[0].ResponseText, 140)))
		b.WriteString("================================================================================\n")
	}

	return b.String()
}

// ToJSON serializes the suite report to formatted JSON.
func (r *SuiteReport) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// MarshalIndentJSON serializes any data structure to indented JSON.
func MarshalIndentJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

