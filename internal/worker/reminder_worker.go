package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aminshahid573/taskmanager/internal/repository"
)

type ReminderWorker struct {
	taskRepo    *repository.TaskRepository
	userRepo    *repository.UserRepository
	emailWorker *EmailWorker
	logger      *slog.Logger
}

func NewReminderWorker(taskRepo *repository.TaskRepository, userRepo *repository.UserRepository, emailWorker *EmailWorker, logger *slog.Logger) *ReminderWorker {
	return &ReminderWorker{
		taskRepo:    taskRepo,
		userRepo:    userRepo,
		emailWorker: emailWorker,
		logger:      logger,
	}
}

func (w *ReminderWorker) Start(ctx context.Context) {
	w.logger.Info("Reminder worker started")

	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	defer ticker.Stop()

	// Run immediately on start
	w.checkAndSendReminders(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Reminder worker stopping")
			return
		case <-ticker.C:
			w.checkAndSendReminders(ctx)
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
				// Fetch user details to get actual email and name
				user, err := w.userRepo.GetByID(ctx, *task.AssignedTo)
				if err != nil {
					w.logger.Error("Failed to get user details", "error", err, "user_id", task.AssignedTo)
					continue
				}
				w.emailWorker.QueueJob(EmailJob{
					Type:           "due_soon",
					TaskID:         task.ID,
					TaskTitle:      task.Title,
					OrgID:          task.OrgID,
					DueDate:        task.DueDate,
					RecipientEmail: user.Email,
					RecipientName:  user.Name,
					ActionURL:      fmt.Sprintf("https://yourapp.com/tasks/%s", task.ID),
				})
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
				// Fetch user details to get actual email and name
				user, err := w.userRepo.GetByID(ctx, *task.AssignedTo)
				if err != nil {
					w.logger.Error("Failed to get user details", "error", err, "user_id", task.AssignedTo)
					continue
				}
				w.emailWorker.QueueJob(EmailJob{
					Type:           "overdue",
					TaskID:         task.ID,
					TaskTitle:      task.Title,
					OrgID:          task.OrgID,
					DueDate:        task.DueDate,
					RecipientEmail: user.Email,
					RecipientName:  user.Name,
					ActionURL:      fmt.Sprintf("https://yourapp.com/tasks/%s", task.ID),
				})
			}
		}
	}
}
