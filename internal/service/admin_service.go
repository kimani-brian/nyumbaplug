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
	ResolveReport(ctx context.Context, reportID uuid.UUID) error
	GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error)
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

func (s *adminService) ResolveReport(ctx context.Context, reportID uuid.UUID) error {
	return s.repo.ResolvePropertyReport(ctx, reportID)
}

func (s *adminService) GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error) {
	return s.repo.GetAuditLogs(ctx)
}
