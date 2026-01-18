package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
	"github.com/aminshahid573/taskmanager/internal/service"
	"github.com/aminshahid573/taskmanager/internal/validator"
	"github.com/aminshahid573/taskmanager/internal/worker"
)

// TaskService defines the behavior TaskHandler needs from the task service.
type TaskService interface {
	Create(ctx context.Context, userID, orgID uuid.UUID, req domain.CreateTaskRequest) (*domain.Task, error)
	Get(ctx context.Context, userID, orgID, taskID uuid.UUID) (*domain.Task, error)
	List(ctx context.Context, userID, orgID uuid.UUID, query domain.ListTasksQuery) (*domain.PaginatedResponse, error)
	Update(ctx context.Context, userID, orgID, taskID uuid.UUID, req domain.UpdateTaskRequest) (*domain.Task, error)
	Delete(ctx context.Context, userID, orgID, taskID uuid.UUID) error
	Assign(ctx context.Context, userID, orgID, taskID, assigneeID uuid.UUID) error
}

type TaskHandler struct {
	taskService      TaskService
	userRepo         *repository.UserRepository
	orgRepo          *repository.OrgRepository
	notificationRepo *repository.NotificationRepository
	emailWorker      *worker.EmailWorker
	logger           *slog.Logger
}

func NewTaskHandler(taskService *service.TaskService, userRepo *repository.UserRepository, orgRepo *repository.OrgRepository, notificationRepo *repository.NotificationRepository, emailWorker *worker.EmailWorker, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		taskService:      taskService,
		userRepo:         userRepo,
		orgRepo:          orgRepo,
		notificationRepo: notificationRepo,
		emailWorker:      emailWorker,
		logger:           logger,
	}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("orgId"))

	fmt.Println("Creating task for user:", userID, "in organization:", orgID)

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateCreateTask(req); err != nil {
		respondError(w, err)
		return
	}

	task, err := h.taskService.Create(r.Context(), userID, orgID, req)
	if err != nil {
		h.logger.Error("Failed to create task", "error", err, "org_id", orgID)
		respondError(w, err)
		return
	}
	// Queue email if task is assigned
	if task.AssignedTo != nil {
		// Get the assigned user and org for email details
		assignedUser, err := h.userRepo.GetByID(r.Context(), *task.AssignedTo)
		org, orgErr := h.orgRepo.GetByID(r.Context(), task.OrgID)

		if err == nil && assignedUser != nil {
			orgName := ""
			if orgErr == nil && org != nil {
				orgName = org.Name
			}

			// Create notification record for tracking
			notification := &domain.TaskNotification{
				TaskID:           task.ID,
				UserID:           assignedUser.ID,
				NotificationType: domain.NotificationTypeTaskAssigned,
				Status:           domain.NotificationStatusPending,
			}
			if err := h.notificationRepo.Create(r.Context(), notification); err != nil {
				h.logger.Error("Failed to create notification record", "error", err, "task_id", task.ID)
			}

			h.emailWorker.QueueJob(worker.EmailJob{
				Type:           "task_assigned",
				TaskID:         task.ID,
				RecipientEmail: assignedUser.Email,
				RecipientName:  assignedUser.Name,
				TaskTitle:      task.Title,
				OrgID:          task.OrgID,
				OrgName:        orgName,
				DueDate:        task.DueDate,
				ActionURL:      fmt.Sprintf("http://localhost:3000/organizations/%s/tasks/%s", task.OrgID, task.ID),
				ExtraNote:      task.Description,
			})

			// Mark as sent after queueing
			if notification.ID != uuid.Nil {
				if err := h.notificationRepo.MarkAsSent(r.Context(), notification.ID); err != nil {
					h.logger.Error("Failed to mark notification as sent", "error", err, "notification_id", notification.ID)
				}
			}
		}
	}

	h.logger.Info("Task created", "task_id", task.ID, "org_id", orgID)
	respondJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("orgId"))
	taskID := mustParseUUID(r.PathValue("id"))

	task, err := h.taskService.Get(r.Context(), userID, orgID, taskID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("orgId"))

	// Parse query parameters
	query := domain.ListTasksQuery{
		Page:  1,
		Limit: 20,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			query.Page = p
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			query.Limit = l
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		taskStatus := domain.TaskStatus(status)
		if err := validator.ValidateTaskStatus(taskStatus); err == nil {
			query.Status = &taskStatus
		}
	}

	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		if id, err := uuid.Parse(assignedTo); err == nil {
			query.AssignedTo = &id
		}
	}

	result, err := h.taskService.List(r.Context(), userID, orgID, query)
	if err != nil {
		h.logger.Error("Failed to list tasks", "error", err, "org_id", orgID)
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("orgId"))
	taskID := mustParseUUID(r.PathValue("id"))

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if req.Status != nil {
		if err := validator.ValidateTaskStatus(*req.Status); err != nil {
			respondError(w, err)
			return
		}
	}

	task, err := h.taskService.Update(r.Context(), userID, orgID, taskID, req)
	if err != nil {
		h.logger.Error("Failed to update task", "error", err, "task_id", taskID)
		respondError(w, err)
		return
	}

	h.logger.Info("Task updated", "task_id", task.ID, "org_id", orgID)
	respondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("orgId"))
	taskID := mustParseUUID(r.PathValue("id"))

	if err := h.taskService.Delete(r.Context(), userID, orgID, taskID); err != nil {
		h.logger.Error("Failed to delete task", "error", err, "task_id", taskID)
		respondError(w, err)
		return
	}

	h.logger.Info("Task deleted", "task_id", taskID, "org_id", orgID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	userID := mustParseUUID(r.Context().Value("user_id").(string))
	orgID := mustParseUUID(r.PathValue("orgId"))
	taskID := mustParseUUID(r.PathValue("id"))

	var req domain.AssignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := h.taskService.Assign(r.Context(), userID, orgID, taskID, req.UserID); err != nil {
		h.logger.Error("Failed to assign task", "error", err, "task_id", taskID)
		respondError(w, err)
		return
	}

	// Fetch task, user and org details for email notification
	task, taskErr := h.taskService.Get(r.Context(), userID, orgID, taskID)
	assignedUser, userErr := h.userRepo.GetByID(r.Context(), req.UserID)
	org, orgErr := h.orgRepo.GetByID(r.Context(), orgID)

	// Queue email notification only if we have the required details
	if taskErr == nil && userErr == nil && assignedUser != nil && task != nil {
		orgName := ""
		if orgErr == nil && org != nil {
			orgName = org.Name
		}

		// Create notification record for tracking
		notification := &domain.TaskNotification{
			TaskID:           taskID,
			UserID:           req.UserID,
			NotificationType: domain.NotificationTypeTaskAssigned,
			Status:           domain.NotificationStatusPending,
		}
		if err := h.notificationRepo.Create(r.Context(), notification); err != nil {
			h.logger.Error("Failed to create notification record", "error", err, "task_id", taskID)
		}

		h.logger.Debug("Queuing assignment email",
			"task_id", taskID,
			"recipient", assignedUser.Email,
			"user_id", req.UserID,
		)
		h.emailWorker.QueueJob(worker.EmailJob{
			Type:           "task_assigned",
			TaskID:         taskID,
			OrgID:          orgID,
			RecipientEmail: assignedUser.Email,
			RecipientName:  assignedUser.Name,
			TaskTitle:      task.Title,
			OrgName:        orgName,
			DueDate:        task.DueDate,
			ActionURL:      fmt.Sprintf("http://localhost:3000/organizations/%s/tasks/%s", orgID, taskID),
			ExtraNote:      task.Description,
		})

		// Mark as sent after queueing
		if notification.ID != uuid.Nil {
			if err := h.notificationRepo.MarkAsSent(r.Context(), notification.ID); err != nil {
				h.logger.Error("Failed to mark notification as sent", "error", err, "notification_id", notification.ID)
			}
		}
	} else {
		h.logger.Warn("Could not queue assignment email - failed to fetch details",
			"task_err", taskErr,
			"user_err", userErr,
			"assigned_user_exists", assignedUser != nil,
			"task_exists", task != nil,
			"task_id", taskID,
		)
	}

	h.logger.Info("Task assigned", "task_id", taskID, "assignee_id", req.UserID)
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Task assigned successfully",
	})
}

