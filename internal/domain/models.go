package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	PasswordHash    string     `json:"-" db:"password_hash"`
	Name            string     `json:"name" db:"name"`
	EmailVerified   bool       `json:"email_verified" db:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Organization represents a multi-tenant organization
type Organization struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	OwnerID     uuid.UUID  `json:"owner_id" db:"owner_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Role types
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// OrgMember represents the membership relationship
type OrgMember struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	OrgID     uuid.UUID  `json:"org_id" db:"org_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Role      Role       `json:"role" db:"role"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Task status
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// Task represents a task within an organization
type Task struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	OrgID       uuid.UUID  `json:"org_id" db:"org_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      TaskStatus `json:"status" db:"status"`
	AssignedTo  *uuid.UUID `json:"assigned_to" db:"assigned_to"`
	DueDate     *time.Time `json:"due_date" db:"due_date"`
	CreatedBy   uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Request/Response DTOs
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}
type SignupResponse struct {
	UserID       uuid.UUID `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	OTPSent      bool      `json:"otp_sent"`
	OTPExpiresIn int       `json:"otp_expires_in"` // seconds
	Message      string    `json:"message"`
}

type VerifyOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type VerifyOTPResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

type ResendOTPRequest struct {
	Email string `json:"email"`
}

type ResendOTPResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	OTPExpiresIn  int    `json:"otp_expires_in"`           // seconds
	CooldownUntil *int64 `json:"cooldown_until,omitempty"` // unix timestamp
	RetryAfter    *int   `json:"retry_after,omitempty"`    // seconds
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type CreateOrgRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateOrgRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type AddMemberRequest struct {
	UserEmail string `json:"user_email"`
	Role      Role   `json:"role"`
}

type UpdateRoleRequest struct {
	Role Role `json:"role"`
}

type CreateTaskRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AssignedTo  *uuid.UUID `json:"assigned_to,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	Status      *TaskStatus `json:"status,omitempty"`
	DueDate     *time.Time  `json:"due_date,omitempty"`
}

type AssignTaskRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

type ListTasksQuery struct {
	Status     *TaskStatus `json:"status"`
	AssignedTo *uuid.UUID  `json:"assigned_to"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}

type ErrorResponse struct {
	Code    ErrorCode         `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type NotificationType string

const (
	NotificationTypeDueSoon      NotificationType = "due_soon"
	NotificationTypeOverdue      NotificationType = "overdue"
	NotificationTypeTaskAssigned NotificationType = "task_assigned"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

type TaskNotification struct {
	ID               uuid.UUID          `json:"id" db:"id"`
	TaskID           uuid.UUID          `json:"task_id" db:"task_id"`
	UserID           uuid.UUID          `json:"user_id" db:"user_id"`
	NotificationType NotificationType   `json:"notification_type" db:"notification_type"`
	SentAt           time.Time          `json:"sent_at" db:"sent_at"`
	Status           NotificationStatus `json:"status" db:"status"`
	RetryCount       int                `json:"retry_count" db:"retry_count"`
	LastError        *string            `json:"last_error,omitempty" db:"last_error"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
}

