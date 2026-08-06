package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/repository"
)

type AdminService interface {
	GetPendingVerifications(ctx context.Context, status string) ([]domain.LandlordProfile, error)
	ApproveLandlord(ctx context.Context, adminID, landlordID uuid.UUID) error
	RevokeLandlord(ctx context.Context, adminID, landlordID uuid.UUID, reason string) error
	GetReports(ctx context.Context, resolved *bool) ([]domain.PropertyReport, error)
	ResolveReport(ctx context.Context, adminID, reportID uuid.UUID) error
	GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error)
	GetCustomers(ctx context.Context) ([]domain.CustomerView, error)
	GetCustomerProfile(ctx context.Context, userID uuid.UUID) (*domain.CustomerProfile, error)
	GetAllPropertyManagers(ctx context.Context) ([]domain.PropertyManagerView, error)
	GetPropertyManagerProperties(ctx context.Context, landlordProfileID uuid.UUID) ([]domain.Property, error)
	GetPropertyManagerProfile(ctx context.Context, landlordProfileID uuid.UUID) (*domain.PropertyManagerDetail, error)
}

type adminService struct {
	repo repository.Repository
}

func NewAdminService(repo repository.Repository) AdminService {
	return &adminService{repo: repo}
}

func (s *adminService) GetPendingVerifications(ctx context.Context, status string) ([]domain.LandlordProfile, error) {
	if status == "" {
		status = domain.StatusPending
	}
	return s.repo.GetLandlordProfilesByStatus(ctx, status)
}

// Enforces Rule 4 (Audit logging) & Rule 5 (Caretaker validation)
func (s *adminService) ApproveLandlord(ctx context.Context, adminID, landlordID uuid.UUID) error {
	profile, err := s.repo.GetLandlordProfileByID(ctx, landlordID)
	if err != nil {
		return err
	}

	// Business Rule 5: Caretaker validation before approval
	if profile.IsCaretaker {
		if profile.AuthorizedByLandlordID == nil {
			return domain.ErrCaretakerNotAuthorized
		}

		authorizer, err := s.repo.GetLandlordProfileByID(ctx, *profile.AuthorizedByLandlordID)
		if err != nil {
			return domain.ErrAuthorizerNotVerified
		}

		if authorizer.VerificationStatus != domain.StatusVerified {
			return domain.ErrAuthorizerNotVerified
		}
	}

	now := time.Now()
	profile.VerificationStatus = domain.StatusVerified
	profile.VerifiedBy = &adminID
	profile.VerifiedAt = &now
	profile.RevokedAt = nil
	profile.RevokeReason = nil

	if err := s.repo.UpdateLandlordVerification(ctx, profile); err != nil {
		return err
	}

	// Business Rule 4: Audit log entry
	auditLog := &domain.AdminAuditLog{
		ID:         uuid.New(),
		AdminID:    adminID,
		Action:     "verify_landlord",
		TargetType: "landlord_profile",
		TargetID:   landlordID,
		CreatedAt:  now,
	}

	return s.repo.CreateAuditLog(ctx, auditLog)
}

// Enforces Rule 2 & Rule 4
func (s *adminService) RevokeLandlord(ctx context.Context, adminID, landlordID uuid.UUID, reason string) error {
	if reason == "" {
		return domain.ErrInvalidInput
	}

	profile, err := s.repo.GetLandlordProfileByID(ctx, landlordID)
	if err != nil {
		return err
	}

	now := time.Now()
	profile.VerificationStatus = domain.StatusRevoked
	profile.RevokedAt = &now
	profile.RevokeReason = &reason

	if err := s.repo.UpdateLandlordVerification(ctx, profile); err != nil {
		return err
	}

	// Business Rule 4: Audit log entry
	auditLog := &domain.AdminAuditLog{
		ID:         uuid.New(),
		AdminID:    adminID,
		Action:     "revoke_landlord",
		TargetType: "landlord_profile",
		TargetID:   landlordID,
		Reason:     &reason,
		CreatedAt:  now,
	}

	return s.repo.CreateAuditLog(ctx, auditLog)
}

func (s *adminService) GetReports(ctx context.Context, resolved *bool) ([]domain.PropertyReport, error) {
	return s.repo.GetPropertyReports(ctx, resolved)
}

func (s *adminService) ResolveReport(ctx context.Context, adminID, reportID uuid.UUID) error {
	report, err := s.repo.GetPropertyReportByID(ctx, reportID)
	if err != nil {
		return err
	}

	// Auto-restore the property manager flagged in the report (Rule: resolve closes the loop —
	// re-approves the landlord so their listings reappear in search).
	if report.LandlordID != uuid.Nil {
		if err := s.restoreLandlord(ctx, report.LandlordID); err != nil {
			return err
		}
	}

	if err := s.repo.ResolvePropertyReport(ctx, reportID); err != nil {
		return err
	}

	// Business Rule 4: Audit log entry
	auditLog := &domain.AdminAuditLog{
		ID:         uuid.New(),
		AdminID:    adminID,
		Action:     "resolve_report",
		TargetType: "property_report",
		TargetID:   reportID,
		CreatedAt:  time.Now(),
	}
	return s.repo.CreateAuditLog(ctx, auditLog)
}

// restoreLandlord re-approves a landlord profile, clearing any prior revocation.
func (s *adminService) restoreLandlord(ctx context.Context, landlordID uuid.UUID) error {
	profile, err := s.repo.GetLandlordProfileByID(ctx, landlordID)
	if err != nil {
		return err
	}

	profile.VerificationStatus = domain.StatusVerified
	profile.RevokedAt = nil
	profile.RevokeReason = nil

	return s.repo.UpdateLandlordVerification(ctx, profile)
}

func (s *adminService) GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error) {
	return s.repo.GetAuditLogs(ctx)
}

func (s *adminService) GetCustomers(ctx context.Context) ([]domain.CustomerView, error) {
	return s.repo.GetCustomers(ctx)
}

func (s *adminService) GetCustomerProfile(ctx context.Context, userID uuid.UUID) (*domain.CustomerProfile, error) {
	return s.repo.GetCustomerProfile(ctx, userID)
}

func (s *adminService) GetAllPropertyManagers(ctx context.Context) ([]domain.PropertyManagerView, error) {
	return s.repo.GetAllLandlordProfiles(ctx)
}

func (s *adminService) GetPropertyManagerProfile(ctx context.Context, landlordProfileID uuid.UUID) (*domain.PropertyManagerDetail, error) {
	return s.repo.GetPropertyManagerDetailByID(ctx, landlordProfileID)
}

func (s *adminService) GetPropertyManagerProperties(ctx context.Context, landlordProfileID uuid.UUID) ([]domain.Property, error) {
	properties, err := s.repo.GetPropertiesByLandlordID(ctx, landlordProfileID)
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
