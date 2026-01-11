package ratelimit

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Middleware returns the rate limiting middleware with metrics
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Extract IP and endpoint for metrics
		ip := extractIP(r)
		endpoint := r.URL.Path
		key := fmt.Sprintf("rate_limit:%s", ip)

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		now := time.Now().UnixMilli()
		windowMs := rl.window.Milliseconds()

		// Execute Lua script
		result, err := rl.script.Run(ctx, rl.client,
			[]string{key},
			rl.limit,
			windowMs,
			now,
		).Int64Slice()

		// Record Redis latency
		duration := time.Since(startTime).Seconds()
		rl.metrics.redisLatency.WithLabelValues("rate_check").Observe(duration)

		if err != nil {
			log.Printf("Redis error: %v", err)
			rl.metrics.redisErrors.WithLabelValues("rate_check", classifyError(err)).Inc()

			// Fail open: allow request if Redis is down
			next.ServeHTTP(w, r)
			return
		}

		allowed := result[0] == 1
		remaining := result[1]
		resetTime := result[2] // Unix timestamp in milliseconds

		// Convert reset time to seconds for header (Unix timestamp)
		resetTimeSec := resetTime / 1000

		// Calculate seconds until reset for Retry-After header
		retryAfterSec := (resetTime - now) / 1000
		if retryAfterSec < 0 {
			retryAfterSec = 0
		}

		// Always set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTimeSec, 10))

		// Record remaining quota distribution
		quotaPercent := float64(remaining) / float64(rl.limit) * 100
		rl.metrics.remainingQuota.WithLabelValues(endpoint).Observe(quotaPercent)

		// Update reset time gauge
		rl.metrics.rateLimitResetTime.WithLabelValues(ip).Set(float64(resetTimeSec))

		if !allowed {
			w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSec, 10))

			// Record blocked request metrics
			rl.metrics.requestsBlocked.WithLabelValues(endpoint, ip).Inc()

			log.Printf("Rate limit exceeded for IP %s on endpoint %s (reset in %ds)",
				ip, endpoint, retryAfterSec)

			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Record allowed request
		rl.metrics.requestsAllowed.WithLabelValues(endpoint).Inc()

		next.ServeHTTP(w, r)
	})
}

