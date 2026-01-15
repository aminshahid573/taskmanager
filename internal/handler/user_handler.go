package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
)

type UserHandler struct {
	userRepo *repository.UserRepository
	logger   interface{} // slog.Logger type
}

func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

// GetProfile returns the current user's profile information
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware) - stored as string
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok || userIDStr == "" {
		respondError(w, domain.NewAppError(
			domain.ErrCodeUnauthorized,
			"Unauthorized",
			401,
		))
		return
	}

	userID := mustParseUUID(userIDStr)
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	// Return user profile without sensitive data
	profile := map[string]interface{}{
		"id":                user.ID,
		"email":             user.Email,
		"name":              user.Name,
		"email_verified":    user.EmailVerified,
		"email_verified_at": user.EmailVerifiedAt,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    profile,
	})
}

// GetUserByID returns a specific user's public profile
// GET /api/v1/users/{id}
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	if userIDStr == "" {
		respondError(w, domain.NewAppError(
			domain.ErrCodeValidationFailed,
			"User ID is required",
			400,
		))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondError(w, domain.NewAppError(
			domain.ErrCodeValidationFailed,
			"Invalid user ID format",
			400,
		))
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	// Return only public profile information
	profile := map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"created_at": user.CreatedAt,
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    profile,
	})
}

// UpdateProfile updates the current user's profile
// PATCH /api/v1/users/me
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok || userIDStr == "" {
		respondError(w, domain.NewAppError(
			domain.ErrCodeUnauthorized,
			"Unauthorized",
			401,
		))
		return
	}

	type UpdateRequest struct {
		Name string `json:"name,omitempty"`
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	// For now, we only support updating name
	// If needed, expand this to include other fields
	if req.Name == "" {
		respondError(w, domain.NewAppError(
			domain.ErrCodeValidationFailed,
			"Name cannot be empty",
			400,
		))
		return
	}

	userID := mustParseUUID(userIDStr)
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}

	// Update user
	user.Name = req.Name

	if err := h.userRepo.Update(r.Context(), user); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profile updated successfully",
		"data": map[string]interface{}{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"updated_at": user.UpdatedAt,
		},
	})
}

