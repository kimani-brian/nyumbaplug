package service

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/repository"
)

type PropertyService interface {
	SearchProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error)
	GetPropertyDetail(ctx context.Context, id uuid.UUID) (*domain.Property, []domain.UnitCategory, error)
	GetUnitContact(ctx context.Context, categoryID uuid.UUID) (*domain.ContactInfoResponse, error)
	ReportProperty(ctx context.Context, tenantUserID, propertyID uuid.UUID, reason string) error
}

type propertyService struct {
	repo repository.Repository
}

func NewPropertyService(repo repository.Repository) PropertyService {
	return &propertyService{repo: repo}
}

var coordsRe = regexp.MustCompile(`@(-?\d+\.\d+),(-?\d+\.\d+)`)

func resolveMapCoords(mapsURL string) *string {
	if mapsURL == "" {
		return nil
	}

	// Try to extract coordinates directly from the URL first
	matches := coordsRe.FindStringSubmatch(mapsURL)
	if len(matches) == 3 {
		coords := matches[1] + "," + matches[2]
		return &coords
	}

	// For short URLs (goo.gl), follow the redirect to find coordinates
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", mapsURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	resp.Body.Close()

	finalURL := resp.Request.URL.String()
	matches = coordsRe.FindStringSubmatch(finalURL)
	if len(matches) == 3 {
		coords := matches[1] + "," + matches[2]
		return &coords
	}

	return nil
}

// Business Rule 2: Excludes properties whose landlords are revoked or unverified
func (s *propertyService) SearchProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	return s.repo.SearchVerifiedProperties(ctx, filter)
}

// Business Rule 2: Returns property details and categories ONLY for verified landlords
func (s *propertyService) GetPropertyDetail(ctx context.Context, id uuid.UUID) (*domain.Property, []domain.UnitCategory, error) {
	property, err := s.repo.GetVerifiedPropertyByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	categories, err := s.repo.GetCategoriesByPropertyID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	if property.MapsURL != nil && *property.MapsURL != "" {
		property.MapCoords = resolveMapCoords(*property.MapsURL)
	}

	return property, categories, nil
}

// Business Rule 3: Enforces contact detail gating (must be verified landlord)
func (s *propertyService) GetUnitContact(ctx context.Context, categoryID uuid.UUID) (*domain.ContactInfoResponse, error) {
	return s.repo.GetUnitContactDetails(ctx, categoryID)
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
