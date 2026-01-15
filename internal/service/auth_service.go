package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aminshahid573/taskmanager/internal/config"
	"github.com/aminshahid573/taskmanager/internal/domain"
	"github.com/aminshahid573/taskmanager/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository defines the behavior AuthService needs from a user repository.
type UserRepository interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// TokenStore defines the minimal operations AuthService needs for token storage.
type TokenStore interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type AuthService struct {
	userRepo UserRepository
	redis    TokenStore
	jwtCfg   config.JWTConfig
}

func NewAuthService(userRepo *repository.UserRepository, redis TokenStore, jwtCfg config.JWTConfig) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		redis:    redis,
		jwtCfg:   jwtCfg,
	}
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthService) Signup(ctx context.Context, req domain.SignupRequest) (*domain.User, error) {
	// Check if email exists
	exists, err := s.userRepo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrAlreadyExists.WithDetails(map[string]string{
			"email": "already registered",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.ErrInternal.WithError(err)
	}

	// Create user
	user := &domain.User{
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		Name:          req.Name,
		EmailVerified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Check if email is verified
	if !user.EmailVerified {
		return nil, domain.NewAppError(
			domain.ErrCodeEmailNotVerified,
			"Email not verified. Please verify your email first.",
			403,
		).WithDetails(map[string]string{
			"email":  user.Email,
			"action": "verify_email",
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Store refresh token in Redis
	key := fmt.Sprintf("refresh_token:%s", user.ID)
	if err := s.redis.Set(ctx, key, refreshToken, time.Duration(s.jwtCfg.RefreshTokenDuration)*time.Minute); err != nil {
		return nil, domain.NewAppError(domain.ErrCodeRedisError, "Failed to store token", 500).WithError(err)
	}

	return &domain.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtCfg.AccessTokenDuration * 60,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenResponse, error) {
	// Parse and validate refresh token
	claims, err := s.validateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Check if token exists in Redis
	key := fmt.Sprintf("refresh_token:%s", claims.UserID)
	var storedToken string
	if err := s.redis.Get(ctx, key, &storedToken); err != nil {
		return nil, domain.ErrInvalidToken
	}

	if storedToken != refreshToken {
		return nil, domain.ErrInvalidToken
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new tokens
	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Update refresh token in Redis
	if err := s.redis.Set(ctx, key, newRefreshToken, time.Duration(s.jwtCfg.RefreshTokenDuration)*time.Minute); err != nil {
		return nil, domain.NewAppError(domain.ErrCodeRedisError, "Failed to store token", 500).WithError(err)
	}

	return &domain.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    s.jwtCfg.AccessTokenDuration * 60,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, accessToken string) error {
	// Delete refresh token
	key := fmt.Sprintf("refresh_token:%s", userID)
	if err := s.redis.Delete(ctx, key); err != nil {
		return domain.NewAppError(domain.ErrCodeRedisError, "Failed to logout", 500).WithError(err)
	}

	// Blacklist access token
	blacklistKey := fmt.Sprintf("blacklist:%s", accessToken)
	if err := s.redis.Set(ctx, blacklistKey, "1", time.Duration(s.jwtCfg.AccessTokenDuration)*time.Minute); err != nil {
		return domain.NewAppError(domain.ErrCodeRedisError, "Failed to blacklist token", 500).WithError(err)
	}

	return nil
}

func (s *AuthService) ValidateAccessToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Check if token is blacklisted
	blacklistKey := fmt.Sprintf("blacklist:%s", tokenString)
	exists, err := s.redis.Exists(ctx, blacklistKey)
	if err != nil {
		return nil, domain.NewAppError(domain.ErrCodeRedisError, "Failed to check token", 500).WithError(err)
	}
	if exists {
		return nil, domain.ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return []byte(s.jwtCfg.AccessSecret), nil
	})

	if err != nil {
		return nil, domain.ErrInvalidToken.WithError(err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().Unix() > claims.ExpiresAt.Unix() {
		return nil, domain.ErrExpiredToken
	}

	return claims, nil
}

func (s *AuthService) generateAccessToken(user *domain.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.jwtCfg.AccessTokenDuration) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.AccessSecret))
}

func (s *AuthService) generateRefreshToken(user *domain.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.jwtCfg.RefreshTokenDuration) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.RefreshSecret))
}

func (s *AuthService) validateRefreshToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return []byte(s.jwtCfg.RefreshSecret), nil
	})

	if err != nil {
		return nil, domain.ErrInvalidToken.WithError(err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().Unix() > claims.ExpiresAt.Unix() {
		return nil, domain.ErrExpiredToken
	}

	return claims, nil
}

// GenerateTokensAfterVerification generates tokens after OTP verification
// This bypasses password check since user has already verified via OTP
func (s *AuthService) GenerateTokensAfterVerification(ctx context.Context, user *domain.User) (*domain.TokenResponse, error) {
	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Store refresh token in Redis
	key := fmt.Sprintf("refresh_token:%s", user.ID)
	if err := s.redis.Set(ctx, key, refreshToken, time.Duration(s.jwtCfg.RefreshTokenDuration)*time.Minute); err != nil {
		return nil, domain.NewAppError(domain.ErrCodeRedisError, "Failed to store token", 500).WithError(err)
	}

	return &domain.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtCfg.AccessTokenDuration * 60,
	}, nil
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
