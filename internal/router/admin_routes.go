package router

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aminshahid573/taskmanager/internal/ratelimit"
)

// registerAdminRoutes registers admin/monitoring endpoints.
// The routes are protected by the provided authMiddleware.
func registerAdminRoutes(
	mux *http.ServeMux,
	rl *ratelimit.RateLimiter,
	logger *slog.Logger,
	authMiddleware func(http.Handler) http.Handler,
) {
	mux.Handle("GET /admin/ratelimit/stats", authMiddleware(http.HandlerFunc(handleRateLimitStats(rl, logger))))
}

// handleRateLimitStats returns basic rate limiter statistics.
func handleRateLimitStats(rl *ratelimit.RateLimiter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rl == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"enabled":false}`))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		stats, err := rl.GetStats(ctx)
		if err != nil {
			logger.Error("Failed to get rate limit stats", "error", err)
			http.Error(w, "Failed to get stats", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"enabled":true,"active_limits":%d,"sample_size":%d}`,
			stats.ActiveLimits, len(stats.Limits))
	}
}

