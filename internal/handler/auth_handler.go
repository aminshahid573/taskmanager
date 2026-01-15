package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
	"github.com/aminshahid573/taskmanager/internal/service"
	"github.com/aminshahid573/taskmanager/internal/validator"
	"github.com/aminshahid573/taskmanager/internal/worker"
	"github.com/google/uuid"
)

// AuthService defines the behavior AuthHandler needs from the authentication service.
type AuthService interface {
	Signup(ctx context.Context, req domain.SignupRequest) (*domain.User, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenResponse, error)
	GenerateTokensAfterVerification(ctx context.Context, user *domain.User) (*domain.TokenResponse, error)
	Logout(ctx context.Context, userID uuid.UUID, accessToken string) error
}

type AuthHandler struct {
	authService AuthService
	otpService  *service.OTPService
	userRepo    *repository.UserRepository
	emailWorker *worker.EmailWorker
	logger      *slog.Logger
}

func NewAuthHandler(
	authService *service.AuthService,
	otpService *service.OTPService,
	userRepo *repository.UserRepository,
	emailWorker *worker.EmailWorker,
	logger *slog.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		otpService:  otpService,
		userRepo:    userRepo,
		emailWorker: emailWorker,
		logger:      logger,
	}
}
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req domain.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode signup request", "error", err)
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateSignup(req); err != nil {
		respondError(w, err)
		return
	}

	user, err := h.authService.Signup(r.Context(), req)
	if err != nil {
		h.logger.Error("Signup failed", "error", err, "email", req.Email)
		respondError(w, err)
		return
	}

	// Generate OTP
	ipAddress := getClientIP(r)
	otpData, err := h.otpService.GenerateOTP(r.Context(), user.Email, user.ID.String(), ipAddress)
	if err != nil {
		h.logger.Error("Failed to generate OTP", "error", err, "user_id", user.ID)
		respondError(w, err)
		return
	}

	// Queue OTP email
	h.emailWorker.QueueJob(worker.EmailJob{
		Type:           "otp_verification",
		RecipientEmail: user.Email,
		RecipientName:  user.Name,
		OTPCode:        otpData.Code,
	})

	h.logger.Info("User signed up successfully, OTP sent", "user_id", user.ID, "email", user.Email)

	response := domain.SignupResponse{
		UserID:       user.ID,
		Email:        user.Email,
		Name:         user.Name,
		OTPSent:      true,
		OTPExpiresIn: int(time.Until(otpData.ExpiresAt).Seconds()),
		Message:      "Account created successfully. Please verify your email with the OTP sent to your inbox.",
	}

	respondJSON(w, http.StatusCreated, response)
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req domain.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateEmail(req.Email); err != nil {
		respondError(w, err)
		return
	}

	if req.OTP == "" || len(req.OTP) != 6 {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"otp": "must be 6 digits",
		}))
		return
	}

	ipAddress := getClientIP(r)
	_, err := h.otpService.VerifyOTP(r.Context(), req.Email, req.OTP, ipAddress)
	if err != nil {
		h.logger.Warn("OTP verification failed", "error", err, "email", req.Email, "ip", ipAddress)
		respondError(w, err)
		return
	}

	// Get user
	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		h.logger.Error("User not found after OTP verification", "error", err, "email", req.Email)
		respondError(w, err)
		return
	}

	// Mark email as verified
	if err := h.userRepo.VerifyEmail(r.Context(), user.ID); err != nil {
		h.logger.Error("Failed to verify email", "error", err, "user_id", user.ID)
		respondError(w, err)
		return
	}

	// Generate tokens
	tokens, err := h.authService.GenerateTokensAfterVerification(r.Context(), user)

	if err != nil {
		h.logger.Error("Failed to generate tokens", "error", err, "user_id", user.ID)
		respondError(w, err)
		return
	}
	// Invalidate OTP
	h.otpService.InvalidateOTP(r.Context(), req.Email, ipAddress)

	h.logger.Info("Email verified successfully", "user_id", user.ID, "email", user.Email)

	response := domain.VerifyOTPResponse{
		Success:      true,
		Message:      "Email verified successfully",
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn, // 15 minutes
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req domain.ResendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateEmail(req.Email); err != nil {
		respondError(w, err)
		return
	}

	// Get user
	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		// Don't reveal if user exists or not
		respondJSON(w, http.StatusOK, domain.ResendOTPResponse{
			Success:      true,
			Message:      "If the email exists, an OTP has been sent.",
			OTPExpiresIn: 600,
		})
		return
	}

	// Check if already verified
	if user.EmailVerified {
		respondError(w, domain.NewAppError(
			domain.ErrCodeOTPAlreadyVerified,
			"Email is already verified.",
			400,
		))
		return
	}

	ipAddress := getClientIP(r)

	// Check cooldown
	cooldown, err := h.otpService.CheckCooldown(r.Context(), req.Email, ipAddress)
	if err == nil && cooldown != nil {
		retryAfter := int(time.Until(cooldown.CooldownUntil).Seconds())
		cooldownUntil := cooldown.CooldownUntil.Unix()

		respondJSON(w, http.StatusTooManyRequests, domain.ResendOTPResponse{
			Success:       false,
			Message:       "Too many requests. Please try again later.",
			OTPExpiresIn:  0,
			CooldownUntil: &cooldownUntil,
			RetryAfter:    &retryAfter,
		})
		return
	}

	// Generate new OTP
	otpData, err := h.otpService.GenerateOTP(r.Context(), user.Email, user.ID.String(), ipAddress)
	if err != nil {
		appErr, ok := err.(*domain.AppError)
		if ok && appErr.Code == domain.ErrCodeOTPCooldown {
			// Extract cooldown info from error details
			var retryAfter int
			var cooldownUntil int64
			var otpExpiresIn int
			if appErr.Details != nil {
				fmt.Sscanf(appErr.Details["retry_after"], "%d", &retryAfter)
				fmt.Sscanf(appErr.Details["cooldown_until"], "%d", &cooldownUntil)
				fmt.Sscanf(appErr.Details["otp_expires_in"], "%d", &otpExpiresIn)
			}

			respondJSON(w, http.StatusTooManyRequests, domain.ResendOTPResponse{
				Success:       false,
				Message:       appErr.Message,
				OTPExpiresIn:  otpExpiresIn,
				CooldownUntil: &cooldownUntil,
				RetryAfter:    &retryAfter,
			})
			return
		}

		h.logger.Error("Failed to generate OTP", "error", err, "email", req.Email)
		respondError(w, err)
		return
	}

	// Queue OTP email
	h.emailWorker.QueueJob(worker.EmailJob{
		Type:           "otp_verification",
		RecipientEmail: user.Email,
		RecipientName:  user.Name,
		OTPCode:        otpData.Code,
	})

	h.logger.Info("OTP resent", "email", req.Email, "ip", ipAddress)

	respondJSON(w, http.StatusOK, domain.ResendOTPResponse{
		Success:      true,
		Message:      "OTP sent successfully",
		OTPExpiresIn: int(time.Until(otpData.ExpiresAt).Seconds()),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode login request", "error", err)
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if err := validator.ValidateLogin(req); err != nil {
		respondError(w, err)
		return
	}

	tokens, err := h.authService.Login(r.Context(), req)
	if err != nil {
		h.logger.Warn("Login failed", "error", err, "email", req.Email)
		respondError(w, err)
		return
	}

	h.logger.Info("User logged in successfully", "email", req.Email)
	respondJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"body": "invalid JSON format",
		}))
		return
	}

	if req.RefreshToken == "" {
		respondError(w, domain.ErrValidationFailed.WithDetails(map[string]string{
			"refresh_token": "is required",
		}))
		return
	}

	tokens, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Warn("Token refresh failed", "error", err)
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, domain.ErrUnauthorized)
		return
	}
	// Get token from header
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if err := h.authService.Logout(r.Context(), mustParseUUID(userID), token); err != nil {
		h.logger.Error("Logout failed", "error", err, "user_id", userID)
		respondError(w, err)
		return
	}

	h.logger.Info("User logged out successfully", "user_id", userID)
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}
