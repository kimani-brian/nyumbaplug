package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	_ "github.com/lib/pq"
)

type Repository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByPhoneOrEmail(ctx context.Context, identifier string) (*domain.User, error)

	CreateLandlordProfile(ctx context.Context, profile *domain.LandlordProfile) error
	GetLandlordProfileByID(ctx context.Context, id uuid.UUID) (*domain.LandlordProfile, error)
	GetLandlordProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error)
	GetLandlordProfilesByStatus(ctx context.Context, status string) ([]domain.LandlordProfile, error)
	UpdateLandlordVerification(ctx context.Context, profile *domain.LandlordProfile) error

	CreateTenantProfile(ctx context.Context, profile *domain.TenantProfile) error
	GetTenantProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.TenantProfile, error)

	CreateProperty(ctx context.Context, property *domain.Property) error
	GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error)
	GetVerifiedPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error)
	GetPropertiesByLandlordID(ctx context.Context, landlordID uuid.UUID) ([]domain.Property, error)
	SearchVerifiedProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error)

	CreateUnit(ctx context.Context, unit *domain.Unit) error
	GetUnitByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error)
	GetUnitsByPropertyID(ctx context.Context, propertyID uuid.UUID) ([]domain.Unit, error)
	UpdateUnit(ctx context.Context, unit *domain.Unit) error

	CreatePropertyReport(ctx context.Context, report *domain.PropertyReport) error
	GetPropertyReports(ctx context.Context, resolved *bool) ([]domain.PropertyReport, error)
	ResolvePropertyReport(ctx context.Context, id uuid.UUID) error

	CreateAuditLog(ctx context.Context, log *domain.AdminAuditLog) error
	GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error)

	GetUnitContactDetails(ctx context.Context, unitID uuid.UUID) (*domain.ContactInfoResponse, error)
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type PostgresRepo struct {
	db DBTX
}

func NewPostgresRepo(db DBTX) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) CreateUser(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (id, role, phone, email, password_hash, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, u.ID, u.Role, u.Phone, u.Email, u.PasswordHash, u.CreatedAt)
	return err
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u := &domain.User{}
	query := `SELECT id, role, phone, email, password_hash, created_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Role, &u.Phone, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepo) GetUserByPhoneOrEmail(ctx context.Context, identifier string) (*domain.User, error) {
	u := &domain.User{}
	query := `SELECT id, role, phone, email, password_hash, created_at FROM users WHERE phone = $1 OR email = $1`
	err := r.db.QueryRowContext(ctx, query, identifier).Scan(&u.ID, &u.Role, &u.Phone, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepo) CreateLandlordProfile(ctx context.Context, lp *domain.LandlordProfile) error {
	query := `INSERT INTO landlord_profiles (id, user_id, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, lp.ID, lp.UserID, lp.NationalIDNumber, lp.IDDocumentURL, lp.IsCaretaker, lp.AuthorizedByLandlordID, lp.VerificationStatus, lp.CreatedAt)
	return err
}

func (r *PostgresRepo) GetLandlordProfileByID(ctx context.Context, id uuid.UUID) (*domain.LandlordProfile, error) {
	lp := &domain.LandlordProfile{}
	query := `SELECT id, user_id, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, verified_by, verified_at, revoked_at, revoke_reason, created_at FROM landlord_profiles WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&lp.ID, &lp.UserID, &lp.NationalIDNumber, &lp.IDDocumentURL, &lp.IsCaretaker, &lp.AuthorizedByLandlordID, &lp.VerificationStatus, &lp.VerifiedBy, &lp.VerifiedAt, &lp.RevokedAt, &lp.RevokeReason, &lp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrLandlordNotFound
	}
	return lp, err
}

func (r *PostgresRepo) GetLandlordProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error) {
	lp := &domain.LandlordProfile{}
	query := `SELECT id, user_id, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, verified_by, verified_at, revoked_at, revoke_reason, created_at FROM landlord_profiles WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&lp.ID, &lp.UserID, &lp.NationalIDNumber, &lp.IDDocumentURL, &lp.IsCaretaker, &lp.AuthorizedByLandlordID, &lp.VerificationStatus, &lp.VerifiedBy, &lp.VerifiedAt, &lp.RevokedAt, &lp.RevokeReason, &lp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrLandlordNotFound
	}
	return lp, err
}

