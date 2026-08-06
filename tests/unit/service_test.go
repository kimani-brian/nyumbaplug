package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) CreateUser(ctx context.Context, u *domain.User) error { return nil }
func (m *MockRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}
func (m *MockRepo) GetUserByPhoneOrEmail(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}
func (m *MockRepo) UpdateUserPhone(ctx context.Context, userID uuid.UUID, phone *string) error {
	return nil
}
func (m *MockRepo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *MockRepo) SetUserOTP(ctx context.Context, userID uuid.UUID, code string, expiresAt, sentAt time.Time) error {
	return nil
}
func (m *MockRepo) GetUserOTP(ctx context.Context, userID uuid.UUID) (string, time.Time, time.Time, error) {
	return "", time.Time{}, time.Time{}, nil
}
func (m *MockRepo) VerifyUserEmail(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (m *MockRepo) CreateLandlordProfile(ctx context.Context, p *domain.LandlordProfile) error {
	return nil
}
func (m *MockRepo) GetLandlordProfileByID(ctx context.Context, id uuid.UUID) (*domain.LandlordProfile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandlordProfile), args.Error(1)
}
func (m *MockRepo) GetLandlordProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LandlordProfile), args.Error(1)
}
func (m *MockRepo) GetLandlordProfilesByStatus(ctx context.Context, s string) ([]domain.LandlordProfile, error) {
	return nil, nil
}
func (m *MockRepo) UpdateLandlordVerification(ctx context.Context, p *domain.LandlordProfile) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockRepo) UpdateLandlordProfile(ctx context.Context, p *domain.LandlordProfile) error {
	return nil
}
func (m *MockRepo) CreateTenantProfile(ctx context.Context, p *domain.TenantProfile) error {
	return nil
}
func (m *MockRepo) GetTenantProfileByUserID(ctx context.Context, id uuid.UUID) (*domain.TenantProfile, error) {
	return nil, nil
}
func (m *MockRepo) UpdateTenantProfile(ctx context.Context, p *domain.TenantProfile) error {
	return nil
}
func (m *MockRepo) GetCustomerProfile(ctx context.Context, id uuid.UUID) (*domain.CustomerProfile, error) {
	return nil, nil
}
func (m *MockRepo) CreateProperty(ctx context.Context, p *domain.Property) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockRepo) GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	return nil, nil
}
func (m *MockRepo) GetVerifiedPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	return nil, nil
}
func (m *MockRepo) GetPropertiesByLandlordID(ctx context.Context, id uuid.UUID) ([]domain.Property, error) {
	return nil, nil
}
func (m *MockRepo) SearchVerifiedProperties(ctx context.Context, f domain.PropertyFilter) ([]domain.Property, error) {
	return nil, nil
}
func (m *MockRepo) UpdateProperty(ctx context.Context, p *domain.Property) error { return nil }
func (m *MockRepo) DeleteProperty(ctx context.Context, id uuid.UUID) error       { return nil }
func (m *MockRepo) CreateCategory(ctx context.Context, c *domain.UnitCategory) error {
	return nil
}
func (m *MockRepo) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.UnitCategory, error) {
	return nil, nil
}
func (m *MockRepo) GetCategoriesByPropertyID(ctx context.Context, id uuid.UUID) ([]domain.UnitCategory, error) {
	return nil, nil
}
func (m *MockRepo) UpdateCategory(ctx context.Context, c *domain.UnitCategory) error { return nil }
func (m *MockRepo) DeleteCategory(ctx context.Context, id uuid.UUID) error           { return nil }
func (m *MockRepo) UpdateCategoryQuantity(ctx context.Context, id uuid.UUID, delta int) error {
	return nil
}
func (m *MockRepo) CreatePropertyReport(ctx context.Context, r *domain.PropertyReport) error {
	return nil
}
func (m *MockRepo) GetPropertyReports(ctx context.Context, res *bool) ([]domain.PropertyReport, error) {
	return nil, nil
}
func (m *MockRepo) GetPropertyReportByID(ctx context.Context, id uuid.UUID) (*domain.PropertyReport, error) {
	return nil, nil
}
func (m *MockRepo) ResolvePropertyReport(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockRepo) CreateAuditLog(ctx context.Context, l *domain.AdminAuditLog) error {
	return m.Called(ctx, l).Error(0)
}
func (m *MockRepo) GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error) { return nil, nil }
func (m *MockRepo) GetCustomers(ctx context.Context) ([]domain.CustomerView, error)  { return nil, nil }
func (m *MockRepo) GetAllLandlordProfiles(ctx context.Context) ([]domain.PropertyManagerView, error) {
	return nil, nil
}
func (m *MockRepo) GetPropertyManagerDetailByID(ctx context.Context, id uuid.UUID) (*domain.PropertyManagerDetail, error) {
	return nil, nil
}
func (m *MockRepo) GetUnitContactDetails(ctx context.Context, unitID uuid.UUID) (*domain.ContactInfoResponse, error) {
	args := m.Called(ctx, unitID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ContactInfoResponse), args.Error(1)
}

// Business Rule 1 Test: Unverified Landlords Cannot Create Properties
func TestCreateProperty_UnverifiedLandlord(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := service.NewLandlordService(mockRepo)

	userID := uuid.New()
	mockRepo.On("GetLandlordProfileByUserID", mock.Anything, userID).Return(&domain.LandlordProfile{
		ID:                 uuid.New(),
		UserID:             userID,
		VerificationStatus: domain.StatusPending,
	}, nil)

	_, err := svc.CreateProperty(context.Background(), userID, domain.CreatePropertyRequest{
		Name:     "Kileleshwa Heights",
		Location: "Nairobi",
	})

	assert.ErrorIs(t, err, domain.ErrLandlordNotVerified)
}

// Business Rule 3 Test: Gated Contact Info for Occupied/Unverified Units
func TestGetUnitContact_OccupiedOrUnverified(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := service.NewPropertyService(mockRepo)

	unitID := uuid.New()
	mockRepo.On("GetUnitContactDetails", mock.Anything, unitID).Return(nil, domain.ErrContactNotAvailable)

	_, err := svc.GetUnitContact(context.Background(), unitID)
	assert.ErrorIs(t, err, domain.ErrContactNotAvailable)
}

// Business Rule 5 Test: Caretaker Approval Requires Verified Authorizing Landlord
func TestApproveLandlord_CaretakerUnverifiedAuthorizer(t *testing.T) {
	mockRepo := new(MockRepo)
	svc := service.NewAdminService(mockRepo)

	adminID := uuid.New()
	caretakerID := uuid.New()
	authorizerID := uuid.New()

	mockRepo.On("GetLandlordProfileByID", mock.Anything, caretakerID).Return(&domain.LandlordProfile{
		ID:                     caretakerID,
		IsCaretaker:            true,
		AuthorizedByLandlordID: &authorizerID,
		VerificationStatus:     domain.StatusPending,
	}, nil)

	mockRepo.On("GetLandlordProfileByID", mock.Anything, authorizerID).Return(&domain.LandlordProfile{
		ID:                 authorizerID,
		VerificationStatus: domain.StatusPending, // Not verified!
	}, nil)

	err := svc.ApproveLandlord(context.Background(), adminID, caretakerID)
	assert.ErrorIs(t, err, domain.ErrAuthorizerNotVerified)
}
