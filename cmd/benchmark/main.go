package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	embeddedllm "adk-embedded-litert"
	"adk-embedded-litert/pkg/benchmark"
)

func main() {
	modelPathFlag := flag.String("model", "", "Path to .litertlm model file (or set LITERT_MODEL_PATH env var)")
	backendFlag := flag.String("backend", "cpu", "Backend to use: 'cpu' or 'gpu'")
	runsFlag := flag.Int("runs", 5, "Number of measured benchmark runs")
	warmupFlag := flag.Int("warmup", 1, "Number of warmup iterations")
	promptFlag := flag.String("prompt", "", "Custom prompt text to benchmark")
	promptFileFlag := flag.String("prompt-file", "", "Path to file containing prompt text")
	suiteFlag := flag.Bool("suite", false, "Run comprehensive multi-prompt workload suite (Short, Medium, Code/Reasoning)")
	jsonFlag := flag.Bool("json", false, "Output results as formatted JSON")
	outputFlag := flag.String("output", "", "Save benchmark results to specified file path")
	maxSeqLenFlag := flag.Int("max-seq-len", 2048, "Maximum sequence length for model context")

	flag.Parse()

	modelPath := *modelPathFlag
	if modelPath == "" {
		modelPath = os.Getenv("LITERT_MODEL_PATH")
	}
	if modelPath == "" {
		if _, err := os.Stat("gemma3-270m-it-q8.litertlm"); err == nil {
			modelPath = "gemma3-270m-it-q8.litertlm"
		} else if _, err := os.Stat("tiny_garden.litertlm"); err == nil {
			modelPath = "tiny_garden.litertlm"
		} else if _, err := os.Stat("model.litertlm"); err == nil {
			modelPath = "model.litertlm"
		} else {
			log.Fatalf("Error: Model file not specified. Use -model flag or set LITERT_MODEL_PATH.")
		}
	}

	backend := embeddedllm.BackendCPU
	if strings.ToLower(*backendFlag) == "gpu" {
		backend = embeddedllm.BackendGPU
	}

	ctx := context.Background()

	cfg := embeddedllm.DefaultConfig(modelPath)
	cfg.Backend = backend
	cfg.MaxSeqLen = *maxSeqLenFlag
	cfg.ModelName = "litert-benchmark"

	fmt.Printf("==> Loading LiteRT model from %s (%s)...\n", modelPath, *backendFlag)
	model, err := embeddedllm.NewModel(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize model: %v", err)
	}
	defer model.Close()

	var scenarios []struct {
		Label  string
		Prompt string
	}

	if *promptFlag != "" {
		scenarios = append(scenarios, struct {
			Label  string
			Prompt string
		}{
			Label:  "Custom Prompt",
			Prompt: *promptFlag,
		})
	} else if *promptFileFlag != "" {
		content, err := os.ReadFile(*promptFileFlag)
		if err != nil {
			log.Fatalf("Failed to read prompt file %s: %v", *promptFileFlag, err)
		}
		scenarios = append(scenarios, struct {
			Label  string
			Prompt string
		}{
			Label:  "File Prompt",
			Prompt: string(content),
		})
	} else if *suiteFlag {
		scenarios = []struct {
			Label  string
			Prompt string
		}{
			{
				Label:  "1. Short Query (Low Prefill)",
				Prompt: "What is the capital of France?",
			},
			{
				Label:  "2. Medium Generation (Standard Chat)",
				Prompt: "Explain the difference between concurrency and parallelism in two clear sentences.",
			},
			{
				Label:  "3. Reasoning / Code Task (High Decode)",
				Prompt: "Write a short Python function to calculate the Fibonacci series using memoization.",
			},
		}
	} else {
		scenarios = append(scenarios, struct {
			Label  string
			Prompt string
		}{
			Label:  "Standard Benchmark",
			Prompt: "Explain the theory of general relativity in 3 concise bullet points.",
		})
	}

	var reports []*benchmark.SuiteReport

	for _, sc := range scenarios {
		benchCfg := benchmark.Config{
			WarmupRuns:   *warmupFlag,
			MeasuredRuns: *runsFlag,
			Prompt:       sc.Prompt,
			Label:        sc.Label,
		}

		if !*jsonFlag {
			fmt.Printf("\n==> Running workload: %s (%d warmup, %d runs)...\n", sc.Label, *warmupFlag, *runsFlag)
		}

		report, err := benchmark.Run(ctx, model, benchCfg, "LiteRT-LM", *backendFlag, modelPath)
		if err != nil {
			log.Fatalf("Benchmark error: %v", err)
		}
		reports = append(reports, report)

		if !*jsonFlag {
			fmt.Print(report.FormatTable())
		}
	}

	if *jsonFlag {
		var outStr string
		if len(reports) == 1 {
			outStr, _ = reports[0].ToJSON()
		} else {
			bytes, _ := benchmark.MarshalIndentJSON(reports)
			outStr = string(bytes)
		}
		fmt.Println(outStr)

		if *outputFlag != "" {
			if err := os.WriteFile(*outputFlag, []byte(outStr), 0644); err != nil {
				log.Fatalf("Failed to save output to %s: %v", *outputFlag, err)
			}
			fmt.Fprintf(os.Stderr, "==> Results saved to %s\n", *outputFlag)
		}
	} else if *outputFlag != "" {
		var sb strings.Builder
		for _, r := range reports {
			sb.WriteString(r.FormatTable())
		}
		if err := os.WriteFile(*outputFlag, []byte(sb.String()), 0644); err != nil {
			log.Fatalf("Failed to save output to %s: %v", *outputFlag, err)
		}
		fmt.Printf("==> Report saved to %s\n", *outputFlag)
	}
}