func (r *PostgresRepo) GetLandlordProfilesByStatus(ctx context.Context, status string) ([]domain.LandlordProfile, error) {
	query := `SELECT id, user_id, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, verified_by, verified_at, revoked_at, revoke_reason, created_at FROM landlord_profiles WHERE verification_status = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.LandlordProfile
	for rows.Next() {
		var lp domain.LandlordProfile
		if err := rows.Scan(&lp.ID, &lp.UserID, &lp.NationalIDNumber, &lp.IDDocumentURL, &lp.IsCaretaker, &lp.AuthorizedByLandlordID, &lp.VerificationStatus, &lp.VerifiedBy, &lp.VerifiedAt, &lp.RevokedAt, &lp.RevokeReason, &lp.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, lp)
	}
	return list, nil
}

func (r *PostgresRepo) UpdateLandlordVerification(ctx context.Context, lp *domain.LandlordProfile) error {
	query := `UPDATE landlord_profiles SET verification_status = $1, verified_by = $2, verified_at = $3, revoked_at = $4, revoke_reason = $5 WHERE id = $6`
	_, err := r.db.ExecContext(ctx, query, lp.VerificationStatus, lp.VerifiedBy, lp.VerifiedAt, lp.RevokedAt, lp.RevokeReason, lp.ID)
	return err
}

func (r *PostgresRepo) CreateTenantProfile(ctx context.Context, tp *domain.TenantProfile) error {
	query := `INSERT INTO tenant_profiles (id, user_id, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, tp.ID, tp.UserID, tp.CreatedAt)
	return err
}

func (r *PostgresRepo) GetTenantProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.TenantProfile, error) {
	tp := &domain.TenantProfile{}
	query := `SELECT id, user_id, created_at FROM tenant_profiles WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&tp.ID, &tp.UserID, &tp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return tp, err
}

func (r *PostgresRepo) CreateProperty(ctx context.Context, p *domain.Property) error {
	query := `INSERT INTO properties (id, landlord_id, name, location, address, description, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.LandlordID, p.Name, p.Location, p.Address, p.Description, p.CreatedAt)
	return err
}

