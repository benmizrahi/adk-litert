package embeddedllm_test

import (
	"context"
	"os"
	"testing"

	embeddedllm "adk-embedded-litert"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

func getTestModel(b *testing.B) *embeddedllm.Model {
	b.Helper()
	modelPath := os.Getenv("LITERT_MODEL_PATH")
	if modelPath == "" {
		if _, err := os.Stat("gemma3-270m-it-q8.litertlm"); err == nil {
			modelPath = "gemma3-270m-it-q8.litertlm"
		} else if _, err := os.Stat("tiny_garden.litertlm"); err == nil {
			modelPath = "tiny_garden.litertlm"
		} else if _, err := os.Stat("model.litertlm"); err == nil {
			modelPath = "model.litertlm"
		} else {
			b.Skip("Skipping benchmark: no .litertlm model found on path (set LITERT_MODEL_PATH)")
		}
	}

	cfg := embeddedllm.DefaultConfig(modelPath)
	cfg.Backend = embeddedllm.BackendCPU
	cfg.MaxSeqLen = 2048

	m, err := embeddedllm.NewModel(context.Background(), cfg)
	if err != nil {
		b.Fatalf("Failed to initialize model for benchmark: %v", err)
	}
	return m
}

// BenchmarkLiteRT_TTFT measures Time To First Token for short queries.
func BenchmarkLiteRT_TTFT(b *testing.B) {
	m := getTestModel(b)
	defer m.Close()

	ctx := context.Background()
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: "What is 2 + 2?"},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream := m.GenerateContent(ctx, req, true)
		for resp, err := range stream {
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
			if len(resp.Content.Parts) > 0 && resp.Content.Parts[0].Text != "" {
				// Stop immediately on receiving the first token to isolate TTFT
				break
			}
		}
	}
}

// BenchmarkLiteRT_Throughput measures complete generation throughput and latency.
func BenchmarkLiteRT_Throughput(b *testing.B) {
	m := getTestModel(b)
	defer m.Close()

	ctx := context.Background()
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: "Explain the solar system in 3 bullet points."},
				},
			},
		},
	}

	b.ResetTimer()
	var totalTokens int
	for i := 0; i < b.N; i++ {
		stream := m.GenerateContent(ctx, req, true)
		for resp, err := range stream {
			if err != nil {
				b.Fatalf("Generation failed: %v", err)
			}
			for _, part := range resp.Content.Parts {
				if part.Text != "" {
					totalTokens++
				}
			}
		}
	}
	b.ReportMetric(float64(totalTokens)/float64(b.N), "tokens/op")
}
