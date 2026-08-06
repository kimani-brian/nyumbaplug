package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleAdmin    = "admin"
	RoleLandlord = "landlord"
	RoleTenant   = "tenant"

	StatusPending  = "pending"
	StatusVerified = "verified"
	StatusRevoked  = "revoked"

	UnitVacant      = "vacant"
	UnitOccupied    = "occupied"
	UnitReserved    = "reserved"
	UnitMaintenance = "maintenance"
)

type User struct {
	ID            uuid.UUID `json:"id"`
	Role          string    `json:"role"`
	Phone         *string   `json:"phone,omitempty"`
	Email         *string   `json:"email,omitempty"`
	PasswordHash  string    `json:"-"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type LandlordProfile struct {
	ID                     uuid.UUID  `json:"id"`
	UserID                 uuid.UUID  `json:"user_id"`
	FullName               string     `json:"full_name"`
	PageName               *string    `json:"page_name,omitempty"`
	NationalIDNumber       string     `json:"national_id_number"`
	IDDocumentURL          *string    `json:"id_document_url,omitempty"`
	IsCaretaker            bool       `json:"is_caretaker"`
	AuthorizedByLandlordID *uuid.UUID `json:"authorized_by_landlord_id,omitempty"`
	VerificationStatus     string     `json:"verification_status"`
	VerifiedBy             *uuid.UUID `json:"verified_by,omitempty"`
	VerifiedAt             *time.Time `json:"verified_at,omitempty"`
	RevokedAt              *time.Time `json:"revoked_at,omitempty"`
	RevokeReason           *string    `json:"revoke_reason,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}

type TenantProfile struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	FullName  string    `json:"full_name"`
	Location  *string   `json:"location,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Property struct {
	ID           uuid.UUID      `json:"id"`
	LandlordID   uuid.UUID      `json:"landlord_id"`
	Name         string         `json:"name"`
	Location     string         `json:"location"`
	County       *string        `json:"county,omitempty"`
	Address      *string        `json:"address,omitempty"`
	MapsURL      *string        `json:"maps_url,omitempty"`
	Description  *string        `json:"description,omitempty"`
	ImageURL     *string        `json:"image_url,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	Categories   []UnitCategory `json:"categories,omitempty"`
	MinRent      *float64       `json:"min_rent,omitempty"`
	TotalUnits   *int           `json:"total_units,omitempty"`
	MapCoords    *string        `json:"map_coords,omitempty"`
	LandlordName string         `json:"landlord_name,omitempty"`
}

type UnitCategory struct {
	ID                uuid.UUID `json:"id"`
	PropertyID        uuid.UUID `json:"property_id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	RentAmount        float64   `json:"rent_amount"`
	QuantityAvailable int       `json:"quantity_available"`
	Photos            []string  `json:"photos"`
	VideoURL          *string   `json:"video_url,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type PropertyReport struct {
	ID         uuid.UUID `json:"id"`
	PropertyID uuid.UUID `json:"property_id"`
	ReportedBy uuid.UUID `json:"reported_by"`
	Reason     string    `json:"reason"`
	Details    string    `json:"details"`
	Resolved   bool      `json:"resolved"`
	CreatedAt  time.Time `json:"created_at"`

	// Computed admin fields (populated by GetPropertyReports)
	PropertyName string    `json:"property_name,omitempty"`
	LandlordName string    `json:"landlord_name,omitempty"`
	LandlordID   uuid.UUID `json:"landlord_id,omitempty"`
	TenantPhone  string    `json:"tenant_phone,omitempty"`
}

