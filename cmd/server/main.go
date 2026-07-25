package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kenya-houses/backend/internal/config"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/handlers"
	"github.com/kenya-houses/backend/internal/middleware"
	"github.com/kenya-houses/backend/internal/repository"
	"github.com/kenya-houses/backend/internal/service"
	_ "github.com/lib/pq"
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

	r.Route("/api/v1", func(r chi.Router) {
		// Public Auth Endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Public & Tenant Browse Endpoints
		r.Get("/properties", propertyHandler.SearchProperties)
		r.Get("/properties/{id}", propertyHandler.GetPropertyDetail)
		r.Get("/units/{id}/contact", propertyHandler.GetUnitContact)

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
				r.Post("/landlord/properties", landlordHandler.CreateProperty)
				r.Get("/landlord/properties", landlordHandler.GetProperties)
				r.Post("/landlord/properties/{id}/units", landlordHandler.CreateUnit)
				r.Patch("/landlord/units/{id}", landlordHandler.UpdateUnit)
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
			})
		})
	})

	log.Printf("Server running on port %s", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
