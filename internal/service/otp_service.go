package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/aminshahid573/taskmanager/internal/cache"
	"github.com/aminshahid573/taskmanager/internal/domain"
)

const (
	OTPLength              = 6
	OTPExpiryMinutes       = 10
	MaxOTPAttempts         = 5
	InitialCooldownSeconds = 60   // 1 minute
	MaxCooldownSeconds     = 3600 // 1 hour
)

type OTPData struct {
	Code          string    `json:"code"`
	Email         string    `json:"email"`
	UserID        string    `json:"user_id"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Attempts      int       `json:"attempts"`
	IPAddress     string    `json:"ip_address"`
	Verified      bool      `json:"verified"`
	LastAttemptAt time.Time `json:"last_attempt_at"`
}

type CooldownData struct {
	FailedAttempts int       `json:"failed_attempts"`
	CooldownUntil  time.Time `json:"cooldown_until"`
	LastFailedAt   time.Time `json:"last_failed_at"`
}

type OTPService struct {
	redis *cache.RedisClient
}

func NewOTPService(redis *cache.RedisClient) *OTPService {
	return &OTPService{redis: redis}
}

// GenerateOTP creates a new OTP and stores it in Redis
func (s *OTPService) GenerateOTP(ctx context.Context, email, userID, ipAddress string) (*OTPData, error) {
	// // Check if user is in cooldown
	// cooldownKey := fmt.Sprintf("otp:cooldown:%s:%s", email, ipAddress)
	// var cooldown CooldownData
	// err := s.redis.Get(ctx, cooldownKey, &cooldown)
	// if err == nil {
	// 	// Cooldown exists
	// 	if time.Now().Before(cooldown.CooldownUntil) {
	// 		retryAfter := int(time.Until(cooldown.CooldownUntil).Seconds())
	// 		return nil, domain.NewAppError(
	// 			domain.ErrCodeOTPCooldown,
	// 			"Too many OTP requests. Please try again later.",
	// 			429,
	// 		).WithDetails(map[string]string{
	// 			"retry_after":    fmt.Sprintf("%d", retryAfter),
	// 			"cooldown_until": fmt.Sprintf("%d", cooldown.CooldownUntil.Unix()),
	// 		})
	// 	}
	// }

	generationKey := fmt.Sprintf("otp:generation:%s:%s", email, ipAddress)
	generationCountKey := fmt.Sprintf("otp:generation:count:%s:%s", email, ipAddress)

	exists, err := s.redis.Exists(ctx, generationKey)
	if err == nil && exists {
		// Get the TTL for the generation key to calculate retry_after
		ttl, _ := s.redis.TTL(ctx, generationKey)
		if ttl <= 0 {
			ttl = 60 // Default to 60 if TTL is 0
		}
		cooldownUntil := time.Now().Add(time.Duration(ttl) * time.Second).Unix()

		// Get remaining expiry time of existing OTP
		otpExpiryTime, _ := s.GetOTPExpiryTime(ctx, email, ipAddress)

		// Increment generation request count for exponential backoff
		_, _ = s.redis.Incr(ctx, generationCountKey)
		// Set expiry on count key (expires after 1 hour of last request)
		s.redis.Expire(ctx, generationCountKey, 1*time.Hour)

		return nil, domain.NewAppError(
			domain.ErrCodeOTPCooldown,
			"OTP was recently sent. Please wait before requesting another.",
			429,
		).WithDetails(map[string]string{
			"retry_after":    fmt.Sprintf("%d", ttl),
			"cooldown_until": fmt.Sprintf("%d", cooldownUntil),
			"otp_expires_in": fmt.Sprintf("%d", otpExpiryTime),
		})
	}

	// Check if user is in cooldown (from failed verification attempts)
	cooldownKey := fmt.Sprintf("otp:cooldown:%s:%s", email, ipAddress)
	var cooldown CooldownData
	err = s.redis.Get(ctx, cooldownKey, &cooldown)
	if err == nil {
		// Cooldown exists
		if time.Now().Before(cooldown.CooldownUntil) {
			retryAfter := int(time.Until(cooldown.CooldownUntil).Seconds())
			return nil, domain.NewAppError(
				domain.ErrCodeOTPCooldown,
				"Too many OTP requests. Please try again later.",
				429,
			).WithDetails(map[string]string{
				"retry_after":    fmt.Sprintf("%d", retryAfter),
				"cooldown_until": fmt.Sprintf("%d", cooldown.CooldownUntil.Unix()),
			})
		}
	}

	// Generate 6-digit OTP
	code, err := generateSecureOTP(OTPLength)
	if err != nil {
		return nil, domain.ErrInternal.WithError(err)
	}

	now := time.Now()
	otpData := &OTPData{
		Code:          code,
		Email:         email,
		UserID:        userID,
		CreatedAt:     now,
		ExpiresAt:     now.Add(OTPExpiryMinutes * time.Minute),
		Attempts:      0,
		IPAddress:     ipAddress,
		Verified:      false,
		LastAttemptAt: now,
	}

	// Store OTP in Redis with multiple keys for different lookups
	otpKey := fmt.Sprintf("otp:code:%s:%s", email, ipAddress)

	// Store with expiry
	if err := s.redis.Set(ctx, otpKey, otpData, OTPExpiryMinutes*time.Minute); err != nil {
		return nil, domain.NewAppError(domain.ErrCodeRedisError, "Failed to store OTP", 500).WithError(err)
	}

	// Calculate exponential backoff for next generation request
	// Formula: min(60 * 2^(attempts), 3600) where attempts is number of consecutive requests
	count, _ := s.redis.Incr(ctx, generationCountKey)
	s.redis.Expire(ctx, generationCountKey, 1*time.Hour)

	// Exponential backoff: 60 * 2^(count-1), capped at 1 hour
	cooldownSeconds := int64(InitialCooldownSeconds) * int64(math.Pow(2, float64(count-1)))
	if cooldownSeconds > MaxCooldownSeconds {
		cooldownSeconds = MaxCooldownSeconds
	}

	// Track generation time to prevent spam
	s.redis.Set(ctx, generationKey, now.Format(time.RFC3339), time.Duration(cooldownSeconds)*time.Second)

	return otpData, nil
}

// VerifyOTP validates the OTP code
func (s *OTPService) VerifyOTP(ctx context.Context, email, code, ipAddress string) (*OTPData, error) {
	otpKey := fmt.Sprintf("otp:code:%s:%s", email, ipAddress)

	var otpData OTPData
	err := s.redis.Get(ctx, otpKey, &otpData)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrCodeOTPNotFound,
			"OTP not found or expired. Please request a new one.",
			404,
		)
	}

	// Check if already verified
	if otpData.Verified {
		return nil, domain.NewAppError(
			domain.ErrCodeOTPAlreadyVerified,
			"This OTP has already been used.",
			400,
		)
	}

	// Check expiry
	if time.Now().After(otpData.ExpiresAt) {
		s.redis.Delete(ctx, otpKey)
		return nil, domain.NewAppError(
			domain.ErrCodeOTPExpired,
			"OTP has expired. Please request a new one.",
			400,
		)
	}

	// Increment attempts
	otpData.Attempts++
	otpData.LastAttemptAt = time.Now()

	// Check max attempts
	if otpData.Attempts > MaxOTPAttempts {
		s.redis.Delete(ctx, otpKey)

		// Apply exponential cooldown
		if err := s.applyCooldown(ctx, email, ipAddress, otpData.Attempts-MaxOTPAttempts); err != nil {
			return nil, err
		}

		return nil, domain.NewAppError(
			domain.ErrCodeOTPAttemptsExceeded,
			"Maximum OTP attempts exceeded. Please request a new OTP.",
			429,
		)
	}

	// Verify code (constant-time comparison to prevent timing attacks)
	if !secureCompare(otpData.Code, code) {
		// Update attempts in Redis
		s.redis.Set(ctx, otpKey, &otpData, time.Until(otpData.ExpiresAt))

		remainingAttempts := MaxOTPAttempts - otpData.Attempts
		if remainingAttempts == 0 {
			s.redis.Delete(ctx, otpKey)
			s.applyCooldown(ctx, email, ipAddress, 1)

			return nil, domain.NewAppError(
				domain.ErrCodeOTPAttemptsExceeded,
				"Maximum OTP attempts exceeded. Please request a new OTP.",
				429,
			)
		}

		return nil, domain.NewAppError(
			domain.ErrCodeOTPInvalid,
			"Invalid OTP code.",
			400,
		).WithDetails(map[string]string{
			"remaining_attempts": fmt.Sprintf("%d", remainingAttempts),
		})
	}

	// Mark as verified
	otpData.Verified = true
	s.redis.Set(ctx, otpKey, &otpData, 5*time.Minute) // Keep for 5 more minutes

	// Clear any cooldown
	cooldownKey := fmt.Sprintf("otp:cooldown:%s:%s", email, ipAddress)
	s.redis.Delete(ctx, cooldownKey)

	// Reset generation count on successful verification
	generationCountKey := fmt.Sprintf("otp:generation:count:%s:%s", email, ipAddress)
	s.redis.Delete(ctx, generationCountKey)

	return &otpData, nil
}

// applyCooldown applies exponential backoff cooldown
func (s *OTPService) applyCooldown(ctx context.Context, email, ipAddress string, failureCount int) error {
	cooldownKey := fmt.Sprintf("otp:cooldown:%s:%s", email, ipAddress)

	// Exponential backoff: 60s, 120s, 240s, 480s, 960s, capped at 1 hour
	cooldownSeconds := int(math.Min(
		float64(InitialCooldownSeconds)*math.Pow(2, float64(failureCount-1)),
		float64(MaxCooldownSeconds),
	))

	cooldown := CooldownData{
		FailedAttempts: failureCount,
		CooldownUntil:  time.Now().Add(time.Duration(cooldownSeconds) * time.Second),
		LastFailedAt:   time.Now(),
	}

	return s.redis.Set(ctx, cooldownKey, &cooldown, time.Duration(cooldownSeconds)*time.Second)
}

// CheckCooldown checks if user is in cooldown period
func (s *OTPService) CheckCooldown(ctx context.Context, email, ipAddress string) (*CooldownData, error) {
	cooldownKey := fmt.Sprintf("otp:cooldown:%s:%s", email, ipAddress)

	var cooldown CooldownData
	err := s.redis.Get(ctx, cooldownKey, &cooldown)
	if err != nil {
		return nil, nil // No cooldown active
	}

	if time.Now().After(cooldown.CooldownUntil) {
		s.redis.Delete(ctx, cooldownKey)
		return nil, nil
	}

	return &cooldown, nil
}

// InvalidateOTP removes an OTP (used after successful verification or on explicit request)
func (s *OTPService) InvalidateOTP(ctx context.Context, email, ipAddress string) error {
	otpKey := fmt.Sprintf("otp:code:%s:%s", email, ipAddress)
	return s.redis.Delete(ctx, otpKey)
}

// generateSecureOTP generates a cryptographically secure random OTP
func generateSecureOTP(length int) (string, error) {
	const digits = "0123456789"
	otp := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[num.Int64()]
	}

	return string(otp), nil
}

// secureCompare performs constant-time comparison to prevent timing attacks
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

// GetOTPExpiryTime returns the remaining expiry time for an existing OTP
func (s *OTPService) GetOTPExpiryTime(ctx context.Context, email, ipAddress string) (int, error) {
	otpKey := fmt.Sprintf("otp:code:%s:%s", email, ipAddress)

	var otpData OTPData
	err := s.redis.Get(ctx, otpKey, &otpData)
	if err != nil {
		return 0, err
	}

	// Check if OTP has expired
	if time.Now().After(otpData.ExpiresAt) {
		return 0, nil
	}

	// Return remaining seconds
	remaining := int(time.Until(otpData.ExpiresAt).Seconds())
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

