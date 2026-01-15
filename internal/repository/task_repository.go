package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/domain"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	task.ID = uuid.New()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = domain.TaskStatusTodo

	query := `
		INSERT INTO tasks (id, org_id, title, description, status, assigned_to, due_date, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.OrgID, task.Title, task.Description, task.Status,
		task.AssignedTo, task.DueDate, task.CreatedBy,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*domain.Task, error) {
	query := `
		SELECT id, org_id, title, description, status, assigned_to, due_date, created_by, created_at, updated_at
		FROM tasks
		WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
	`

	var task domain.Task
	err := r.db.QueryRowContext(ctx, query, id, orgID).Scan(
		&task.ID, &task.OrgID, &task.Title, &task.Description, &task.Status,
		&task.AssignedTo, &task.DueDate, &task.CreatedBy,
		&task.CreatedAt, &task.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewAppError(domain.ErrCodeTaskNotFound, "Task not found", 404)
		}
		return nil, domain.ErrDatabaseError.WithError(err)
	}

	return &task, nil
}

func (r *TaskRepository) List(ctx context.Context, orgID uuid.UUID, query domain.ListTasksQuery) ([]*domain.Task, int, error) {
	// Build dynamic query
	var conditions []string
	var args []interface{}
	argPos := 1

	conditions = append(conditions, fmt.Sprintf("org_id = $%d", argPos))
	args = append(args, orgID)
	argPos++

	conditions = append(conditions, "deleted_at IS NULL")

	if query.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *query.Status)
		argPos++
	}

	if query.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argPos))
		args = append(args, *query.AssignedTo)
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, domain.ErrDatabaseError.WithError(err)
	}

	// Get paginated results
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.Page < 1 {
		query.Page = 1
	}
	offset := (query.Page - 1) * query.Limit

	listQuery := fmt.Sprintf(`
		SELECT id, org_id, title, description, status, assigned_to, due_date, created_by, created_at, updated_at
		FROM tasks
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, query.Limit, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, domain.ErrDatabaseError.WithError(err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		var task domain.Task
		err := rows.Scan(
			&task.ID, &task.OrgID, &task.Title, &task.Description, &task.Status,
			&task.AssignedTo, &task.DueDate, &task.CreatedBy,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, 0, domain.ErrDatabaseError.WithError(err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, total, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	task.UpdatedAt = time.Now()

	query := `
		UPDATE tasks
		SET title = $1, description = $2, status = $3, due_date = $4, updated_at = $5
		WHERE id = $6 AND org_id = $7 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title, task.Description, task.Status, task.DueDate, task.UpdatedAt,
		task.ID, task.OrgID,
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeTaskNotFound, "Task not found", 404)
	}

	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	query := `
		UPDATE tasks
		SET deleted_at = $1
		WHERE id = $2 AND org_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, orgID)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeTaskNotFound, "Task not found", 404)
	}

	return nil
}

func (r *TaskRepository) Assign(ctx context.Context, taskID, orgID, userID uuid.UUID) error {
	query := `
		UPDATE tasks
		SET assigned_to = $1, updated_at = $2
		WHERE id = $3 AND org_id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, userID, time.Now(), taskID, orgID)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return domain.NewAppError(domain.ErrCodeTaskNotFound, "Task not found", 404)
	}

	return nil
}

func (r *TaskRepository) GetDueSoonTasks(ctx context.Context, hours int) ([]*domain.Task, error) {
	query := `
		SELECT t.id, t.org_id, t.title, t.description, t.status, t.assigned_to, t.due_date, t.created_by, t.created_at, t.updated_at
		FROM tasks t
		WHERE t.due_date IS NOT NULL
		AND t.due_date > NOW()
		AND t.due_date <= NOW() + INTERVAL '1 hour' * $1
		AND t.status != $2
		AND t.deleted_at IS NULL
	`

	rows, err := r.db.QueryContext(ctx, query, hours, domain.TaskStatusDone)
	if err != nil {
		return nil, domain.ErrDatabaseError.WithError(err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		var task domain.Task
		err := rows.Scan(
			&task.ID, &task.OrgID, &task.Title, &task.Description, &task.Status,
			&task.AssignedTo, &task.DueDate, &task.CreatedBy,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, domain.ErrDatabaseError.WithError(err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

func (r *TaskRepository) GetOverdueTasks(ctx context.Context) ([]*domain.Task, error) {
	query := `
		SELECT t.id, t.org_id, t.title, t.description, t.status, t.assigned_to, t.due_date, t.created_by, t.created_at, t.updated_at
		FROM tasks t
		WHERE t.due_date IS NOT NULL
		AND t.due_date < NOW()
		AND t.status != $1
		AND t.deleted_at IS NULL
	`

	rows, err := r.db.QueryContext(ctx, query, domain.TaskStatusDone)
	if err != nil {
		return nil, domain.ErrDatabaseError.WithError(err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		var task domain.Task
		err := rows.Scan(
			&task.ID, &task.OrgID, &task.Title, &task.Description, &task.Status,
			&task.AssignedTo, &task.DueDate, &task.CreatedBy,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, domain.ErrDatabaseError.WithError(err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

