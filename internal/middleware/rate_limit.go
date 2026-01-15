package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aminshahid573/taskmanager/internal/cache"
	"github.com/aminshahid573/taskmanager/internal/config"
)

func RateLimit(redis *cache.RedisClient, cfg config.RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as identifier
			identifier := r.RemoteAddr

			// Check rate limit
			allowed, err := checkRateLimit(r.Context(), redis, identifier, cfg)
			if err != nil {
				// Log error but allow request to proceed
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"code":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkRateLimit(ctx context.Context, redis *cache.RedisClient, identifier string, cfg config.RateLimitConfig) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", identifier)

	count, err := redis.Incr(ctx, key)
	if err != nil {
		return false, err
	}

	// Set expiration on first request
	if count == 1 {
		if err := redis.Expire(ctx, key, time.Minute); err != nil {
			return false, err
		}
	}

	return count <= int64(cfg.RequestsPerMinute), nil
}

