package router

import (
	"log/slog"
	"net/http"

	"github.com/aminshahid573/taskmanager/internal/handler"
	"github.com/aminshahid573/taskmanager/internal/middleware"
	"github.com/aminshahid573/taskmanager/internal/ratelimit"
	"github.com/aminshahid573/taskmanager/internal/service"
)

// RouterConfig holds all dependencies needed for route setup.
type RouterConfig struct {
	AuthHandler *handler.AuthHandler
	UserHandler *handler.UserHandler
	OrgHandler  *handler.OrgHandler
	TaskHandler *handler.TaskHandler

	AuthService *service.AuthService

	RateLimiterMiddleware func(http.Handler) http.Handler
	RateLimiter           *ratelimit.RateLimiter

	Logger *slog.Logger
}

// Setup initializes all routes and returns the configured HTTP handler.
func Setup(config RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Create authentication middleware
	authMiddleware := middleware.Authenticate(config.AuthService, config.Logger)

	// Register all routes
	registerPublicRoutes(mux)
	registerAuthRoutes(mux, config.AuthHandler, authMiddleware)
	registerUserRoutes(mux, config.UserHandler, authMiddleware)
	registerOrgRoutes(mux, config.OrgHandler, authMiddleware)
	registerTaskRoutes(mux, config.TaskHandler, authMiddleware)
	registerAdminRoutes(mux, config.RateLimiter, config.Logger, authMiddleware)

	// Build middleware chain (applied in reverse order)
	var handler http.Handler = mux
	handler = middleware.Recovery(config.Logger)(handler)
	handler = middleware.RequestID()(handler)
	handler = middleware.Logging(config.Logger)(handler)

	if config.RateLimiterMiddleware != nil {
		handler = config.RateLimiterMiddleware(handler)
	}

	return handler
}

