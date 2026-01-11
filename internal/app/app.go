package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aminshahid573/taskmanager/internal/cache"
	"github.com/aminshahid573/taskmanager/internal/config"
	"github.com/aminshahid573/taskmanager/internal/database"
	"github.com/aminshahid573/taskmanager/internal/ratelimit"
)

func Run(cfg *config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//track cleanup functions (LIFO order)
	cleanupFuncs := make([]func() error,0)
	defer func() {
		//execute cleanup in reverse order
		for i := len(cleanupFuncs) - 1; i > 0; i-- {
			if err := cleanupFuncs[i](); err != nil {
				slog.Error("Cleanup failed", "error", err)
			}
		}
	}()

	//initialize postgress
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return fmt.Errorf("postgres connection: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() error {
		slog.Info("Closing database connection")
		return db.Close()
	})

	// Initialize Redis
	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis connection: %w", err)
	}
	cleanupFuncs = append(cleanupFuncs, func() error {
		slog.Info("Closing Redis connection")
		return redisClient.Close()
	})

		var rateLimiterMiddleware func(http.Handler) http.Handler
	var rateLimiterInstance *ratelimit.RateLimiter
	if cfg.RateLimit.Enabled {
		rateLimiter, err := ratelimit.NewRateLimiter(cfg, redisClient)
		if err != nil {
			return fmt.Errorf("rate limiter initialization: %w", err)
		}

		rateLimiterInstance = rateLimiter // store reference

		cleanupFuncs = append(cleanupFuncs, func() error {
			slog.Info("Closing rate limiter")
			return rateLimiter.Close()
		})
		rateLimiterMiddleware = rateLimiter.Middleware
		slog.Info("Rate limiting enabled",
			"limit", cfg.RateLimit.RequestsPerMinute,
			"window", cfg.RateLimit.Window,
		)
	} else {
		// No-op middleware if rate limiting is disabled
		rateLimiterMiddleware = func(next http.Handler) http.Handler {
			return next
		}
		slog.Info("Rate limiting disabled")
	}



	return nil
}
