package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/MohammedZakirMemon/NexusLLM/internal/auth"
	"github.com/MohammedZakirMemon/NexusLLM/internal/cache"
	"github.com/MohammedZakirMemon/NexusLLM/internal/config"
	"github.com/MohammedZakirMemon/NexusLLM/internal/db"
	"github.com/MohammedZakirMemon/NexusLLM/internal/metrics"
	"github.com/MohammedZakirMemon/NexusLLM/internal/middleware"
	"github.com/MohammedZakirMemon/NexusLLM/internal/providers"
	"github.com/MohammedZakirMemon/NexusLLM/internal/router"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	// Database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	if err := database.Migrate(); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	// Redis
	rdbOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("invalid redis url", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(rdbOpts)

	// Providers
	providerList := []providers.Provider{}
	if cfg.OpenAIKey != "" {
		providerList = append(providerList, providers.NewOpenAI(cfg.OpenAIKey))
	}
	if cfg.AnthropicKey != "" {
		providerList = append(providerList, providers.NewAnthropic(cfg.AnthropicKey))
	}
	if len(providerList) == 0 {
		logger.Warn("no LLM providers configured — set OPENAI_API_KEY or ANTHROPIC_API_KEY")
	}

	llmRouter := router.New(logger, providerList...)
	keyStore := auth.NewAPIKeyStore(database.DB)
	authMW := middleware.NewAuthMiddleware(keyStore, cfg.JWTSecret)
	rateLimiter := middleware.NewRateLimiter(rdb)
	semanticCache := cache.New(rdb, cfg.CacheTTLSeconds)

	metrics.Register()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "env": cfg.Environment})
	})

	// Metrics
	mux.Handle("GET /metrics", promhttp.Handler())

	// Chat completions — authenticated + rate limited
	mux.Handle("POST /v1/chat/completions",
		authMW.Authenticate(
			rateLimiter.Middleware(cfg.RateLimitRPM)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req providers.CompletionRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
						return
					}

					// Semantic cache check
					msgMaps := make([]map[string]string, len(req.Messages))
					for i, m := range req.Messages {
						msgMaps[i] = map[string]string{"role": m.Role, "content": m.Content}
					}
					if cached, ok := semanticCache.Get(r.Context(), msgMaps, req.Model); ok {
						metrics.CacheHitsTotal.WithLabelValues("hit").Inc()
						w.Header().Set("X-Cache", "HIT")
						w.Header().Set("Content-Type", "application/json")
						w.Write([]byte(cached))
						return
					}
					metrics.CacheHitsTotal.WithLabelValues("miss").Inc()

					metrics.ActiveRequests.Inc()
					defer metrics.ActiveRequests.Dec()

					resp, err := llmRouter.Route(r.Context(), &req)
					if err != nil {
						metrics.RequestsTotal.WithLabelValues("unknown", req.Model, "500").Inc()
						http.Error(w, `{"error":"all providers failed"}`, http.StatusBadGateway)
						return
					}

					metrics.RequestsTotal.WithLabelValues(resp.Provider, resp.Model, "200").Inc()
					metrics.RequestDuration.WithLabelValues(resp.Provider, resp.Model).Observe(float64(resp.LatencyMS))
					metrics.TokensTotal.WithLabelValues(resp.Provider, resp.Model, "prompt").Add(float64(resp.PromptTokens))
					metrics.TokensTotal.WithLabelValues(resp.Provider, resp.Model, "completion").Add(float64(resp.CompletionTokens))

					out, _ := json.Marshal(resp)
					semanticCache.Set(r.Context(), msgMaps, req.Model, string(out))

					w.Header().Set("X-Cache", "MISS")
					w.Header().Set("Content-Type", "application/json")
					w.Write(out)
				}),
			),
		),
	)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("NexusLLM gateway starting", "port", cfg.Port, "env", cfg.Environment)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
