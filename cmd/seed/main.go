package main

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/config"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load error: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	adminID := uuid.New()

	var phone *string
	query := `INSERT INTO users (id, role, phone, email, password_hash) VALUES ($1, 'admin', $2, $3, $4) ON CONFLICT (email) DO NOTHING`
	_, err = db.Exec(query, adminID, phone, cfg.AdminEmail, string(hash))
	if err != nil {
		log.Fatalf("Admin seeding failed: %v", err)
	}

	log.Printf("Admin account ready: Email (%s), Password (%s)", cfg.AdminEmail, cfg.AdminPassword)
}
