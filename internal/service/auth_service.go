package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
}

type authService struct {
	repo      repository.Repository
	jwtSecret string
}

func NewAuthService(repo repository.Repository, jwtSecret string) AuthService {
	return &authService{repo: repo, jwtSecret: jwtSecret}
}

func (s *authService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	if req.Password == "" || req.Role == "" {
		return nil, domain.ErrInvalidInput
	}
	if req.Email == nil || *req.Email == "" {
		return nil, domain.ErrInvalidInput
	}
	if req.Role != domain.RoleLandlord && req.Role != domain.RoleTenant && req.Role != domain.RoleAdmin {
		return nil, domain.ErrInvalidInput
	}

	existing, _ := s.repo.GetUserByPhoneOrEmail(ctx, *req.Email)
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Role:         req.Role,
		Phone:        req.Phone,
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    now,
	}
	if req.Phone != nil && *req.Phone == "" {
		user.Phone = nil
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	if req.Role == domain.RoleTenant {
		tenant := &domain.TenantProfile{
			ID:        uuid.New(),
			UserID:    user.ID,
			CreatedAt: now,
		}
		if err := s.repo.CreateTenantProfile(ctx, tenant); err != nil {
			return nil, err
		}
	}

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	if req.Identifier == "" || req.Password == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.repo.GetUserByPhoneOrEmail(ctx, req.Identifier)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) generateToken(userID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
