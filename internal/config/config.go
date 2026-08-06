package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	Environment        string
	AdminEmail         string
	AdminPassword      string
	ResendAPIKey       string
	ResendFromEmail    string
	OTPExpiryMinutes   int
	OTPCooldownSeconds int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/kenyahouses?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-dev-secret-key-change-me"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@nyumbaplug.com"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "AdminPass123!"
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")

	resendFromEmail := os.Getenv("RESEND_FROM_EMAIL")
	if resendFromEmail == "" {
		resendFromEmail = "NyumbaPlug <onboarding@resend.dev>"
	}

	otpExpiryMinutes := 10
	if v := os.Getenv("OTP_EXPIRY_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			otpExpiryMinutes = n
		}
	}

	otpCooldownSeconds := 60
	if v := os.Getenv("OTP_COOLDOWN_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			otpCooldownSeconds = n
		}
	}

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		Environment:        env,
		AdminEmail:         adminEmail,
		AdminPassword:      adminPassword,
		ResendAPIKey:       resendAPIKey,
		ResendFromEmail:    resendFromEmail,
		OTPExpiryMinutes:   otpExpiryMinutes,
		OTPCooldownSeconds: otpCooldownSeconds,
	}, nil
}
