package router

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MohammedZakirMemon/NexusLLM/internal/providers"
)

type LLMRouter struct {
	providers []providers.Provider
	logger    *slog.Logger
}

func New(logger *slog.Logger, provs ...providers.Provider) *LLMRouter {
	return &LLMRouter{providers: provs, logger: logger}
}

// Route sends the request to the first healthy provider and falls back on error.
func (r *LLMRouter) Route(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	var lastErr error
	for _, p := range r.providers {
		if !p.IsHealthy(ctx) {
			r.logger.Warn("provider unhealthy, skipping", "provider", p.Name())
			continue
		}
		resp, err := p.Complete(ctx, req)
		if err != nil {
			r.logger.Warn("provider error, trying fallback", "provider", p.Name(), "error", err)
			lastErr = err
			continue
		}
		r.logger.Info("request routed", "provider", p.Name(), "model", resp.Model, "latency_ms", resp.LatencyMS)
		return resp, nil
	}
	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// Cheapest returns the lowest-cost healthy provider for cost-optimized routing.
func (r *LLMRouter) Cheapest() providers.Provider {
	for _, p := range r.providers {
		if p.IsHealthy(context.Background()) {
			return p
		}
	}
	return nil
}
