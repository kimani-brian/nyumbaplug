package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	Environment   string
	AdminEmail    string
	AdminPassword string
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

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		JWTSecret:     jwtSecret,
		Environment:   env,
		AdminEmail:    adminEmail,
		AdminPassword: adminPassword,
	}, nil
}