func (r *PostgresRepo) GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	p := &domain.Property{}
	query := `SELECT id, landlord_id, name, location, address, description, created_at FROM properties WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.Address, &p.Description, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPropertyNotFound
	}
	return p, err
}

// Business Rule 2: Exclude properties from revoked/unverified landlords at the SQL query layer
func (r *PostgresRepo) GetVerifiedPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	p := &domain.Property{}
	query := `
		SELECT p.id, p.landlord_id, p.name, p.location, p.address, p.description, p.created_at 
		FROM properties p
		JOIN landlord_profiles lp ON p.landlord_id = lp.id
		WHERE p.id = $1 AND lp.verification_status = 'verified'`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.Address, &p.Description, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPropertyNotFound
	}
	return p, err
}

func (r *PostgresRepo) GetPropertiesByLandlordID(ctx context.Context, landlordID uuid.UUID) ([]domain.Property, error) {
	query := `SELECT id, landlord_id, name, location, address, description, created_at FROM properties WHERE landlord_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, landlordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Property
	for rows.Next() {
		var p domain.Property
		if err := rows.Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.Address, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// Business Rule 2: SQL query filtering to ensure only properties belonging to VERIFIED landlords appear
func (r *PostgresRepo) SearchVerifiedProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	query := `
		SELECT DISTINCT p.id, p.landlord_id, p.name, p.location, p.address, p.description, p.created_at
		FROM properties p
		JOIN landlord_profiles lp ON p.landlord_id = lp.id
		LEFT JOIN units u ON p.id = u.property_id
		WHERE lp.verification_status = 'verified'`

	args := []interface{}{}
	argId := 1

	if filter.Location != "" {
		query += fmt.Sprintf(" AND p.location ILIKE $%d", argId)
		args = append(args, "%"+filter.Location+"%")
		argId++
	}

	if filter.Bedrooms != nil {
		query += fmt.Sprintf(" AND u.bedrooms = $%d", argId)
		args = append(args, *filter.Bedrooms)
		argId++
	}

	if filter.UnitType != "" {
		query += fmt.Sprintf(" AND u.unit_type = $%d", argId)
		args = append(args, filter.UnitType)
		argId++
	}

	query += " ORDER BY p.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Property
	for rows.Next() {
		var p domain.Property
		if err := rows.Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.Address, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

func (r *PostgresRepo) CreateUnit(ctx context.Context, u *domain.Unit) error {
	query := `INSERT INTO units (id, property_id, unit_label, bedrooms, unit_type, rent_amount, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, u.ID, u.PropertyID, u.UnitLabel, u.Bedrooms, u.UnitType, u.RentAmount, u.Status, u.CreatedAt)
	return err
}

func (r *PostgresRepo) GetUnitByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error) {
	u := &domain.Unit{}
	query := `SELECT id, property_id, unit_label, bedrooms, unit_type, rent_amount, status, created_at FROM units WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.PropertyID, &u.UnitLabel, &u.Bedrooms, &u.UnitType, &u.RentAmount, &u.Status, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUnitNotFound
	}
	return u, err
}

func (r *PostgresRepo) GetUnitsByPropertyID(ctx context.Context, propertyID uuid.UUID) ([]domain.Unit, error) {
	query := `SELECT id, property_id, unit_label, bedrooms, unit_type, rent_amount, status, created_at FROM units WHERE property_id = $1 ORDER BY unit_label ASC`
	rows, err := r.db.QueryContext(ctx, query, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Unit
	for rows.Next() {
		var u domain.Unit
		if err := rows.Scan(&u.ID, &u.PropertyID, &u.UnitLabel, &u.Bedrooms, &u.UnitType, &u.RentAmount, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, nil
}

func (r *PostgresRepo) UpdateUnit(ctx context.Context, u *domain.Unit) error {
	query := `UPDATE units SET unit_label = $1, bedrooms = $2, unit_type = $3, rent_amount = $4, status = $5 WHERE id = $6`
	_, err := r.db.ExecContext(ctx, query, u.UnitLabel, u.Bedrooms, u.UnitType, u.RentAmount, u.Status, u.ID)
	return err
}

// Business Rule 3: Query join for tenant contact info enforcing landlord verification and unit status
func (r *PostgresRepo) GetUnitContactDetails(ctx context.Context, unitID uuid.UUID) (*domain.ContactInfoResponse, error) {
	query := `
		SELECT 
			u.id, p.name, u.unit_label, u.status, lp.verification_status,
			usr.phone, usr.email, lp.id, lp.user_id, lp.national_id_number,
			lp.id_document_url, lp.is_caretaker, lp.authorized_by_landlord_id,
			lp.verification_status, lp.created_at
		FROM units u
		JOIN properties p ON u.property_id = p.id
		JOIN landlord_profiles lp ON p.landlord_id = lp.id
		JOIN users usr ON lp.user_id = usr.id
		WHERE u.id = $1`

	var resp domain.ContactInfoResponse
	var unitStatus, landlordStatus string

	err := r.db.QueryRowContext(ctx, query, unitID).Scan(
		&resp.UnitID, &resp.PropertyName, &resp.UnitLabel, &unitStatus, &landlordStatus,
		&resp.LandlordPhone, &resp.LandlordEmail, &resp.LandlordProfile.ID, &resp.LandlordProfile.UserID,
		&resp.LandlordProfile.NationalIDNumber, &resp.LandlordProfile.IDDocumentURL, &resp.LandlordProfile.IsCaretaker,
		&resp.LandlordProfile.AuthorizedByLandlordID, &resp.LandlordProfile.VerificationStatus, &resp.LandlordProfile.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUnitNotFound
	}
	if err != nil {
		return nil, err
	}

	// Business Rule 3 Enforcement
	if landlordStatus != domain.StatusVerified || unitStatus != domain.UnitVacant {
		return nil, domain.ErrContactNotAvailable
	}

	return &resp, nil
}

func (r *PostgresRepo) CreatePropertyReport(ctx context.Context, rep *domain.PropertyReport) error {
	query := `INSERT INTO property_reports (id, property_id, reported_by, reason, resolved, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, rep.ID, rep.PropertyID, rep.ReportedBy, rep.Reason, rep.Resolved, rep.CreatedAt)
	return err
}

func (r *PostgresRepo) GetPropertyReports(ctx context.Context, resolved *bool) ([]domain.PropertyReport, error) {
	query := `SELECT id, property_id, reported_by, reason, resolved, created_at FROM property_reports`
	args := []interface{}{}
	if resolved != nil {
		query += " WHERE resolved = $1"
		args = append(args, *resolved)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PropertyReport
	for rows.Next() {
		var rep domain.PropertyReport
		if err := rows.Scan(&rep.ID, &rep.PropertyID, &rep.ReportedBy, &rep.Reason, &rep.Resolved, &rep.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, rep)
	}
	return list, nil
}

func (r *PostgresRepo) ResolvePropertyReport(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE property_reports SET resolved = true WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrReportNotFound
	}
	return nil
}

func (r *PostgresRepo) CreateAuditLog(ctx context.Context, log *domain.AdminAuditLog) error {
	query := `INSERT INTO admin_audit_log (id, admin_id, action, target_type, target_id, reason, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, log.ID, log.AdminID, log.Action, log.TargetType, log.TargetID, log.Reason, log.CreatedAt)
	return err
}

func (r *PostgresRepo) GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error) {
	query := `SELECT id, admin_id, action, target_type, target_id, reason, created_at FROM admin_audit_log ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.AdminAuditLog
	for rows.Next() {
		var log domain.AdminAuditLog
		if err := rows.Scan(&log.ID, &log.AdminID, &log.Action, &log.TargetType, &log.TargetID, &log.Reason, &log.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, log)
	}
	return list, nil
}
