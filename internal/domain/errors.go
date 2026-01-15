package domain

import (
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	// Authentication & Authorization
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrCodeExpiredToken       ErrorCode = "EXPIRED_TOKEN"
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"

	// Validation
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField     ErrorCode = "MISSING_FIELD"

	// OTP Related
	ErrCodeOTPExpired          ErrorCode = "OTP_EXPIRED"
	ErrCodeOTPInvalid          ErrorCode = "OTP_INVALID"
	ErrCodeOTPAttemptsExceeded ErrorCode = "OTP_ATTEMPTS_EXCEEDED"
	ErrCodeOTPCooldown         ErrorCode = "OTP_COOLDOWN"
	ErrCodeOTPNotFound         ErrorCode = "OTP_NOT_FOUND"
	ErrCodeEmailNotVerified    ErrorCode = "EMAIL_NOT_VERIFIED"
	ErrCodeOTPAlreadyVerified  ErrorCode = "OTP_ALREADY_VERIFIED"

	// Resource
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict      ErrorCode = "CONFLICT"

	// Business Logic
	ErrCodeInsufficientPermissions ErrorCode = "INSUFFICIENT_PERMISSIONS"
	ErrCodeNotMember               ErrorCode = "NOT_MEMBER"
	ErrCodeCannotDeleteOwner       ErrorCode = "CANNOT_DELETE_OWNER"
	ErrCodeOrgNotFound             ErrorCode = "ORG_NOT_FOUND"
	ErrCodeTaskNotFound            ErrorCode = "TASK_NOT_FOUND"
	ErrCodeUserNotFound            ErrorCode = "USER_NOT_FOUND"

	// External Services
	ErrCodeDatabaseError     ErrorCode = "DATABASE_ERROR"
	ErrCodeRedisError        ErrorCode = "REDIS_ERROR"
	ErrCodeEmailServiceError ErrorCode = "EMAIL_SERVICE_ERROR"
	ErrCodeExternalAPIError  ErrorCode = "EXTERNAL_API_ERROR"

	// System
	ErrCodeInternal           ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeTimeout            ErrorCode = "TIMEOUT"
)

type AppError struct {
	Code       ErrorCode         `json:"code"`
	Message    string            `json:"message"`
	StatusCode int               `json:"-"`
	Details    map[string]string `json:"details,omitempty"`
	Err        error             `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

func (e *AppError) WithDetails(details map[string]string) *AppError {
	e.Details = details
	return e
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// Common errors
var (
	ErrUnauthorized = NewAppError(
		ErrCodeUnauthorized,
		"Authentication required",
		http.StatusUnauthorized,
	)

	ErrForbidden = NewAppError(
		ErrCodeForbidden,
		"Access denied",
		http.StatusForbidden,
	)

	ErrInvalidToken = NewAppError(
		ErrCodeInvalidToken,
		"Invalid or malformed token",
		http.StatusUnauthorized,
	)

	ErrExpiredToken = NewAppError(
		ErrCodeExpiredToken,
		"Token has expired",
		http.StatusUnauthorized,
	)

	ErrInvalidCredentials = NewAppError(
		ErrCodeInvalidCredentials,
		"Invalid email or password",
		http.StatusUnauthorized,
	)

	ErrValidationFailed = NewAppError(
		ErrCodeValidationFailed,
		"Validation failed",
		http.StatusBadRequest,
	)

	ErrNotFound = NewAppError(
		ErrCodeNotFound,
		"Resource not found",
		http.StatusNotFound,
	)

	ErrAlreadyExists = NewAppError(
		ErrCodeAlreadyExists,
		"Resource already exists",
		http.StatusConflict,
	)

	ErrInsufficientPermissions = NewAppError(
		ErrCodeInsufficientPermissions,
		"Insufficient permissions to perform this action",
		http.StatusForbidden,
	)

	ErrNotMember = NewAppError(
		ErrCodeNotMember,
		"User is not a member of this organization",
		http.StatusForbidden,
	)

	ErrCannotDeleteOwner = NewAppError(
		ErrCodeCannotDeleteOwner,
		"Cannot remove the organization owner",
		http.StatusBadRequest,
	)

	ErrDatabaseError = NewAppError(
		ErrCodeDatabaseError,
		"Database operation failed",
		http.StatusInternalServerError,
	)

	ErrInternal = NewAppError(
		ErrCodeInternal,
		"Internal server error",
		http.StatusInternalServerError,
	)

	ErrRateLimitExceeded = NewAppError(
		ErrCodeRateLimitExceeded,
		"Rate limit exceeded",
		http.StatusTooManyRequests,
	)
)
