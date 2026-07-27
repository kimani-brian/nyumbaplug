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
	SubmitVerificationRequest(ctx context.Context, userID uuid.UUID, req domain.SubmitVerificationRequest) (*domain.LandlordProfile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req domain.UpdateLandlordProfileRequest) (*domain.LandlordProfile, error)
	CreateProperty(ctx context.Context, userID uuid.UUID, req domain.CreatePropertyRequest) (*domain.Property, error)
	GetProperties(ctx context.Context, userID uuid.UUID) ([]domain.Property, error)
	DeleteProperty(ctx context.Context, userID, propertyID uuid.UUID) error
	UpdateProperty(ctx context.Context, userID, propertyID uuid.UUID, req domain.UpdatePropertyRequest) (*domain.Property, error)
	AddCategory(ctx context.Context, userID, propertyID uuid.UUID, req domain.CreateCategoryRequest) (*domain.UnitCategory, error)
	UpdateCategory(ctx context.Context, userID, categoryID uuid.UUID, req domain.UpdateCategoryRequest) (*domain.UnitCategory, error)
	DeleteCategory(ctx context.Context, userID, categoryID uuid.UUID) error
	AdjustCategoryQuantity(ctx context.Context, userID, categoryID uuid.UUID, delta int) error
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

func (s *landlordService) UpdateProfile(ctx context.Context, userID uuid.UUID, req domain.UpdateLandlordProfileRequest) (*domain.LandlordProfile, error) {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FullName != nil {
		profile.FullName = *req.FullName
	}
	if req.IDDocumentURL != nil {
		profile.IDDocumentURL = req.IDDocumentURL
	}

	if req.Phone != nil {
		if err := s.repo.UpdateUserPhone(ctx, userID, req.Phone); err != nil {
			return nil, err
		}
	}

	if err := s.repo.UpdateLandlordProfile(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *landlordService) SubmitVerificationRequest(ctx context.Context, userID uuid.UUID, req domain.SubmitVerificationRequest) (*domain.LandlordProfile, error) {
	if req.FullName == "" || req.NationalIDNumber == "" {
		return nil, domain.ErrInvalidInput
	}

	if req.Phone != nil {
		if err := s.repo.UpdateUserPhone(ctx, userID, req.Phone); err != nil {
			return nil, err
		}
	}

	profile := &domain.LandlordProfile{
		ID:                 uuid.New(),
		UserID:             userID,
		FullName:           req.FullName,
		NationalIDNumber:   req.NationalIDNumber,
		IDDocumentURL:      req.IDDocumentURL,
		VerificationStatus: domain.StatusPending,
		CreatedAt:          time.Now(),
	}
	if err := s.repo.CreateLandlordProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
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

	if profile.VerificationStatus != domain.StatusVerified {
		return nil, domain.ErrLandlordNotVerified
	}

	p := &domain.Property{
		ID:          uuid.New(),
		LandlordID:  profile.ID,
		Name:        req.Name,
		Location:    req.Location,
		County:      req.County,
		Address:     req.Address,
		MapsURL:     req.MapsURL,
		Description: req.Description,
		ImageURL:    req.ImageURL,
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

	properties, err := s.repo.GetPropertiesByLandlordID(ctx, profile.ID)
	if err != nil {
		return nil, err
	}

	for i := range properties {
		cats, err := s.repo.GetCategoriesByPropertyID(ctx, properties[i].ID)
		if err != nil {
			return nil, err
		}
		properties[i].Categories = cats
	}

	return properties, nil
}

func (s *landlordService) DeleteProperty(ctx context.Context, userID, propertyID uuid.UUID) error {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}

	property, err := s.repo.GetPropertyByID(ctx, propertyID)
	if err != nil {
		return err
	}

	if property.LandlordID != profile.ID {
		return domain.ErrForbidden
	}

	return s.repo.DeleteProperty(ctx, propertyID)
}

func (s *landlordService) UpdateProperty(ctx context.Context, userID, propertyID uuid.UUID, req domain.UpdatePropertyRequest) (*domain.Property, error) {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	property, err := s.repo.GetPropertyByID(ctx, propertyID)
	if err != nil {
		return nil, err
	}

	if property.LandlordID != profile.ID {
		return nil, domain.ErrForbidden
	}

	if req.Name != nil {
		property.Name = *req.Name
	}
	if req.Location != nil {
		property.Location = *req.Location
	}
	if req.County != nil {
		property.County = req.County
	}
	if req.Address != nil {
		property.Address = req.Address
	}
	if req.MapsURL != nil {
		property.MapsURL = req.MapsURL
	}
	if req.Description != nil {
		property.Description = req.Description
	}
	if req.ImageURL != nil {
		property.ImageURL = req.ImageURL
	}

	if err := s.repo.UpdateProperty(ctx, property); err != nil {
		return nil, err
	}

	return property, nil
}

// Category management

func (s *landlordService) AddCategory(ctx context.Context, userID, propertyID uuid.UUID, req domain.CreateCategoryRequest) (*domain.UnitCategory, error) {
	if req.Name == "" || req.RentAmount <= 0 {
		return nil, domain.ErrInvalidInput
	}

	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	property, err := s.repo.GetPropertyByID(ctx, propertyID)
	if err != nil {
		return nil, err
	}

	if property.LandlordID != profile.ID {
		return nil, domain.ErrForbidden
	}

	if profile.VerificationStatus != domain.StatusVerified {
		return nil, domain.ErrLandlordNotVerified
	}

	cat := &domain.UnitCategory{
		ID:                uuid.New(),
		PropertyID:        propertyID,
		Name:              req.Name,
		Description:       req.Description,
		RentAmount:        req.RentAmount,
		QuantityAvailable: req.QuantityAvailable,
		Photos:            req.Photos,
		VideoURL:          req.VideoURL,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

func (s *landlordService) UpdateCategory(ctx context.Context, userID, categoryID uuid.UUID, req domain.UpdateCategoryRequest) (*domain.UnitCategory, error) {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	cat, err := s.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	property, err := s.repo.GetPropertyByID(ctx, cat.PropertyID)
	if err != nil {
		return nil, err
	}

	if property.LandlordID != profile.ID {
		return nil, domain.ErrForbidden
	}

	if req.Name != nil {
		cat.Name = *req.Name
	}
	if req.Description != nil {
		cat.Description = req.Description
	}
	if req.RentAmount != nil {
		cat.RentAmount = *req.RentAmount
	}
	if req.QuantityAvailable != nil {
		cat.QuantityAvailable = *req.QuantityAvailable
	}
	if req.Photos != nil {
		cat.Photos = req.Photos
	}
	if req.VideoURL != nil {
		cat.VideoURL = req.VideoURL
	}

	if err := s.repo.UpdateCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

func (s *landlordService) DeleteCategory(ctx context.Context, userID, categoryID uuid.UUID) error {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}

	cat, err := s.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}

	property, err := s.repo.GetPropertyByID(ctx, cat.PropertyID)
	if err != nil {
		return err
	}

	if property.LandlordID != profile.ID {
		return domain.ErrForbidden
	}

	return s.repo.DeleteCategory(ctx, categoryID)
}

func (s *landlordService) AdjustCategoryQuantity(ctx context.Context, userID, categoryID uuid.UUID, delta int) error {
	profile, err := s.repo.GetLandlordProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}

	cat, err := s.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}

	property, err := s.repo.GetPropertyByID(ctx, cat.PropertyID)
	if err != nil {
		return err
	}

	if property.LandlordID != profile.ID {
		return domain.ErrForbidden
	}

	return s.repo.UpdateCategoryQuantity(ctx, categoryID, delta)
}
