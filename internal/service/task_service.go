package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
)

// TaskRepository defines the behavior TaskService needs from the task repository.
type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, taskID, orgID uuid.UUID) (*domain.Task, error)
	List(ctx context.Context, orgID uuid.UUID, query domain.ListTasksQuery) ([]*domain.Task, int, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, taskID, orgID uuid.UUID) error
	Assign(ctx context.Context, taskID, orgID, assigneeID uuid.UUID) error
}

type TaskService struct {
	taskRepo TaskRepository
	orgRepo  OrgRepository
}

func NewTaskService(taskRepo *repository.TaskRepository, orgRepo *repository.OrgRepository) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
		orgRepo:  orgRepo,
	}
}

func (s *TaskService) Create(ctx context.Context, userID, orgID uuid.UUID, req domain.CreateTaskRequest) (*domain.Task, error) {
	// Check membership
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	//TODO:: remove
	fmt.Println("isMember:", isMember)
	//prints all incoming tasks
	fmt.Println("request:", req)

	if !isMember {
		return nil, domain.ErrNotMember
	}

	// If assigned to someone, check they are a member
	if req.AssignedTo != nil {
		isMember, err := s.orgRepo.IsMember(ctx, orgID, *req.AssignedTo)
		if err != nil {
			return nil, err
		}
		if !isMember {
			return nil, domain.ErrNotMember.WithDetails(map[string]string{
				"assigned_to": "user is not a member of this organization",
			})
		}
	}

	task := &domain.Task{
		OrgID:       orgID,
		Title:       req.Title,
		Description: req.Description,
		AssignedTo:  req.AssignedTo,
		DueDate:     req.DueDate,
		CreatedBy:   userID,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) Get(ctx context.Context, userID, orgID, taskID uuid.UUID) (*domain.Task, error) {
	// Check membership
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	return s.taskRepo.GetByID(ctx, taskID, orgID)
}

func (s *TaskService) List(ctx context.Context, userID, orgID uuid.UUID, query domain.ListTasksQuery) (*domain.PaginatedResponse, error) {
	// Check membership
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	tasks, total, err := s.taskRepo.List(ctx, orgID, query)
	if err != nil {
		return nil, err
	}

	totalPages := total / query.Limit
	if total%query.Limit > 0 {
		totalPages++
	}

	return &domain.PaginatedResponse{
		Data:       tasks,
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *TaskService) Update(ctx context.Context, userID, orgID, taskID uuid.UUID, req domain.UpdateTaskRequest) (*domain.Task, error) {
	// Check membership
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrNotMember
	}

	task, err := s.taskRepo.GetByID(ctx, taskID, orgID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, userID, orgID, taskID uuid.UUID) error {
	// Check membership
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return domain.ErrNotMember
	}

	return s.taskRepo.Delete(ctx, taskID, orgID)
}

func (s *TaskService) Assign(ctx context.Context, userID, orgID, taskID, assigneeID uuid.UUID) error {
	// Check membership of current user
	isMember, err := s.orgRepo.IsMember(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return domain.ErrNotMember
	}

	// Check membership of assignee
	isMember, err = s.orgRepo.IsMember(ctx, orgID, assigneeID)
	if err != nil {
		return err
	}
	if !isMember {
		return domain.ErrNotMember.WithDetails(map[string]string{
			"assignee": "user is not a member of this organization",
		})
	}

	return s.taskRepo.Assign(ctx, taskID, orgID, assigneeID)
}

