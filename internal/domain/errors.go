package domain

import "errors"

var (
	ErrUserAlreadyExists      = errors.New("user with phone or email already exists")
	ErrInvalidCredentials     = errors.New("invalid phone/email or password")
	ErrUnauthorized           = errors.New("unauthorized access")
	ErrForbidden              = errors.New("access forbidden: insufficient permissions")
	ErrUserNotFound           = errors.New("user not found")
	ErrLandlordNotFound       = errors.New("landlord profile not found")
	ErrPropertyNotFound       = errors.New("property not found")
	ErrUnitNotFound           = errors.New("unit not found")
	ErrReportNotFound         = errors.New("report not found")
	ErrLandlordNotVerified    = errors.New("landlord profile is not verified")
	ErrCaretakerNotAuthorized = errors.New("caretaker must have a valid authorized_by_landlord_id")
	ErrAuthorizerNotVerified  = errors.New("authorizing landlord is not verified")
	ErrContactNotAvailable    = errors.New("contact details not available for non-vacant units or unverified landlords")
	ErrInvalidInput           = errors.New("invalid request payload")
	ErrCategoryNotFound       = errors.New("category not found")
)
