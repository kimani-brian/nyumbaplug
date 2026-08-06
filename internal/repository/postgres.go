package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	_ "github.com/lib/pq"
)

type Repository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByPhoneOrEmail(ctx context.Context, identifier string) (*domain.User, error)
	UpdateUserPhone(ctx context.Context, userID uuid.UUID, phone *string) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	SetUserOTP(ctx context.Context, userID uuid.UUID, code string, expiresAt, sentAt time.Time) error
	GetUserOTP(ctx context.Context, userID uuid.UUID) (code string, expiresAt, sentAt time.Time, err error)
	VerifyUserEmail(ctx context.Context, userID uuid.UUID) error

	CreateLandlordProfile(ctx context.Context, profile *domain.LandlordProfile) error
	GetLandlordProfileByID(ctx context.Context, id uuid.UUID) (*domain.LandlordProfile, error)
	GetLandlordProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error)
	GetLandlordProfilesByStatus(ctx context.Context, status string) ([]domain.LandlordProfile, error)
	UpdateLandlordVerification(ctx context.Context, profile *domain.LandlordProfile) error
	UpdateLandlordProfile(ctx context.Context, profile *domain.LandlordProfile) error

	CreateTenantProfile(ctx context.Context, profile *domain.TenantProfile) error
	GetTenantProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.TenantProfile, error)
	UpdateTenantProfile(ctx context.Context, profile *domain.TenantProfile) error
	GetCustomerProfile(ctx context.Context, userID uuid.UUID) (*domain.CustomerProfile, error)

	CreateProperty(ctx context.Context, property *domain.Property) error
	GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error)
	GetVerifiedPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error)
	GetPropertiesByLandlordID(ctx context.Context, landlordID uuid.UUID) ([]domain.Property, error)
	SearchVerifiedProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error)
	UpdateProperty(ctx context.Context, p *domain.Property) error
	DeleteProperty(ctx context.Context, id uuid.UUID) error

	CreateCategory(ctx context.Context, cat *domain.UnitCategory) error
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.UnitCategory, error)
	GetCategoriesByPropertyID(ctx context.Context, propertyID uuid.UUID) ([]domain.UnitCategory, error)
	UpdateCategory(ctx context.Context, cat *domain.UnitCategory) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	UpdateCategoryQuantity(ctx context.Context, id uuid.UUID, delta int) error

	CreatePropertyReport(ctx context.Context, report *domain.PropertyReport) error
	GetPropertyReports(ctx context.Context, resolved *bool) ([]domain.PropertyReport, error)
	GetPropertyReportByID(ctx context.Context, id uuid.UUID) (*domain.PropertyReport, error)
	ResolvePropertyReport(ctx context.Context, id uuid.UUID) error

	CreateAuditLog(ctx context.Context, log *domain.AdminAuditLog) error
	GetAuditLogs(ctx context.Context) ([]domain.AdminAuditLog, error)

	GetCustomers(ctx context.Context) ([]domain.CustomerView, error)
	GetAllLandlordProfiles(ctx context.Context) ([]domain.PropertyManagerView, error)
	GetPropertyManagerDetailByID(ctx context.Context, id uuid.UUID) (*domain.PropertyManagerDetail, error)

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
	query := `INSERT INTO users (id, role, phone, email, password_hash, email_verified, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	var phone interface{} = u.Phone
	if u.Phone == nil {
		phone = nil
	}
	_, err := r.db.ExecContext(ctx, query, u.ID, u.Role, phone, u.Email, u.PasswordHash, u.EmailVerified, u.CreatedAt)
	return err
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u := &domain.User{}
	query := `SELECT id, role, phone, email, password_hash, email_verified, created_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Role, &u.Phone, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepo) GetUserByPhoneOrEmail(ctx context.Context, identifier string) (*domain.User, error) {
	u := &domain.User{}
	query := `SELECT id, role, phone, email, password_hash, email_verified, created_at FROM users WHERE phone = $1 OR email = $1`
	err := r.db.QueryRowContext(ctx, query, identifier).Scan(&u.ID, &u.Role, &u.Phone, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *PostgresRepo) SetUserOTP(ctx context.Context, userID uuid.UUID, code string, expiresAt, sentAt time.Time) error {
	query := `UPDATE users SET otp_code = $1, otp_expires_at = $2, otp_sent_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, code, expiresAt, sentAt, userID)
	return err
}

func (r *PostgresRepo) GetUserOTP(ctx context.Context, userID uuid.UUID) (string, time.Time, time.Time, error) {
	var code sql.NullString
	var expiresAt, sentAt sql.NullTime
	query := `SELECT otp_code, otp_expires_at, otp_sent_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&code, &expiresAt, &sentAt)
	if err != nil {
		return "", time.Time{}, time.Time{}, err
	}
	var c string
	var e, s time.Time
	if code.Valid {
		c = code.String
	}
	if expiresAt.Valid {
		e = expiresAt.Time
	}
	if sentAt.Valid {
		s = sentAt.Time
	}
	return c, e, s, nil
}

