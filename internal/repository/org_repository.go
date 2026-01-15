package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/google/uuid"
)

type OrgRepository struct {
	db *sql.DB
}

func NewOrgRepository(db *sql.DB) *OrgRepository {
	return &OrgRepository{db: db}
}

func (r *OrgRepository) Create(ctx context.Context, org *domain.Organization) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	defer tx.Rollback()

	// Create organization
	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	query := `
		INSERT INTO organizations (id, name, description, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.ExecContext(ctx, query,
		org.ID, org.Name, org.Description, org.OwnerID,
		org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	// Add owner as member
	memberQuery := `
		INSERT INTO org_members (id, org_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.ExecContext(ctx, memberQuery,
		uuid.New(), org.ID, org.OwnerID, domain.RoleOwner,
		time.Now(), time.Now(),
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	return tx.Commit()
}

func (r *OrgRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at, deleted_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`

	var org domain.Organization
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Description, &org.OwnerID,
		&org.CreatedAt, &org.UpdatedAt, &org.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewAppError(domain.ErrCodeOrgNotFound, "Organization not found", 404)
		}
		return nil, domain.ErrDatabaseError.WithError(err)
	}

	return &org, nil
}

func (r *OrgRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Organization, error) {
	query := `
		SELECT o.id, o.name, o.description, o.owner_id, o.created_at, o.updated_at
		FROM organizations o
		INNER JOIN org_members om ON o.id = om.org_id
		WHERE om.user_id = $1 AND o.deleted_at IS NULL AND om.deleted_at IS NULL
		ORDER BY o.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, domain.ErrDatabaseError.WithError(err)
	}
	defer rows.Close()

	var orgs []*domain.Organization
	for rows.Next() {
		var org domain.Organization
		err := rows.Scan(
			&org.ID, &org.Name, &org.Description, &org.OwnerID,
			&org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, domain.ErrDatabaseError.WithError(err)
		}
		orgs = append(orgs, &org)
	}

	return orgs, nil
}

func (r *OrgRepository) Update(ctx context.Context, org *domain.Organization) error {
	org.UpdatedAt = time.Now()

	query := `
		UPDATE organizations
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		org.Name, org.Description, org.UpdatedAt, org.ID,
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeOrgNotFound, "Organization not found", 404)
	}

	return nil
}

func (r *OrgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE organizations
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeOrgNotFound, "Organization not found", 404)
	}

	return nil
}

func (r *OrgRepository) AddMember(ctx context.Context, member *domain.OrgMember) error {
	member.ID = uuid.New()
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()

	query := `
		INSERT INTO org_members (id, org_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		member.ID, member.OrgID, member.UserID, member.Role,
		member.CreatedAt, member.UpdatedAt,
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	return nil
}

func (r *OrgRepository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `
		UPDATE org_members
		SET deleted_at = $1
		WHERE org_id = $2 AND user_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), orgID, userID)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.ErrNotMember
	}

	return nil
}

func (r *OrgRepository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrgMember, error) {
	query := `
		SELECT id, org_id, user_id, role, created_at, updated_at
		FROM org_members
		WHERE org_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	var member domain.OrgMember
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&member.ID, &member.OrgID, &member.UserID, &member.Role,
		&member.CreatedAt, &member.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotMember
		}
		return nil, domain.ErrDatabaseError.WithError(err)
	}

	return &member, nil
}

func (r *OrgRepository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role domain.Role) error {
	query := `
		UPDATE org_members
		SET role = $1, updated_at = $2
		WHERE org_id = $3 AND user_id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, role, time.Now(), orgID, userID)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.ErrNotMember
	}

	return nil
}

func (r *OrgRepository) IsMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM org_members
			WHERE org_id = $1 AND user_id = $2 AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(&exists)
	if err != nil {
		return false, domain.ErrDatabaseError.WithError(err)
	}

	return exists, nil
}
