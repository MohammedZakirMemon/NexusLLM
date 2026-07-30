package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	OpenAIKey       string
	AnthropicKey    string
	GeminiKey       string
	RateLimitRPM    int
	CacheTTLSeconds int
	Environment     string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nexusllm?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me-in-production"),
		OpenAIKey:       getEnv("OPENAI_API_KEY", ""),
		AnthropicKey:    getEnv("ANTHROPIC_API_KEY", ""),
		GeminiKey:       getEnv("GEMINI_API_KEY", ""),
		RateLimitRPM:    getEnvInt("RATE_LIMIT_RPM", 60),
		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 3600),
		Environment:     getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
