package router

import (
	"net/http"

	"github.com/aminshahid573/taskmanager/internal/handler"
)

// registerAuthRoutes registers authentication-related routes.
func registerAuthRoutes(
	mux *http.ServeMux,
	h *handler.AuthHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	// Public auth routes
	mux.HandleFunc("POST /api/v1/auth/signup", h.Signup)
	mux.HandleFunc("POST /api/v1/auth/verify-otp", h.VerifyOTP)
	mux.HandleFunc("POST /api/v1/auth/resend-otp", h.ResendOTP)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.RefreshToken)

	// Protected auth routes
	mux.Handle("POST /api/v1/auth/logout", authMiddleware(http.HandlerFunc(h.Logout)))
}