func (r *PostgresRepo) VerifyUserEmail(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET email_verified = TRUE, otp_code = NULL, otp_expires_at = NULL, otp_sent_at = NULL WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresRepo) UpdateUserPhone(ctx context.Context, userID uuid.UUID, phone *string) error {
	query := `UPDATE users SET phone = $1 WHERE id = $2`
	var p interface{} = phone
	if phone == nil {
		p = nil
	}
	_, err := r.db.ExecContext(ctx, query, p, userID)
	return err
}

func (r *PostgresRepo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresRepo) CreateLandlordProfile(ctx context.Context, lp *domain.LandlordProfile) error {
	query := `INSERT INTO landlord_profiles (id, user_id, full_name, page_name, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query, lp.ID, lp.UserID, lp.FullName, lp.PageName, lp.NationalIDNumber, lp.IDDocumentURL, lp.IsCaretaker, lp.AuthorizedByLandlordID, lp.VerificationStatus, lp.CreatedAt)
	return err
}

func (r *PostgresRepo) GetLandlordProfileByID(ctx context.Context, id uuid.UUID) (*domain.LandlordProfile, error) {
	lp := &domain.LandlordProfile{}
	query := `SELECT id, user_id, full_name, page_name, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, verified_by, verified_at, revoked_at, revoke_reason, created_at FROM landlord_profiles WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&lp.ID, &lp.UserID, &lp.FullName, &lp.PageName, &lp.NationalIDNumber, &lp.IDDocumentURL, &lp.IsCaretaker, &lp.AuthorizedByLandlordID, &lp.VerificationStatus, &lp.VerifiedBy, &lp.VerifiedAt, &lp.RevokedAt, &lp.RevokeReason, &lp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrLandlordNotFound
	}
	return lp, err
}

