package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
)

const (
	// MaxRetries is the maximum number of retry attempts for failed notifications
	MaxRetries = 3
	// RetryInterval is the base interval between retry attempts
	RetryInterval = 5 * time.Minute
)

type ReminderWorker struct {
	taskRepo         *repository.TaskRepository
	userRepo         *repository.UserRepository
	notificationRepo *repository.NotificationRepository
	emailWorker      *EmailWorker
	logger           *slog.Logger
}

func NewReminderWorker(
	taskRepo *repository.TaskRepository,
	userRepo *repository.UserRepository,
	notificationRepo *repository.NotificationRepository,
	emailWorker *EmailWorker,
	logger *slog.Logger,
) *ReminderWorker {
	return &ReminderWorker{
		taskRepo:         taskRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		emailWorker:      emailWorker,
		logger:           logger,
	}
}

func (w *ReminderWorker) Start(ctx context.Context) {
	w.logger.Info("Reminder worker started")

	// Main ticker for checking due tasks
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Retry ticker for failed notifications
	retryTicker := time.NewTicker(RetryInterval)
	defer retryTicker.Stop()

	// Run once after a short delay on startup to provide immediate feedback
	// while avoiding race conditions with other services starting up
	go func() {
		time.Sleep(5 * time.Second)
		w.checkAndSendReminders(ctx)
		w.retryFailedNotifications(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Reminder worker stopping")
			return
		case <-ticker.C:
			w.checkAndSendReminders(ctx)
		case <-retryTicker.C:
			w.retryFailedNotifications(ctx)
		}
	}
}

func (w *ReminderWorker) checkAndSendReminders(ctx context.Context) {
	w.logger.Info("Checking for tasks due soon and overdue")

	// Check tasks due in next 24 hours
	dueSoonTasks, err := w.taskRepo.GetDueSoonTasks(ctx, 24)
	if err != nil {
		w.logger.Error("Failed to get due soon tasks", "error", err)
	} else {
		w.logger.Info("Found tasks due soon", "count", len(dueSoonTasks))
		for _, task := range dueSoonTasks {
			if task.AssignedTo != nil {
				w.sendTaskNotification(ctx, task, domain.NotificationTypeDueSoon)
			}
		}
	}

	// Check overdue tasks
	overdueTasks, err := w.taskRepo.GetOverdueTasks(ctx)
	if err != nil {
		w.logger.Error("Failed to get overdue tasks", "error", err)
	} else {
		w.logger.Info("Found overdue tasks", "count", len(overdueTasks))
		for _, task := range overdueTasks {
			if task.AssignedTo != nil {
				w.sendTaskNotification(ctx, task, domain.NotificationTypeOverdue)
			}
		}
	}
}

func (w *ReminderWorker) sendTaskNotification(ctx context.Context, task *domain.Task, notificationType domain.NotificationType) {
	// Fetch user details
	user, err := w.userRepo.GetByID(ctx, *task.AssignedTo)
	if err != nil {
		w.logger.Error("Failed to get user details",
			"error", err,
			"user_id", task.AssignedTo,
			"task_id", task.ID,
		)
		return
	}

	// Double-check if notification was already sent (belt and suspenders with the query filter)
	alreadySent, err := w.notificationRepo.WasNotificationSent(ctx, task.ID, user.ID, notificationType, 24*time.Hour)
	if err != nil {
		w.logger.Error("Failed to check notification status",
			"error", err,
			"task_id", task.ID,
			"user_id", user.ID,
		)
		return
	}

	if alreadySent {
		w.logger.Debug("Notification already sent, skipping",
			"task_id", task.ID,
			"user_id", user.ID,
			"type", notificationType,
		)
		return
	}

	// Create notification record first (status: pending)
	notification := &domain.TaskNotification{
		TaskID:           task.ID,
		UserID:           user.ID,
		NotificationType: notificationType,
		Status:           domain.NotificationStatusPending,
	}

	if err := w.notificationRepo.Create(ctx, notification); err != nil {
		w.logger.Error("Failed to create notification record",
			"error", err,
			"task_id", task.ID,
			"user_id", user.ID,
		)
		return
	}

	// Queue the email job
	var emailType string
	switch notificationType {
	case domain.NotificationTypeDueSoon:
		emailType = "due_soon"
	case domain.NotificationTypeOverdue:
		emailType = "overdue"
	case domain.NotificationTypeTaskAssigned:
		emailType = "task_assigned"
	}

	w.emailWorker.QueueJob(EmailJob{
		Type:           emailType,
		TaskID:         task.ID,
		TaskTitle:      task.Title,
		OrgID:          task.OrgID,
		DueDate:        task.DueDate,
		RecipientEmail: user.Email,
		RecipientName:  user.Name,
		ActionURL:      fmt.Sprintf("https://yourapp.com/tasks/%s", task.ID),
	})

	// Mark notification as sent (in a real system, you'd update after actual send confirmation)
	if err := w.notificationRepo.MarkAsSent(ctx, notification.ID); err != nil {
		w.logger.Error("Failed to mark notification as sent",
			"error", err,
			"notification_id", notification.ID,
		)
	}

	w.logger.Info("Task notification queued",
		"task_id", task.ID,
		"user_id", user.ID,
		"type", notificationType,
	)
}

func (w *ReminderWorker) retryFailedNotifications(ctx context.Context) {
	failedNotifications, err := w.notificationRepo.GetPendingRetries(ctx, MaxRetries)
	if err != nil {
		w.logger.Error("Failed to get pending retries", "error", err)
		return
	}

	if len(failedNotifications) == 0 {
		return
	}

	w.logger.Info("Retrying failed notifications", "count", len(failedNotifications))

	for _, notification := range failedNotifications {
		// Fetch user and task details for retry
		user, err := w.userRepo.GetByID(ctx, notification.UserID)
		if err != nil {
			w.logger.Error("Failed to get user for retry",
				"error", err,
				"notification_id", notification.ID,
			)
			errMsg := fmt.Sprintf("failed to get user: %v", err)
			w.notificationRepo.MarkAsFailed(ctx, notification.ID, errMsg)
			continue
		}

		// Note: We don't have org context here, so we can't easily get the task
		// In a production system, you'd store more context in the notification
		// For now, we'll just retry the email with what we have

		var emailType string
		switch notification.NotificationType {
		case domain.NotificationTypeDueSoon:
			emailType = "due_soon"
		case domain.NotificationTypeOverdue:
			emailType = "overdue"
		case domain.NotificationTypeTaskAssigned:
			emailType = "task_assigned"
		}

		w.emailWorker.QueueJob(EmailJob{
			Type:           emailType,
			TaskID:         notification.TaskID,
			RecipientEmail: user.Email,
			RecipientName:  user.Name,
			ActionURL:      fmt.Sprintf("https://yourapp.com/tasks/%s", notification.TaskID),
		})

		// Mark as sent after queueing
		if err := w.notificationRepo.MarkAsSent(ctx, notification.ID); err != nil {
			w.logger.Error("Failed to mark retry as sent",
				"error", err,
				"notification_id", notification.ID,
			)
		} else {
			w.logger.Info("Notification retry queued",
				"notification_id", notification.ID,
				"retry_count", notification.RetryCount+1,
			)
		}
	}
}

