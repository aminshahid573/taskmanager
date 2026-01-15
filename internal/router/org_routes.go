package router

import (
	"net/http"

	"github.com/aminshahid573/taskmanager/internal/handler"
)

// registerOrgRoutes registers organization-related routes.
func registerOrgRoutes(
	mux *http.ServeMux,
	h *handler.OrgHandler,
	authMiddleware func(http.Handler) http.Handler,
) {
	if h == nil {
		return
	}

	mux.Handle("POST /api/v1/organizations", authMiddleware(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/organizations", authMiddleware(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/organizations/{id}", authMiddleware(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /api/v1/organizations/{id}", authMiddleware(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /api/v1/organizations/{id}", authMiddleware(http.HandlerFunc(h.Delete)))
	mux.Handle("POST /api/v1/organizations/{id}/members", authMiddleware(http.HandlerFunc(h.AddMember)))
	mux.Handle("DELETE /api/v1/organizations/{id}/members/{userId}", authMiddleware(http.HandlerFunc(h.RemoveMember)))
	mux.Handle("PUT /api/v1/organizations/{id}/members/{userId}/role", authMiddleware(http.HandlerFunc(h.UpdateMemberRole)))
}

