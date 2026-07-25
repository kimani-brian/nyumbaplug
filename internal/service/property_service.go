package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/repository"
)

type PropertyService interface {
	SearchProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error)
	GetPropertyDetail(ctx context.Context, id uuid.UUID) (*domain.Property, []domain.Unit, error)
	GetUnitContact(ctx context.Context, unitID uuid.UUID) (*domain.ContactInfoResponse, error)
	ReportProperty(ctx context.Context, tenantUserID, propertyID uuid.UUID, reason string) error
}

type propertyService struct {
	repo repository.Repository
}

func NewPropertyService(repo repository.Repository) PropertyService {
	return &propertyService{repo: repo}
}

// Business Rule 2: Excludes properties whose landlords are revoked or unverified
func (s *propertyService) SearchProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	return s.repo.SearchVerifiedProperties(ctx, filter)
}

// Business Rule 2: Returns property details and unit status ONLY for verified landlords
func (s *propertyService) GetPropertyDetail(ctx context.Context, id uuid.UUID) (*domain.Property, []domain.Unit, error) {
	property, err := s.repo.GetVerifiedPropertyByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	units, err := s.repo.GetUnitsByPropertyID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return property, units, nil
}

// Business Rule 3: Enforces contact detail gating (must be verified landlord AND vacant unit)
func (s *propertyService) GetUnitContact(ctx context.Context, unitID uuid.UUID) (*domain.ContactInfoResponse, error) {
	return s.repo.GetUnitContactDetails(ctx, unitID)
}

// Business Rule 6: Tenant property reporting
func (s *propertyService) ReportProperty(ctx context.Context, tenantUserID, propertyID uuid.UUID, reason string) error {
	if reason == "" {
		return domain.ErrInvalidInput
	}

	tenantProfile, err := s.repo.GetTenantProfileByUserID(ctx, tenantUserID)
	if err != nil {
		return err
	}

	// Verify property exists
	if _, err := s.repo.GetPropertyByID(ctx, propertyID); err != nil {
		return err
	}

	report := &domain.PropertyReport{
		ID:         uuid.New(),
		PropertyID: propertyID,
		ReportedBy: tenantProfile.ID,
		Reason:     reason,
		Resolved:   false,
		CreatedAt:  time.Now(),
	}

	return s.repo.CreatePropertyReport(ctx, report)
}
