package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/repository"
)

type TenantService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.TenantProfile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req domain.UpdateTenantProfileRequest) (*domain.TenantProfile, error)
}

type tenantService struct {
	repo repository.Repository
}

func NewTenantService(repo repository.Repository) TenantService {
	return &tenantService{repo: repo}
}

func (s *tenantService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.TenantProfile, error) {
	return s.repo.GetTenantProfileByUserID(ctx, userID)
}

func (s *tenantService) UpdateProfile(ctx context.Context, userID uuid.UUID, req domain.UpdateTenantProfileRequest) (*domain.TenantProfile, error) {
	profile, err := s.repo.GetTenantProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FullName != nil {
		profile.FullName = *req.FullName
	}
	if req.Location != nil {
		profile.Location = req.Location
	}

	if req.Phone != nil {
		if err := s.repo.UpdateUserPhone(ctx, userID, req.Phone); err != nil {
			return nil, err
		}
	}

	if err := s.repo.UpdateTenantProfile(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}
