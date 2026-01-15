package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aminshahid573/taskmanager/internal/cache"
	"github.com/aminshahid573/taskmanager/internal/config"
	"github.com/aminshahid573/taskmanager/internal/database"
	"github.com/aminshahid573/taskmanager/internal/handler"
	"github.com/aminshahid573/taskmanager/internal/ratelimit"
	"github.com/aminshahid573/taskmanager/internal/repository"
	"github.com/aminshahid573/taskmanager/internal/router"
	"github.com/aminshahid573/taskmanager/internal/service"
	"github.com/aminshahid573/taskmanager/internal/worker"
)

func Run(cfg *config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//track cleanup functions (LIFO order)
	cleanupFuncs := make([]func() error, 0)
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

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	orgRepo := repository.NewOrgRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, redisClient, cfg.JWT)
	otpService := service.NewOTPService(redisClient)
	orgService := service.NewOrgService(orgRepo, userRepo)
	taskService := service.NewTaskService(taskRepo, orgRepo)

	// Initialize workers
	emailWorker, err := worker.NewEmailWorker(cfg.Email, logger)
	if err != nil {
		return fmt.Errorf("email worker initialization: %w", err)
	}

	reminderWorker := worker.NewReminderWorker(taskRepo, userRepo, emailWorker, logger)

	// Start background workers
	workers := StartWorkers(ctx, emailWorker, reminderWorker)
	cleanupFuncs = append(cleanupFuncs, func() error {
		slog.Info("Stopping background workers")
		workers.Cancel()
		return nil
	})

		// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, otpService, userRepo, emailWorker, logger)
	userHandler := handler.NewUserHandler(userRepo)
	orgHandler := handler.NewOrgHandler(orgService, logger)
	taskHandler := handler.NewTaskHandler(taskService, userRepo, orgRepo, emailWorker, logger)

	// Setup router
	mux := router.Setup(
		router.RouterConfig{
			AuthHandler:           authHandler,
			UserHandler:           userHandler,
			OrgHandler:            orgHandler,
			TaskHandler:           taskHandler,
			AuthService:           authService,
			RateLimiterMiddleware: rateLimiterMiddleware,
			RateLimiter:           rateLimiterInstance,
			Logger:                logger,
		},
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

		// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Starting HTTP server", "address", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-shutdown:
		slog.Info("Shutdown signal received", "signal", sig.String())

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			time.Duration(cfg.Server.ShutdownTimeout)*time.Second,
		)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("Graceful shutdown failed", "error", err)
			if err := srv.Close(); err != nil {
				return fmt.Errorf("force shutdown: %w", err)
			}
		}

		// Wait for background workers to finish
		workers.Cancel()

		done := make(chan struct{})
		go func() {
			workers.WG.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("Background workers stopped")
		case <-time.After(5 * time.Second):
			slog.Warn("Background workers shutdown timeout")
		}
	}

	return nil

}
