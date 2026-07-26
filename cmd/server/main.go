package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/config"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/handlers"
	"github.com/kenya-houses/backend/internal/middleware"
	"github.com/kenya-houses/backend/internal/repository"
	"github.com/kenya-houses/backend/internal/service"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Postgres ping failed: %v", err)
	}

	repo := repository.NewPostgresRepo(db)

	ensureAdmin(context.Background(), repo, cfg)

	authSvc := service.NewAuthService(repo, cfg.JWTSecret)
	adminSvc := service.NewAdminService(repo)
	landlordSvc := service.NewLandlordService(repo)
	propertySvc := service.NewPropertyService(repo)

	authHandler := handlers.NewAuthHandler(authSvc)
	adminHandler := handlers.NewAdminHandler(adminSvc)
	landlordHandler := handlers.NewLandlordHandler(landlordSvc)
	propertyHandler := handlers.NewPropertyHandler(propertySvc)

	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, 0755)

	r.Route("/api/v1", func(r chi.Router) {
		// Public Auth Endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Public Browse Endpoints
		r.Get("/properties", propertyHandler.SearchProperties)
		r.Get("/properties/{id}", propertyHandler.GetPropertyDetail)
		r.Get("/categories/{id}/contact", propertyHandler.GetUnitContact)

		// Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(cfg.JWTSecret))

			// Tenant Only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleTenant))
				r.Post("/properties/{id}/report", propertyHandler.ReportProperty)
			})

			// Landlord Only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleLandlord))
				r.Get("/landlord/me", landlordHandler.GetMe)
				r.Post("/landlord/profile", landlordHandler.SubmitVerification)
				r.Post("/landlord/properties", landlordHandler.CreateProperty)
				r.Get("/landlord/properties", landlordHandler.GetProperties)
				r.Patch("/landlord/properties/{id}", landlordHandler.UpdateProperty)
				r.Delete("/landlord/properties/{id}", landlordHandler.DeleteProperty)
				r.Post("/landlord/properties/{id}/categories", landlordHandler.AddCategory)
				r.Patch("/landlord/categories/{id}", landlordHandler.UpdateCategory)
				r.Delete("/landlord/categories/{id}", landlordHandler.DeleteCategory)
				r.Post("/landlord/categories/{id}/quantity", landlordHandler.AdjustQuantity)
			})

			// Admin Only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole(domain.RoleAdmin))
				r.Get("/admin/verifications", adminHandler.GetVerifications)
				r.Post("/admin/verifications/{landlord_id}/approve", adminHandler.ApproveLandlord)
				r.Post("/admin/verifications/{landlord_id}/revoke", adminHandler.RevokeLandlord)
				r.Get("/admin/reports", adminHandler.GetReports)
				r.Post("/admin/reports/{id}/resolve", adminHandler.ResolveReport)
				r.Get("/admin/audit-log", adminHandler.GetAuditLog)
				r.Get("/admin/customers", adminHandler.GetCustomers)
				r.Get("/admin/agents", adminHandler.GetAllAgents)
				r.Get("/admin/agents/{landlord_id}/properties", adminHandler.GetAgentProperties)
			})

			// Upload (any authenticated user)
			r.Post("/upload", uploadHandler(uploadDir))
		})
	})

	// Serve uploaded files
	r.Get("/uploads/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir)))
		fs.ServeHTTP(w, r)
	})

	log.Printf("Server running on port %s", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func uploadHandler(uploadDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const maxUploadSize = 10 << 20
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, `{"error":"file too large (max 10MB)"}`, http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"no file provided"}`, http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := filepath.Ext(header.Filename)
		filename := uuid.New().String() + ext
		destPath := filepath.Join(uploadDir, filename)

		dest, err := os.Create(destPath)
		if err != nil {
			http.Error(w, `{"error":"failed to save file"}`, http.StatusInternalServerError)
			return
		}
		defer dest.Close()

		if _, err := io.Copy(dest, file); err != nil {
			http.Error(w, `{"error":"failed to write file"}`, http.StatusInternalServerError)
			return
		}

		url := fmt.Sprintf("/uploads/%s", filename)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": url})
	}
}

func ensureAdmin(ctx context.Context, repo repository.Repository, cfg *config.Config) {
	existing, _ := repo.GetUserByPhoneOrEmail(ctx, cfg.AdminEmail)
	if existing != nil {
		log.Printf("Admin account found: %s", cfg.AdminEmail)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Role:         domain.RoleAdmin,
		Email:        &cfg.AdminEmail,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("Admin account created: %s", cfg.AdminEmail)
}
