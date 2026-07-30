package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MohammedZakirMemon/NexusLLM/internal/auth"
	"github.com/MohammedZakirMemon/NexusLLM/internal/metrics"
)

type AuthMiddleware struct {
	keyStore  *auth.APIKeyStore
	jwtSecret string
}

func NewAuthMiddleware(keyStore *auth.APIKeyStore, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{keyStore: keyStore, jwtSecret: jwtSecret}
}

func (a *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()

		// API Key auth: "Bearer nxl_..."
		if strings.HasPrefix(header, "Bearer nxl_") {
			rawKey := strings.TrimPrefix(header, "Bearer ")
			key, err := a.keyStore.Validate(ctx, rawKey)
			if err != nil {
				metrics.RequestsTotal.WithLabelValues("unknown", "unknown", "401").Inc()
				http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
				return
			}
			ctx = context.WithValue(ctx, contextKeyTenantID, key.TenantID)
			ctx = context.WithValue(ctx, contextKey("api_key_id"), key.ID)
			ctx = context.WithValue(ctx, contextKey("rpm_limit"), key.RPMLimit)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// JWT auth: "Bearer eyJ..."
		if strings.HasPrefix(header, "Bearer ") {
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateJWT(tokenStr, a.jwtSecret)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx = context.WithValue(ctx, contextKeyTenantID, claims.TenantID)
			ctx = context.WithValue(ctx, contextKey("role"), claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		http.Error(w, `{"error":"unsupported auth scheme"}`, http.StatusUnauthorized)
	})
}
