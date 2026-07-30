package providers

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

type CompletionResponse struct {
	ID               string  `json:"id"`
	Model            string  `json:"model"`
	Provider         string  `json:"provider"`
	Content          string  `json:"content"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	LatencyMS        int64   `json:"latency_ms"`
	CostUSD          float64 `json:"cost_usd"`
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	IsHealthy(ctx context.Context) bool
}
