package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/config"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/mail"
	"github.com/kenya-houses/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req domain.RegisterRequest) (*domain.RegisterResult, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
	VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (*domain.AuthResponse, error)
	ResendOtp(ctx context.Context, email string) error
}

type authService struct {
	repo        repository.Repository
	jwtSecret   string
	mailer      mail.Sender
	otpExpiry   time.Duration
	otpCooldown time.Duration
	isDev       bool
}

func NewAuthService(repo repository.Repository, jwtSecret string, mailer mail.Sender, cfg config.Config) AuthService {
	return &authService{
		repo:        repo,
		jwtSecret:   jwtSecret,
		mailer:      mailer,
		otpExpiry:   time.Duration(cfg.OTPExpiryMinutes) * time.Minute,
		otpCooldown: time.Duration(cfg.OTPCooldownSeconds) * time.Second,
		isDev:       strings.EqualFold(cfg.Environment, "development"),
	}
}

func (s *authService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.RegisterResult, error) {
	if req.Password == "" || req.Role == "" {
		return nil, domain.ErrInvalidInput
	}
	if req.Email == nil || *req.Email == "" {
		return nil, domain.ErrInvalidInput
	}
	email := strings.ToLower(strings.TrimSpace(*req.Email))
	if req.Role != domain.RoleLandlord && req.Role != domain.RoleTenant && req.Role != domain.RoleAdmin {
		return nil, domain.ErrInvalidInput
	}

	existing, _ := s.repo.GetUserByPhoneOrEmail(ctx, email)
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
		Email:        &email,
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
			FullName:  req.FullName,
			CreatedAt: now,
		}
		if err := s.repo.CreateTenantProfile(ctx, tenant); err != nil {
			return nil, err
		}
	}

	if req.Role == domain.RoleLandlord {
		var pageName *string
		if req.PageName != "" {
			p := req.PageName
			pageName = &p
		}
		profile := &domain.LandlordProfile{
			ID:                     uuid.New(),
			UserID:                 user.ID,
			FullName:               req.FullName,
			PageName:               pageName,
			NationalIDNumber:       req.NationalIDNumber,
			IsCaretaker:            req.IsCaretaker,
			AuthorizedByLandlordID: req.AuthorizedByLandlordID,
			VerificationStatus:     domain.StatusPending,
			CreatedAt:              now,
		}
		if err := s.repo.CreateLandlordProfile(ctx, profile); err != nil {
			return nil, err
		}
	}

	if err := s.sendOtp(ctx, user.ID, email); err != nil {
		if delErr := s.repo.DeleteUser(ctx, user.ID); delErr != nil {
			log.Printf("[auth] register cleanup for %s failed: %v", email, delErr)
		}
		log.Printf("[auth] register for %s failed to send OTP: %v", email, err)
		return nil, err
	}

	return &domain.RegisterResult{
		Email:   email,
		Message: "Account created. Enter the code sent to your email to verify your account.",
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

func (s *authService) VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (*domain.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Code == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.repo.GetUserByPhoneOrEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidOtp
	}

	code, expiresAt, _, err := s.repo.GetUserOTP(ctx, user.ID)
	if err != nil {
		return nil, domain.ErrInvalidOtp
	}
	if code == "" || code != req.Code {
		return nil, domain.ErrInvalidOtp
	}
	if !expiresAt.IsZero() && time.Now().After(expiresAt) {
		return nil, domain.ErrInvalidOtp
	}

	if err := s.repo.VerifyUserEmail(ctx, user.ID); err != nil {
		return nil, err
	}
	user.EmailVerified = true

	token, err := s.generateToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) ResendOtp(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return domain.ErrInvalidInput
	}

	user, err := s.repo.GetUserByPhoneOrEmail(ctx, email)
	if err != nil {
		return domain.ErrUserNotFound
	}
	if user.EmailVerified {
		return nil
	}

	_, _, sentAt, err := s.repo.GetUserOTP(ctx, user.ID)
	if err != nil {
		return err
	}
	if !sentAt.IsZero() && time.Since(sentAt) < s.otpCooldown {
		return domain.ErrOtpTooSoon
	}

	return s.sendOtp(ctx, user.ID, email)
}

func (s *authService) sendOtp(ctx context.Context, userID uuid.UUID, email string) error {
	code := generateOtp()
	now := time.Now()
	if err := s.repo.SetUserOTP(ctx, userID, code, now.Add(s.otpExpiry), now); err != nil {
		return err
	}

	if err := s.mailer.SendOTP(ctx, email, code); err != nil {
		return err
	}
	if s.isDev {
		log.Printf("[otp] verification code for %s: %s (expires in %s)", email, code, s.otpExpiry)
	}
	return nil
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

func generateOtp() string {
	const digits = "0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
		}
		b[i] = digits[n.Int64()]
	}
	return string(b)
}
