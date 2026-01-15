package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/aminshahid573/taskmanager/internal/domain"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func ValidateSignup(req domain.SignupRequest) error {
	if err := ValidateEmail(req.Email); err != nil {
		return err
	}
	if err := ValidatePassword(req.Password); err != nil {
		return err
	}
	if err := ValidateRequired("name", req.Name); err != nil {
		return err
	}
	if len(req.Name) < 2 || len(req.Name) > 100 {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"name": "must be between 2 and 100 characters",
		})
	}
	return nil
}
func ValidateLogin(req domain.LoginRequest) error {
	if err := ValidateEmail(req.Email); err != nil {
		return err
	}
	if err := ValidateRequired("password", req.Password); err != nil {
		return err
	}
	return nil
}
func ValidateEmail(email string) error {
	if email == "" {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"email": "is required",
		})
	}
	email = strings.TrimSpace(email)
	if _, err := mail.ParseAddress(email); err != nil {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"email": "invalid format",
		})
	}
	if !emailRegex.MatchString(email) {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"email": "invalid format",
		})
	}
	return nil
}
func ValidatePassword(password string) error {
	if password == "" {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"password": "is required",
		})
	}
	if len(password) < 8 {
		return domain.ErrValidationFailed.WithDetails(map[string]string{

			"password": "must be at least 8 characters",
		})
	}
	var hasUpper, hasLower, hasNumber bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}
	if !hasUpper || !hasLower || !hasNumber {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"password": "must contain uppercase, lowercase, and number",
		})
	}
	return nil
}
func ValidateRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			field: "is required",
		})
	}
	return nil
}
func ValidateCreateOrg(req domain.CreateOrgRequest) error {
	if err := ValidateRequired("name", req.Name); err != nil {
		return err
	}
	if len(req.Name) < 2 || len(req.Name) > 100 {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"name": "must be between 2 and 100 characters",
		})
	}
	return nil
}
func ValidateCreateTask(req domain.CreateTaskRequest) error {
	if err := ValidateRequired("title", req.Title); err != nil {
		return err
	}
	if len(req.Title) < 3 || len(req.Title) > 200 {
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"title": "must be between 3 and 200 characters",
		})
	}
	return nil
}
func ValidateTaskStatus(status domain.TaskStatus) error {
	switch status {

	case domain.TaskStatusTodo, domain.TaskStatusInProgress, domain.TaskStatusDone:
		return nil
	default:
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"status": fmt.Sprintf("must be one of: %s, %s, %s",
				domain.TaskStatusTodo, domain.TaskStatusInProgress, domain.TaskStatusDone),
		})
	}
}
func ValidateRole(role domain.Role) error {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleMember:
		return nil
	default:
		return domain.ErrValidationFailed.WithDetails(map[string]string{
			"role": fmt.Sprintf("must be one of: %s, %s, %s",
				domain.RoleOwner, domain.RoleAdmin, domain.RoleMember),
		})
	}
}
