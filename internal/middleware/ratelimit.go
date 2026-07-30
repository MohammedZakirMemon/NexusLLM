package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Allow checks and increments the token bucket for the given key.
// Returns true if the request is allowed, false if rate limited.
func (rl *RateLimiter) Allow(ctx context.Context, tenantID string, rpmLimit int) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:%s:%d", tenantID, time.Now().Unix()/60)
	pipe := rl.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, rpmLimit, nil // fail open
	}
	count := int(incr.Val())
	remaining := rpmLimit - count
	if remaining < 0 {
		remaining = 0
	}
	return count <= rpmLimit, remaining, nil
}

func (rl *RateLimiter) Middleware(rpmLimit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, _ := r.Context().Value(contextKeyTenantID).(string)
			if tenantID == "" {
				tenantID = r.RemoteAddr
			}

			allowed, remaining, _ := rl.Allow(r.Context(), tenantID, rpmLimit)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rpmLimit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type contextKey string

const contextKeyTenantID contextKey = "tenant_id"
