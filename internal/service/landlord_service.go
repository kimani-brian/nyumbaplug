package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/repository"
)

type LandlordService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error)
	CreateProperty(ctx context.Context, userID uuid.UUID, req domain.CreatePropertyRequest) (*domain.Property, error)
	GetProperties(ctx context.Context, userID uuid.UUID) ([]domain.Property, error)
	CreateUnit(ctx context.Context, userID, propertyID uuid.UUID, req domain.CreateUnitRequest) (*domain.Unit, error)
	UpdateUnit(ctx context.Context, userID, unitID uuid.UUID, req domain.UpdateUnitRequest) (*domain.Unit, error)
}

type landlordService struct {
	repo repository.Repository
}

func NewLandlordService(repo repository.Repository) LandlordService {
	return &landlordService{repo: repo}
}

func (s *landlordService) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error) {
	return s.repo.GetLandlordProfileByUserID(ctx, userID)
}

// Business Rule 1: Verified landlord requirement enforced
func (s *landlordService) CreateProperty(ctx context.Context, userID uuid.UUID, req domain.CreatePropertyRequest) (*domain.Property, error) {
	if req.Name == "" || req.Location == "" {
		return nil, domain.ErrInvalidInput
	}

	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Business Rule 1: A landlord cannot create properties until verification_status = 'verified'
	if profile.VerificationStatus != domain.StatusVerified {
		return nil, domain.ErrLandlordNotVerified
	}

	p := &domain.Property{
		ID:          uuid.New(),
		LandlordID:  profile.ID,
		Name:        req.Name,
		Location:    req.Location,
		Address:     req.Address,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateProperty(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (s *landlordService) GetProperties(ctx context.Context, userID uuid.UUID) ([]domain.Property, error) {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetPropertiesByLandlordID(ctx, profile.ID)
}

// Business Rule 1: Verified landlord requirement enforced
func (s *landlordService) CreateUnit(ctx context.Context, userID, propertyID uuid.UUID, req domain.CreateUnitRequest) (*domain.Unit, error) {
	if req.UnitLabel == "" || req.Bedrooms < 0 || req.RentAmount <= 0 {
		return nil, domain.ErrInvalidInput
	}

	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if profile.VerificationStatus != domain.StatusVerified {
		return nil, domain.ErrLandlordNotVerified
	}

	property, err := s.repo.GetPropertyByID(ctx, propertyID)
	if err != nil {
		return nil, err
	}

	if property.LandlordID != profile.ID {
		return nil, domain.ErrForbidden
	}

	u := &domain.Unit{
		ID:         uuid.New(),
		PropertyID: propertyID,
		UnitLabel:  req.UnitLabel,
		Bedrooms:   req.Bedrooms,
		UnitType:   req.UnitType,
		RentAmount: req.RentAmount,
		Status:     domain.UnitVacant,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateUnit(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *landlordService) UpdateUnit(ctx context.Context, userID, unitID uuid.UUID, req domain.UpdateUnitRequest) (*domain.Unit, error) {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if profile.VerificationStatus != domain.StatusVerified {
		return nil, domain.ErrLandlordNotVerified
	}

	unit, err := s.repo.GetUnitByID(ctx, unitID)
	if err != nil {
		return nil, err
	}

	property, err := s.repo.GetPropertyByID(ctx, unit.PropertyID)
	if err != nil {
		return nil, err
	}

	if property.LandlordID != profile.ID {
		return nil, domain.ErrForbidden
	}

	if req.UnitLabel != nil {
		unit.UnitLabel = *req.UnitLabel
	}
	if req.Bedrooms != nil {
		unit.Bedrooms = *req.Bedrooms
	}
	if req.UnitType != nil {
		unit.UnitType = *req.UnitType
	}
	if req.RentAmount != nil {
		unit.RentAmount = *req.RentAmount
	}
	if req.Status != nil {
		unit.Status = *req.Status
	}

	if err := s.repo.UpdateUnit(ctx, unit); err != nil {
		return nil, err
	}

	return unit, nil
}
