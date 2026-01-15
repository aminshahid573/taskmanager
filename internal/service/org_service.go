package service

import (
	"context"

	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
	"github.com/google/uuid"
)

// OrgRepository defines the behavior OrgService needs from the organization repository.
type OrgRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, member *domain.OrgMember) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role domain.Role) error
	IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error)
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error)
}

type OrgService struct {
	orgRepo  OrgRepository
	userRepo UserRepository
}

func NewOrgService(orgRepo *repository.OrgRepository, userRepo *repository.UserRepository) *OrgService {
	return &OrgService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}
}

func (s *OrgService) Create(ctx context.Context, userID uuid.UUID, req domain.CreateOrgRequest) (*domain.Organization, error) {
	org := &domain.Organization{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrgService) Get(ctx context.Context, userID, orgID uuid.UUID) (*domain.Organization, error) {
	// Check membership
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	return s.orgRepo.GetByID(ctx, orgID)
}

func (s *OrgService) List(ctx context.Context, userID uuid.UUID) ([]*domain.Organization, error) {
	return s.orgRepo.ListByUser(ctx, userID)
}

func (s *OrgService) Update(ctx context.Context, userID, orgID uuid.UUID, req domain.UpdateOrgRequest) (*domain.Organization, error) {
	// Check permissions (only owner or admin)
	if err := s.checkAdminPermission(ctx, orgID, userID); err != nil {
		return nil, err
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Description != nil {
		org.Description = *req.Description
	}

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrgService) Delete(ctx context.Context, userID, orgID uuid.UUID) error {
	// Only owner can delete
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}

	if org.OwnerID != userID {
		return domain.ErrInsufficientPermissions
	}

	return s.orgRepo.Delete(ctx, orgID)
}

func (s *OrgService) AddMember(ctx context.Context, userID, orgID uuid.UUID, req domain.AddMemberRequest) error {
	// Check permissions
	if err := s.checkAdminPermission(ctx, orgID, userID); err != nil {
		return err
	}

	// Get user by email
	newUser, err := s.userRepo.GetByEmail(ctx, req.UserEmail)
	if err != nil {
		return err
	}

	// Check if already a member
	isMember, err := s.orgRepo.IsMember(ctx, orgID, newUser.ID)
	if err != nil {
		return err
	}
	if isMember {
		return domain.ErrAlreadyExists.WithDetails(map[string]string{
			"user": "already a member",
		})
	}

	member := &domain.OrgMember{
		OrgID:  orgID,
		UserID: newUser.ID,
		Role:   req.Role,
	}

	return s.orgRepo.AddMember(ctx, member)
}

func (s *OrgService) RemoveMember(ctx context.Context, userID, orgID, memberUserID uuid.UUID) error {
	// Check permissions
	if err := s.checkAdminPermission(ctx, orgID, userID); err != nil {
		return err
	}

	// Cannot remove owner
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}

	if org.OwnerID == memberUserID {
		return domain.ErrCannotDeleteOwner
	}

	return s.orgRepo.RemoveMember(ctx, orgID, memberUserID)
}

func (s *OrgService) UpdateMemberRole(ctx context.Context, userID, orgID, memberUserID uuid.UUID, req domain.UpdateRoleRequest) error {
	// Check permissions
	if err := s.checkAdminPermission(ctx, orgID, userID); err != nil {
		return err
	}

	// Cannot change owner role
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}

	if org.OwnerID == memberUserID {
		return domain.ErrCannotDeleteOwner.WithDetails(map[string]string{
			"role": "cannot change owner role",
		})
	}

	return s.orgRepo.UpdateMemberRole(ctx, orgID, memberUserID, req.Role)
}

func (s *OrgService) checkAdminPermission(ctx context.Context, orgID, userID uuid.UUID) error {
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return err
	}

	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin {
		return domain.ErrInsufficientPermissions
	}

	return nil
}
