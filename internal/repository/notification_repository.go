package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/domain"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create records a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *domain.TaskNotification) error {
	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	if notification.SentAt.IsZero() {
		notification.SentAt = time.Now()
	}

	query := `
		INSERT INTO task_notifications (id, task_id, user_id, notification_type, sent_at, status, retry_count, last_error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		notification.ID,
		notification.TaskID,
		notification.UserID,
		notification.NotificationType,
		notification.SentAt,
		notification.Status,
		notification.RetryCount,
		notification.LastError,
		notification.CreatedAt,
	)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	return nil
}

// WasNotificationSent checks if a notification of the given type was sent to the user for the task within the specified duration
func (r *NotificationRepository) WasNotificationSent(ctx context.Context, taskID, userID uuid.UUID, notificationType domain.NotificationType, within time.Duration) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM task_notifications
			WHERE task_id = $1
			AND user_id = $2
			AND notification_type = $3
			AND status = $4
			AND sent_at > $5
		)
	`

	cutoff := time.Now().Add(-within)
	var exists bool
	err := r.db.QueryRowContext(ctx, query, taskID, userID, notificationType, domain.NotificationStatusSent, cutoff).Scan(&exists)
	if err != nil {
		return false, domain.ErrDatabaseError.WithError(err)
	}

	return exists, nil
}

// GetPendingRetries returns failed notifications that can be retried
func (r *NotificationRepository) GetPendingRetries(ctx context.Context, maxRetries int) ([]*domain.TaskNotification, error) {
	query := `
		SELECT id, task_id, user_id, notification_type, sent_at, status, retry_count, last_error, created_at
		FROM task_notifications
		WHERE status = $1
		AND retry_count < $2
		ORDER BY created_at ASC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, domain.NotificationStatusFailed, maxRetries)
	if err != nil {
		return nil, domain.ErrDatabaseError.WithError(err)
	}
	defer rows.Close()

	var notifications []*domain.TaskNotification
	for rows.Next() {
		var n domain.TaskNotification
		err := rows.Scan(
			&n.ID, &n.TaskID, &n.UserID, &n.NotificationType,
			&n.SentAt, &n.Status, &n.RetryCount, &n.LastError, &n.CreatedAt,
		)
		if err != nil {
			return nil, domain.ErrDatabaseError.WithError(err)
		}
		notifications = append(notifications, &n)
	}

	return notifications, nil
}

// UpdateStatus updates the status of a notification
func (r *NotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.NotificationStatus, lastError *string) error {
	query := `
		UPDATE task_notifications
		SET status = $1, last_error = $2, retry_count = retry_count + 1, sent_at = $3
		WHERE id = $4
	`

	result, err := r.db.ExecContext(ctx, query, status, lastError, time.Now(), id)
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.ErrDatabaseError.WithError(err)
	}
	if rows == 0 {
		return errors.New("notification not found")
	}

	return nil
}

// MarkAsSent marks a notification as successfully sent
func (r *NotificationRepository) MarkAsSent(ctx context.Context, id uuid.UUID) error {
	return r.UpdateStatus(ctx, id, domain.NotificationStatusSent, nil)
}

// MarkAsFailed marks a notification as failed with an error message
func (r *NotificationRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	return r.UpdateStatus(ctx, id, domain.NotificationStatusFailed, &errMsg)
}

