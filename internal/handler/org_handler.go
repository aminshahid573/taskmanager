package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/service"
	"github.com/aminshahid573/taskmanager/internal/validator"
)

// OrgService defines the behavior OrgHandler needs from the organization service.
type OrgService interface {
	Create(ctx context.Context, userID uuid.UUID, req domain.CreateOrgRequest) (*domain.Organization, error)
	Get(ctx context.Context, userID, orgID uuid.UUID) (*domain.Organization, error)
	List(ctx context.Context, userID uuid.UUID) ([]*domain.Organization, error)
	Update(ctx context.Context, userID, orgID uuid.UUID, req domain.UpdateOrgRequest) (*domain.Organization, error)
	Delete(ctx context.Context, userID, orgID uuid.UUID) error
	AddMember(ctx context.Context, userID, orgID uuid.UUID, req domain.AddMemberRequest) error
	RemoveMember(ctx context.Context, userID, orgID, memberUserID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, userID, orgID, memberUserID uuid.UUID, req domain.UpdateRoleRequest) error
}

type OrgHandler struct {
	orgService OrgService
	logger     *slog.Logger
}

func NewOrgHandler(orgService *service.OrgService, logger *slog.Logger) *OrgHandler {
	return &OrgHandler{
		orgService: orgService,
		logger:     logger,
	}
}

func (h *OrgHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))

	var req domain.CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateCreateOrg(req); err != nil {
		respondError(w, err)
		return
	}

	org, err := h.orgService.Create(r.Context(), userID, req)
	if err != nil {
		h.logger.Error("Failed to create organization", "error", err, "user_id", userID)
		respondError(w, err)
		return
	}

	h.logger.Info("Organization created", "org_id", org.ID, "user_id", userID)
	respondJSON(w, http.StatusCreated, org)
}

func (h *OrgHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("id"))

	org, err := h.orgService.Get(r.Context(), userID, orgID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, org)
}

func (h *OrgHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))

	orgs, err := h.orgService.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to list organizations", "error", err, "user_id", userID)
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organizations": orgs,
	})
}

func (h *OrgHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("id"))

	var req domain.UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	org, err := h.orgService.Update(r.Context(), userID, orgID, req)
	if err != nil {
		h.logger.Error("Failed to update organization", "error", err, "org_id", orgID)
		respondError(w, err)
		return
	}

	h.logger.Info("Organization updated", "org_id", org.ID, "user_id", userID)
	respondJSON(w, http.StatusOK, org)
}

func (h *OrgHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("id"))

	if err := h.orgService.Delete(r.Context(), userID, orgID); err != nil {
		h.logger.Error("Failed to delete organization", "error", err, "org_id", orgID)
		respondError(w, err)
		return
	}

	h.logger.Info("Organization deleted", "org_id", orgID, "user_id", userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("id"))

	var req domain.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateEmail(req.UserEmail); err != nil {
		respondError(w, err)
		return
	}

	if err := validator.ValidateRole(req.Role); err != nil {
		respondError(w, err)
		return
	}

	if err := h.orgService.AddMember(r.Context(), userID, orgID, req); err != nil {
		h.logger.Error("Failed to add member", "error", err, "org_id", orgID)
		respondError(w, err)
		return
	}

	h.logger.Info("Member added to organization", "org_id", orgID, "email", req.UserEmail)
	respondJSON(w, http.StatusCreated, map[string]string{
		"message": "Member added successfully",
	})
}

func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("id"))
	memberUserID := mustParseUUID(r.PathValue("userId"))

	if err := h.orgService.RemoveMember(r.Context(), userID, orgID, memberUserID); err != nil {
		h.logger.Error("Failed to remove member", "error", err, "org_id", orgID, "member_id", memberUserID)
		respondError(w, err)
		return
	}

	h.logger.Info("Member removed from organization", "org_id", orgID, "member_id", memberUserID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *OrgHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("id"))
	memberUserID := mustParseUUID(r.PathValue("userId"))

	var req domain.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateRole(req.Role); err != nil {
		respondError(w, err)
		return
	}

	if err := h.orgService.UpdateMemberRole(r.Context(), userID, orgID, memberUserID, req); err != nil {
		h.logger.Error("Failed to update member role", "error", err, "org_id", orgID, "member_id", memberUserID)
		respondError(w, err)
		return
	}

	h.logger.Info("Member role updated", "org_id", orgID, "member_id", memberUserID, "role", req.Role)
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Role updated successfully",
	})
}

