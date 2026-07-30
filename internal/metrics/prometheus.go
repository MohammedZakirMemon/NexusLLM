package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nexusllm_requests_total",
		Help: "Total number of LLM requests",
	}, []string{"provider", "model", "status"})

	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nexusllm_request_duration_ms",
		Help:    "LLM request latency in milliseconds",
		Buckets: []float64{100, 250, 500, 1000, 2000, 5000, 10000},
	}, []string{"provider", "model"})

	TokensTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nexusllm_tokens_total",
		Help: "Total tokens consumed",
	}, []string{"provider", "model", "type"})

	CostTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nexusllm_cost_usd_total",
		Help: "Total cost in USD by provider",
	}, []string{"provider", "tenant"})

	CacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nexusllm_cache_hits_total",
		Help: "Total semantic cache hits",
	}, []string{"type"})

	RateLimitHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nexusllm_rate_limit_hits_total",
		Help: "Total rate limit rejections",
	}, []string{"tenant", "tier"})

	ActiveRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nexusllm_active_requests",
		Help: "Number of in-flight LLM requests",
	})
)

func Register() {
	prometheus.MustRegister(
		RequestsTotal,
		RequestDuration,
		TokensTotal,
		CostTotal,
		CacheHitsTotal,
		RateLimitHitsTotal,
		ActiveRequests,
	)
}