type AdminAuditLog struct {
	ID         uuid.UUID `json:"id"`
	AdminID    uuid.UUID `json:"admin_id"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   uuid.UUID `json:"target_id"`
	Reason     *string   `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// DTOs & Responses

type RegisterRequest struct {
	Phone                  *string    `json:"phone,omitempty"`
	Email                  *string    `json:"email"`
	Password               string     `json:"password"`
	Role                   string     `json:"role"`
	FullName               string     `json:"full_name,omitempty"`
	PageName               string     `json:"page_name,omitempty"`
	NationalIDNumber       string     `json:"national_id_number,omitempty"`
	IDDocumentURL          *string    `json:"id_document_url,omitempty"`
	IsCaretaker            bool       `json:"is_caretaker,omitempty"`
	AuthorizedByLandlordID *uuid.UUID `json:"authorized_by_landlord_id,omitempty"`
}

type SubmitVerificationRequest struct {
	FullName         string  `json:"full_name"`
	PageName         *string `json:"page_name,omitempty"`
	Phone            *string `json:"phone,omitempty"`
	NationalIDNumber string  `json:"national_id_number"`
	IDDocumentURL    *string `json:"id_document_url,omitempty"`
}

type UpdateLandlordProfileRequest struct {
	FullName      *string `json:"full_name,omitempty"`
	PageName      *string `json:"page_name,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	IDDocumentURL *string `json:"id_document_url,omitempty"`
}

type UpdateTenantProfileRequest struct {
	FullName *string `json:"full_name,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Location *string `json:"location,omitempty"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// RegisterResult is returned when an account is created but before the email
// OTP has been verified, so no token is issued yet.
type RegisterResult struct {
	Email   string `json:"email"`
	Message string `json:"message"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type ResendOtpRequest struct {
	Email string `json:"email"`
}

type CreatePropertyRequest struct {
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	County      *string `json:"county,omitempty"`
	Address     *string `json:"address,omitempty"`
	MapsURL     *string `json:"maps_url,omitempty"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type CreateCategoryRequest struct {
	Name              string   `json:"name"`
	Description       *string  `json:"description,omitempty"`
	RentAmount        float64  `json:"rent_amount"`
	QuantityAvailable int      `json:"quantity_available"`
	Photos            []string `json:"photos,omitempty"`
	VideoURL          *string  `json:"video_url,omitempty"`
}

type UpdateCategoryRequest struct {
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	RentAmount        *float64 `json:"rent_amount,omitempty"`
	QuantityAvailable *int     `json:"quantity_available,omitempty"`
	Photos            []string `json:"photos,omitempty"`
	VideoURL          *string  `json:"video_url,omitempty"`
}

type UpdatePropertyRequest struct {
	Name        *string `json:"name,omitempty"`
	Location    *string `json:"location,omitempty"`
	County      *string `json:"county,omitempty"`
	Address     *string `json:"address,omitempty"`
	MapsURL     *string `json:"maps_url,omitempty"`
	Description *string `json:"description,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type RevokeRequest struct {
	Reason string `json:"reason"`
}

type PropertyFilter struct {
	Query   string
	County  string
	MinRent *float64
	MaxRent *float64
}

type ContactInfoResponse struct {
	UnitID          uuid.UUID       `json:"unit_id"`
	PropertyName    string          `json:"property_name"`
	UnitLabel       string          `json:"unit_label"`
	LandlordPhone   string          `json:"landlord_phone"`
	LandlordEmail   *string         `json:"landlord_email,omitempty"`
	LandlordProfile LandlordProfile `json:"landlord_profile"`
}

// Admin response types

type CustomerView struct {
	ID        uuid.UUID `json:"id"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	FullName  string    `json:"full_name"`
	Location  *string   `json:"location,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CustomerProfile struct {
	CustomerView
	Profile TenantProfile `json:"profile"`
}

type PropertyManagerView struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"user_id"`
	Email              *string    `json:"email,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	FullName           string     `json:"full_name"`
	PageName           *string    `json:"page_name,omitempty"`
	NationalIDNumber   string     `json:"national_id_number"`
	VerificationStatus string     `json:"verification_status"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
	RevokeReason       *string    `json:"revoke_reason,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// PropertyManagerDetail augments the full landlord profile with account
// contact fields (email/phone) for the admin detail view.
type PropertyManagerDetail struct {
	LandlordProfile
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}
