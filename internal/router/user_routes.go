package router

import (
	"net/http"

	"github.com/aminshahid573/taskmanager/internal/handler"
)

// registerUserRoutes registers user-related routes.
func registerUserRoutes(
	mux *http.ServeMux,
	h *handler.UserHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	if h == nil {
		return
	}

	mux.Handle("GET /api/v1/users/me", authMiddleware(http.HandlerFunc(h.GetProfile)))
	mux.Handle("GET /api/v1/users/{id}", authMiddleware(http.HandlerFunc(h.GetUserByID)))
	mux.Handle("PATCH /api/v1/users/me", authMiddleware(http.HandlerFunc(h.UpdateProfile)))
}