func (r *PostgresRepo) GetLandlordProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.LandlordProfile, error) {
	lp := &domain.LandlordProfile{}
	query := `SELECT id, user_id, full_name, page_name, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, verified_by, verified_at, revoked_at, revoke_reason, created_at FROM landlord_profiles WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&lp.ID, &lp.UserID, &lp.FullName, &lp.PageName, &lp.NationalIDNumber, &lp.IDDocumentURL, &lp.IsCaretaker, &lp.AuthorizedByLandlordID, &lp.VerificationStatus, &lp.VerifiedBy, &lp.VerifiedAt, &lp.RevokedAt, &lp.RevokeReason, &lp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrLandlordNotFound
	}
	return lp, err
}

func (r *PostgresRepo) GetLandlordProfilesByStatus(ctx context.Context, status string) ([]domain.LandlordProfile, error) {
	query := `SELECT id, user_id, full_name, page_name, national_id_number, id_document_url, is_caretaker, authorized_by_landlord_id, verification_status, verified_by, verified_at, revoked_at, revoke_reason, created_at FROM landlord_profiles WHERE verification_status = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.LandlordProfile
	for rows.Next() {
		var lp domain.LandlordProfile
		if err := rows.Scan(&lp.ID, &lp.UserID, &lp.FullName, &lp.PageName, &lp.NationalIDNumber, &lp.IDDocumentURL, &lp.IsCaretaker, &lp.AuthorizedByLandlordID, &lp.VerificationStatus, &lp.VerifiedBy, &lp.VerifiedAt, &lp.RevokedAt, &lp.RevokeReason, &lp.CreatedAt); err != nil {
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

func (r *PostgresRepo) UpdateLandlordProfile(ctx context.Context, lp *domain.LandlordProfile) error {
	query := `UPDATE landlord_profiles SET full_name = $1, page_name = $2, id_document_url = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, lp.FullName, lp.PageName, lp.IDDocumentURL, lp.ID)
	return err
}

func (r *PostgresRepo) CreateTenantProfile(ctx context.Context, tp *domain.TenantProfile) error {
	query := `INSERT INTO tenant_profiles (id, user_id, full_name, location, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, tp.ID, tp.UserID, tp.FullName, tp.Location, tp.CreatedAt)
	return err
}

func (r *PostgresRepo) GetTenantProfileByUserID(ctx context.Context, userID uuid.UUID) (*domain.TenantProfile, error) {
	tp := &domain.TenantProfile{}
	query := `SELECT id, user_id, full_name, location, created_at FROM tenant_profiles WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&tp.ID, &tp.UserID, &tp.FullName, &tp.Location, &tp.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return tp, err
}

func (r *PostgresRepo) UpdateTenantProfile(ctx context.Context, tp *domain.TenantProfile) error {
	query := `UPDATE tenant_profiles SET full_name = $1, location = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, tp.FullName, tp.Location, tp.ID)
	return err
}

func (r *PostgresRepo) GetCustomerProfile(ctx context.Context, userID uuid.UUID) (*domain.CustomerProfile, error) {
	var cp domain.CustomerProfile
	query := `
		SELECT u.id, u.email, u.phone, COALESCE(tp.full_name, ''), tp.location, u.created_at,
		       tp.id, tp.user_id, tp.full_name, tp.location, tp.created_at
		FROM users u
		JOIN tenant_profiles tp ON u.id = tp.user_id
		WHERE u.id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&cp.ID, &cp.Email, &cp.Phone, &cp.FullName, &cp.Location, &cp.CreatedAt,
		&cp.Profile.ID, &cp.Profile.UserID, &cp.Profile.FullName, &cp.Profile.Location, &cp.Profile.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return &cp, err
}

func (r *PostgresRepo) CreateProperty(ctx context.Context, p *domain.Property) error {
	query := `INSERT INTO properties (id, landlord_id, name, location, county, address, maps_url, description, image_url, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.LandlordID, p.Name, p.Location, p.County, p.Address, p.MapsURL, p.Description, p.ImageURL, p.CreatedAt)
	return err
}

func (r *PostgresRepo) GetPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	p := &domain.Property{}
	query := `SELECT id, landlord_id, name, location, county, address, maps_url, description, image_url, created_at FROM properties WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.County, &p.Address, &p.MapsURL, &p.Description, &p.ImageURL, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPropertyNotFound
	}
	return p, err
}

// Business Rule 2: Exclude properties from revoked/unverified landlords at the SQL query layer
func (r *PostgresRepo) GetVerifiedPropertyByID(ctx context.Context, id uuid.UUID) (*domain.Property, error) {
	p := &domain.Property{}
	query := `
		SELECT p.id, p.landlord_id, p.name, p.location, p.county, p.address, p.maps_url, p.description, p.image_url, p.created_at 
		FROM properties p
		JOIN landlord_profiles lp ON p.landlord_id = lp.id
		WHERE p.id = $1 AND lp.verification_status = 'verified'`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.County, &p.Address, &p.MapsURL, &p.Description, &p.ImageURL, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPropertyNotFound
	}
	return p, err
}

func (r *PostgresRepo) GetPropertiesByLandlordID(ctx context.Context, landlordID uuid.UUID) ([]domain.Property, error) {
	query := `SELECT id, landlord_id, name, location, county, address, maps_url, description, image_url, created_at FROM properties WHERE landlord_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, landlordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Property
	for rows.Next() {
		var p domain.Property
		if err := rows.Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.County, &p.Address, &p.MapsURL, &p.Description, &p.ImageURL, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, nil
}

// Business Rule 2: SQL query filtering to ensure only properties belonging to VERIFIED landlords appear
func (r *PostgresRepo) SearchVerifiedProperties(ctx context.Context, filter domain.PropertyFilter) ([]domain.Property, error) {
	query := `
		SELECT p.id, p.landlord_id, p.name, p.location, p.county, p.address, p.maps_url, p.description, p.image_url, p.created_at,
		       COALESCE(MIN(c.rent_amount), 0) as min_rent,
		       COALESCE(SUM(c.quantity_available), 0) as total_units,
		       COALESCE(lp.full_name, '') as landlord_name
		FROM properties p
		JOIN landlord_profiles lp ON p.landlord_id = lp.id
		LEFT JOIN unit_categories c ON c.property_id = p.id
		WHERE lp.verification_status = 'verified'`

	args := []interface{}{}
	argId := 1

	if filter.Query != "" {
		q := "%" + filter.Query + "%"
		query += fmt.Sprintf(" AND (p.location ILIKE $%d OR p.county ILIKE $%d OR p.name ILIKE $%d OR p.description ILIKE $%d OR EXISTS (SELECT 1 FROM unit_categories c2 WHERE c2.property_id = p.id AND c2.name ILIKE $%d))", argId, argId, argId, argId, argId)
		args = append(args, q)
		argId++
	}

	if filter.County != "" {
		query += fmt.Sprintf(" AND p.county ILIKE $%d", argId)
		args = append(args, "%"+filter.County+"%")
		argId++
	}

	query += ` GROUP BY p.id, p.landlord_id, p.name, p.location, p.county, p.address, p.maps_url, p.description, p.image_url, p.created_at, lp.full_name`

	havingAdded := false
	if filter.MinRent != nil && *filter.MinRent > 0 {
		query += fmt.Sprintf(" HAVING COALESCE(MIN(c.rent_amount), 0) >= $%d", argId)
		args = append(args, *filter.MinRent)
		argId++
		havingAdded = true
	}
	if filter.MaxRent != nil && *filter.MaxRent > 0 {
		if havingAdded {
			query += fmt.Sprintf(" AND COALESCE(MIN(c.rent_amount), 0) <= $%d", argId)
		} else {
			query += fmt.Sprintf(" HAVING COALESCE(MIN(c.rent_amount), 0) <= $%d", argId)
		}
		args = append(args, *filter.MaxRent)
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
		var minRent float64
		var totalUnits int
		if err := rows.Scan(&p.ID, &p.LandlordID, &p.Name, &p.Location, &p.County, &p.Address, &p.MapsURL, &p.Description, &p.ImageURL, &p.CreatedAt, &minRent, &totalUnits, &p.LandlordName); err != nil {
			return nil, err
		}
		p.MinRent = &minRent
		p.TotalUnits = &totalUnits
		list = append(list, p)
	}
	return list, nil
}

func (r *PostgresRepo) UpdateProperty(ctx context.Context, p *domain.Property) error {
	query := `UPDATE properties SET name = $1, location = $2, county = $3, address = $4, maps_url = $5, description = $6, image_url = $7 WHERE id = $8`
	_, err := r.db.ExecContext(ctx, query, p.Name, p.Location, p.County, p.Address, p.MapsURL, p.Description, p.ImageURL, p.ID)
	return err
}

func (r *PostgresRepo) DeleteProperty(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM unit_categories WHERE property_id = $1`, id)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM properties WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrPropertyNotFound
	}
	return nil
}

func (r *PostgresRepo) CreateCategory(ctx context.Context, cat *domain.UnitCategory) error {
	photos, err := json.Marshal(cat.Photos)
	if err != nil {
		return err
	}
	query := `INSERT INTO unit_categories (id, property_id, name, description, rent_amount, quantity_available, photos, video_url, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = r.db.ExecContext(ctx, query, cat.ID, cat.PropertyID, cat.Name, cat.Description, cat.RentAmount, cat.QuantityAvailable, photos, cat.VideoURL, cat.CreatedAt)
	return err
}

func (r *PostgresRepo) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.UnitCategory, error) {
	cat := &domain.UnitCategory{}
	var photos []byte
	query := `SELECT id, property_id, name, description, rent_amount, quantity_available, photos, video_url, created_at FROM unit_categories WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&cat.ID, &cat.PropertyID, &cat.Name, &cat.Description, &cat.RentAmount, &cat.QuantityAvailable, &photos, &cat.VideoURL, &cat.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(photos) > 0 {
		if err := json.Unmarshal(photos, &cat.Photos); err != nil {
			return nil, err
		}
	}
	return cat, nil
}

func (r *PostgresRepo) GetCategoriesByPropertyID(ctx context.Context, propertyID uuid.UUID) ([]domain.UnitCategory, error) {
	query := `SELECT id, property_id, name, description, rent_amount, quantity_available, photos, video_url, created_at FROM unit_categories WHERE property_id = $1 ORDER BY rent_amount ASC`
	rows, err := r.db.QueryContext(ctx, query, propertyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.UnitCategory
	for rows.Next() {
		var cat domain.UnitCategory
		var photos []byte
		if err := rows.Scan(&cat.ID, &cat.PropertyID, &cat.Name, &cat.Description, &cat.RentAmount, &cat.QuantityAvailable, &photos, &cat.VideoURL, &cat.CreatedAt); err != nil {
			return nil, err
		}
		if len(photos) > 0 {
			if err := json.Unmarshal(photos, &cat.Photos); err != nil {
				return nil, err
			}
		}
		list = append(list, cat)
	}
	return list, nil
}

func (r *PostgresRepo) UpdateCategory(ctx context.Context, cat *domain.UnitCategory) error {
	photos, err := json.Marshal(cat.Photos)
	if err != nil {
		return err
	}
	query := `UPDATE unit_categories SET name = $1, description = $2, rent_amount = $3, quantity_available = $4, photos = $5, video_url = $6 WHERE id = $7`
	_, err = r.db.ExecContext(ctx, query, cat.Name, cat.Description, cat.RentAmount, cat.QuantityAvailable, photos, cat.VideoURL, cat.ID)
	return err
}

func (r *PostgresRepo) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM unit_categories WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

func (r *PostgresRepo) UpdateCategoryQuantity(ctx context.Context, id uuid.UUID, delta int) error {
	query := `UPDATE unit_categories SET quantity_available = GREATEST(quantity_available + $1, 0) WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, delta, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

// Business Rule 3: Contact info — gated behind verified landlord + property has available categories
func (r *PostgresRepo) GetUnitContactDetails(ctx context.Context, unitID uuid.UUID) (*domain.ContactInfoResponse, error) {
	query := `
		SELECT 
			c.id, p.name, c.name, lp.verification_status,
			COALESCE(usr.phone, ''), usr.email, lp.id, lp.user_id, COALESCE(lp.full_name, ''),
			lp.national_id_number, lp.id_document_url, lp.is_caretaker, lp.authorized_by_landlord_id,
			lp.verification_status, lp.created_at
		FROM unit_categories c
		JOIN properties p ON c.property_id = p.id
		JOIN landlord_profiles lp ON p.landlord_id = lp.id
		JOIN users usr ON lp.user_id = usr.id
		WHERE c.id = $1`

	var resp domain.ContactInfoResponse
	var landlordStatus string

	err := r.db.QueryRowContext(ctx, query, unitID).Scan(
		&resp.UnitID, &resp.PropertyName, &resp.UnitLabel, &landlordStatus,
		&resp.LandlordPhone, &resp.LandlordEmail, &resp.LandlordProfile.ID, &resp.LandlordProfile.UserID,
		&resp.LandlordProfile.FullName, &resp.LandlordProfile.NationalIDNumber, &resp.LandlordProfile.IDDocumentURL,
		&resp.LandlordProfile.IsCaretaker, &resp.LandlordProfile.AuthorizedByLandlordID,
		&resp.LandlordProfile.VerificationStatus, &resp.LandlordProfile.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUnitNotFound
	}
	if err != nil {
		return nil, err
	}

	// Business Rule 3 Enforcement
	if landlordStatus != domain.StatusVerified {
		return nil, domain.ErrContactNotAvailable
	}

	return &resp, nil
}

func (r *PostgresRepo) CreatePropertyReport(ctx context.Context, rep *domain.PropertyReport) error {
	query := `INSERT INTO property_reports (id, property_id, reported_by, reason, details, resolved, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, rep.ID, rep.PropertyID, rep.ReportedBy, rep.Reason, rep.Details, rep.Resolved, rep.CreatedAt)
	return err
}

func (r *PostgresRepo) GetPropertyReports(ctx context.Context, resolved *bool) ([]domain.PropertyReport, error) {
	query := `
		SELECT r.id, r.property_id, r.reported_by, r.reason, r.details, r.resolved, r.created_at,
		       COALESCE(p.name, ''),
		       COALESCE(lp.full_name, ''),
		       COALESCE(lp.id, '00000000-0000-0000-0000-000000000000'),
		       COALESCE(u.phone, '')
		FROM property_reports r
		JOIN properties p ON p.id = r.property_id
		LEFT JOIN landlord_profiles lp ON lp.id = p.landlord_id
		LEFT JOIN tenant_profiles tp ON tp.id = r.reported_by
		LEFT JOIN users u ON u.id = tp.user_id`
	args := []interface{}{}
	if resolved != nil {
		query += " WHERE r.resolved = $1"
		args = append(args, *resolved)
	}
	query += " ORDER BY r.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PropertyReport
	for rows.Next() {
		var rep domain.PropertyReport
		if err := rows.Scan(
			&rep.ID, &rep.PropertyID, &rep.ReportedBy, &rep.Reason, &rep.Details, &rep.Resolved, &rep.CreatedAt,
			&rep.PropertyName, &rep.LandlordName, &rep.LandlordID, &rep.TenantPhone,
		); err != nil {
			return nil, err
		}
		list = append(list, rep)
	}
	return list, nil
}

func (r *PostgresRepo) GetPropertyReportByID(ctx context.Context, id uuid.UUID) (*domain.PropertyReport, error) {
	query := `
		SELECT r.id, r.property_id, r.reported_by, r.reason, r.details, r.resolved, r.created_at,
		       COALESCE(p.name, ''),
		       COALESCE(lp.full_name, ''),
		       COALESCE(lp.id, '00000000-0000-0000-0000-000000000000'),
		       COALESCE(u.phone, '')
		FROM property_reports r
		JOIN properties p ON p.id = r.property_id
		LEFT JOIN landlord_profiles lp ON lp.id = p.landlord_id
		LEFT JOIN tenant_profiles tp ON tp.id = r.reported_by
		LEFT JOIN users u ON u.id = tp.user_id
		WHERE r.id = $1`

	var rep domain.PropertyReport
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rep.ID, &rep.PropertyID, &rep.ReportedBy, &rep.Reason, &rep.Details, &rep.Resolved, &rep.CreatedAt,
		&rep.PropertyName, &rep.LandlordName, &rep.LandlordID, &rep.TenantPhone,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReportNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rep, nil
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

func (r *PostgresRepo) GetCustomers(ctx context.Context) ([]domain.CustomerView, error) {
	query := `SELECT u.id, u.email, u.phone, COALESCE(tp.full_name, ''), tp.location, u.created_at FROM users u JOIN tenant_profiles tp ON u.id = tp.user_id WHERE u.role = 'tenant' ORDER BY u.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.CustomerView
	for rows.Next() {
		var c domain.CustomerView
		if err := rows.Scan(&c.ID, &c.Email, &c.Phone, &c.FullName, &c.Location, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *PostgresRepo) GetAllLandlordProfiles(ctx context.Context) ([]domain.PropertyManagerView, error) {
	query := `
		SELECT lp.id, lp.user_id, u.email, u.phone, COALESCE(lp.full_name, ''), lp.page_name, lp.national_id_number,
		       lp.verification_status, lp.verified_at, lp.revoked_at, lp.revoke_reason, lp.created_at
		FROM landlord_profiles lp
		JOIN users u ON lp.user_id = u.id
		ORDER BY lp.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PropertyManagerView
	for rows.Next() {
		var a domain.PropertyManagerView
		if err := rows.Scan(&a.ID, &a.UserID, &a.Email, &a.Phone, &a.FullName, &a.PageName, &a.NationalIDNumber,
			&a.VerificationStatus, &a.VerifiedAt, &a.RevokedAt, &a.RevokeReason, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *PostgresRepo) GetPropertyManagerDetailByID(ctx context.Context, id uuid.UUID) (*domain.PropertyManagerDetail, error) {
	d := &domain.PropertyManagerDetail{}
	query := `
		SELECT lp.id, lp.user_id, lp.full_name, lp.page_name, lp.national_id_number, lp.id_document_url,
		       lp.is_caretaker, lp.authorized_by_landlord_id, lp.verification_status, lp.verified_by,
		       lp.verified_at, lp.revoked_at, lp.revoke_reason, lp.created_at,
		       u.email, u.phone
		FROM landlord_profiles lp
		JOIN users u ON lp.user_id = u.id
		WHERE lp.id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.FullName, &d.PageName, &d.NationalIDNumber, &d.IDDocumentURL,
		&d.IsCaretaker, &d.AuthorizedByLandlordID, &d.VerificationStatus, &d.VerifiedBy,
		&d.VerifiedAt, &d.RevokedAt, &d.RevokeReason, &d.CreatedAt,
		&d.Email, &d.Phone,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrLandlordNotFound
	}
	return d, err
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
