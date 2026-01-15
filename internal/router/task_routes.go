package router

import (
	"net/http"

	"github.com/aminshahid573/taskmanager/internal/handler"
)

// registerTaskRoutes registers task-related routes.
func registerTaskRoutes(
	mux *http.ServeMux,
	h *handler.TaskHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	if h == nil {
		return
	}

	mux.Handle("POST /api/v1/organizations/{orgId}/tasks", authMiddleware(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/organizations/{orgId}/tasks", authMiddleware(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/organizations/{orgId}/tasks/{id}", authMiddleware(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /api/v1/organizations/{orgId}/tasks/{id}", authMiddleware(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /api/v1/organizations/{orgId}/tasks/{id}", authMiddleware(http.HandlerFunc(h.Delete)))
	mux.Handle("PUT /api/v1/organizations/{orgId}/tasks/{id}/assign", authMiddleware(http.HandlerFunc(h.Assign)))
}

