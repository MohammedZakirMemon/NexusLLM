package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AnthropicProvider struct {
	apiKey     string
	httpClient *http.Client
}

func NewAnthropic(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}

	body, _ := json.Marshal(map[string]any{
		"model":      model,
		"messages":   req.Messages,
		"max_tokens": max(req.MaxTokens, 1024),
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID      string `json:"id"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", result.Error.Message)
	}

	content := ""
	if len(result.Content) > 0 {
		content = result.Content[0].Text
	}

	latency := time.Since(start).Milliseconds()
	totalTokens := result.Usage.InputTokens + result.Usage.OutputTokens
	cost := float64(result.Usage.InputTokens)*0.0000008 + float64(result.Usage.OutputTokens)*0.0000024

	return &CompletionResponse{
		ID:               result.ID,
		Model:            model,
		Provider:         p.Name(),
		Content:          content,
		PromptTokens:     result.Usage.InputTokens,
		CompletionTokens: result.Usage.OutputTokens,
		TotalTokens:      totalTokens,
		LatencyMS:        latency,
		CostUSD:          cost,
	}, nil
}

func (p *AnthropicProvider) IsHealthy(ctx context.Context) bool {
	return p.apiKey != ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
