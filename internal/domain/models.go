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
	ID           uuid.UUID `json:"id"`
	Role         string    `json:"role"`
	Phone        string    `json:"phone"`
	Email        *string   `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type LandlordProfile struct {
	ID                     uuid.UUID  `json:"id"`
	UserID                 uuid.UUID  `json:"user_id"`
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
	CreatedAt time.Time `json:"created_at"`
}

type Property struct {
	ID          uuid.UUID `json:"id"`
	LandlordID  uuid.UUID `json:"landlord_id"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	Address     *string   `json:"address,omitempty"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Unit struct {
	ID         uuid.UUID `json:"id"`
	PropertyID uuid.UUID `json:"property_id"`
	UnitLabel  string    `json:"unit_label"`
	Bedrooms   int       `json:"bedrooms"`
	UnitType   string    `json:"unit_type"`
	RentAmount float64   `json:"rent_amount"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type PropertyReport struct {
	ID         uuid.UUID `json:"id"`
	PropertyID uuid.UUID `json:"property_id"`
	ReportedBy uuid.UUID `json:"reported_by"`
	Reason     string    `json:"reason"`
	Resolved   bool      `json:"resolved"`
	CreatedAt  time.Time `json:"created_at"`
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
	Phone                  string     `json:"phone"`
	Email                  *string    `json:"email"`
	Password               string     `json:"password"`
	Role                   string     `json:"role"`
	NationalIDNumber       string     `json:"national_id_number,omitempty"`
	IDDocumentURL          *string    `json:"id_document_url,omitempty"`
	IsCaretaker            bool       `json:"is_caretaker,omitempty"`
	AuthorizedByLandlordID *uuid.UUID `json:"authorized_by_landlord_id,omitempty"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"` // Phone or Email
	Password   string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreatePropertyRequest struct {
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	Address     *string `json:"address"`
	Description *string `json:"description"`
}

type CreateUnitRequest struct {
	UnitLabel  string  `json:"unit_label"`
	Bedrooms   int     `json:"bedrooms"`
	UnitType   string  `json:"unit_type"`
	RentAmount float64 `json:"rent_amount"`
}

type UpdateUnitRequest struct {
	UnitLabel  *string  `json:"unit_label"`
	Bedrooms   *int     `json:"bedrooms"`
	UnitType   *string  `json:"unit_type"`
	RentAmount *float64 `json:"rent_amount"`
	Status     *string  `json:"status"`
}

type RevokeRequest struct {
	Reason string `json:"reason"`
}

type PropertyFilter struct {
	Location string
	Bedrooms *int
	UnitType string
}

type ContactInfoResponse struct {
	UnitID          uuid.UUID       `json:"unit_id"`
	PropertyName    string          `json:"property_name"`
	UnitLabel       string          `json:"unit_label"`
	LandlordPhone   string          `json:"landlord_phone"`
	LandlordEmail   *string         `json:"landlord_email,omitempty"`
	LandlordProfile LandlordProfile `json:"landlord_profile"`
}
