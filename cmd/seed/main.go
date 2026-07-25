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

	hash, _ := bcrypt.GenerateFromPassword([]byte("AdminPass123!"), bcrypt.DefaultCost)
	adminID := uuid.New()

	query := `INSERT INTO users (id, role, phone, email, password_hash) VALUES ($1, 'admin', '+254700000000', 'admin@kenyahouses.co.ke', $2) ON CONFLICT (phone) DO NOTHING`
	_, err = db.Exec(query, adminID, string(hash))
	if err != nil {
		log.Fatalf("Admin seeding failed: %v", err)
	}

	log.Println("Admin account created: Phone (+254700000000), Password (AdminPass123!)")
}
