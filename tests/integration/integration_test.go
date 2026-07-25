package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/handlers"
	"github.com/kenya-houses/backend/internal/middleware"
	"github.com/kenya-houses/backend/internal/repository"
	"github.com/kenya-houses/backend/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/kenyahouses_test?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		t.Skip("Postgres DB connection unavailable for integration testing")
	}
	return db
}

func setupTestRouter(repo repository.Repository) http.Handler {
	jwtSecret := "test-secret"

	authSvc := service.NewAuthService(repo, jwtSecret)
	adminSvc := service.NewAdminService(repo)
	landlordSvc := service.NewLandlordService(repo)
	propertySvc := service.NewPropertyService(repo)

	authH := handlers.NewAuthHandler(authSvc)
	adminH := handlers.NewAdminHandler(adminSvc)
	landlordH := handlers.NewLandlordHandler(landlordSvc)
	propertyH := handlers.NewPropertyHandler(propertySvc)

	r := chi.NewRouter()

	r.Post("/api/v1/auth/register", authH.Register)
	r.Post("/api/v1/auth/login", authH.Login)
	r.Get("/api/v1/properties", propertyH.SearchProperties)
	r.Get("/api/v1/properties/{id}", propertyH.GetPropertyDetail)
	r.Get("/api/v1/units/{id}/contact", propertyH.GetUnitContact)

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(jwtSecret))

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("tenant"))
			r.Post("/api/v1/properties/{id}/report", propertyH.ReportProperty)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("landlord"))
			r.Get("/api/v1/landlord/me", landlordH.GetMe)
			r.Post("/api/v1/landlord/properties", landlordH.CreateProperty)
			r.Get("/api/v1/landlord/properties", landlordH.GetProperties)
			r.Post("/api/v1/landlord/properties/{id}/units", landlordH.CreateUnit)
			r.Patch("/api/v1/landlord/units/{id}", landlordH.UpdateUnit)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/api/v1/admin/verifications", adminH.GetVerifications)
			r.Post("/api/v1/admin/verifications/{landlord_id}/approve", adminH.ApproveLandlord)
			r.Post("/api/v1/admin/verifications/{landlord_id}/revoke", adminH.RevokeLandlord)
			r.Get("/api/v1/admin/reports", adminH.GetReports)
			r.Post("/api/v1/admin/reports/{id}/resolve", adminH.ResolveReport)
			r.Get("/api/v1/admin/audit-log", adminH.GetAuditLog)
		})
	})

	return r
}

// helpers

func registerUser(t *testing.T, router http.Handler, req domain.RegisterRequest) *domain.AuthResponse {
	t.Helper()
	b, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(b))
	router.ServeHTTP(w, r)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp domain.AuthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return &resp
}

func doRequest(t *testing.T, router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(b))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ----------------------------------------------------------------
// Test 1: Full landlord lifecycle — register, approve, create,
//         browse, revoke, verify gating (Rules 1, 2, 3, 4)
// ----------------------------------------------------------------
func TestFullLandlordWorkflowAndGating(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// 1. Register landlord (status = pending)
	landlord := registerUser(t, router, domain.RegisterRequest{
		Phone:            "+254711111111",
		Password:         "Landlord123!",
		Role:             domain.RoleLandlord,
		NationalIDNumber: "11111111",
	})

	// 2. Unverified landlord cannot create property (Rule 1)
	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Kilimani Heights", Location: "Kilimani"},
		landlord.Token)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 3. Register admin
	admin := registerUser(t, router, domain.RegisterRequest{
		Phone:    "+254722222222",
		Password: "AdminPass123!",
		Role:     domain.RoleAdmin,
	})

	// Get landlord profile ID for admin actions
	lp, err := repo.GetLandlordProfileByUserID(context.Background(), landlord.User.ID)
	require.NoError(t, err)
	landlordID := lp.ID

	// 4. Admin verifies pending landlords list
	w = doRequest(t, router, "GET", "/api/v1/admin/verifications?status=pending", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)
	var pending []domain.LandlordProfile
	json.NewDecoder(w.Body).Decode(&pending)
	assert.GreaterOrEqual(t, len(pending), 1)

	// 5. Admin approves landlord
	w = doRequest(t, router, "POST", "/api/v1/admin/verifications/"+landlordID.String()+"/approve", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. Audit log was written (Rule 4)
	logs, err := repo.GetAuditLogs(context.Background())
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "verify_landlord", logs[0].Action)
	assert.Equal(t, landlordID, logs[0].TargetID)

	// 7. Verified landlord can create property
	w = doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Kilimani Heights", Location: "Kilimani"},
		landlord.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	var prop domain.Property
	json.NewDecoder(w.Body).Decode(&prop)
	assert.Equal(t, "Kilimani Heights", prop.Name)

	// 8. Create a unit under the property
	w = doRequest(t, router, "POST", "/api/v1/landlord/properties/"+prop.ID.String()+"/units",
		domain.CreateUnitRequest{UnitLabel: "1A", Bedrooms: 2, UnitType: "2br", RentAmount: 45000},
		landlord.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	var unit domain.Unit
	json.NewDecoder(w.Body).Decode(&unit)
	assert.Equal(t, "1A", unit.UnitLabel)

	// 9. Public browse — property appears (enforcing Rule 2 via verified-only query)
	w = doRequest(t, router, "GET", "/api/v1/properties?location=Kilimani", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	var props []domain.Property
	json.NewDecoder(w.Body).Decode(&props)
	require.Len(t, props, 1)
	assert.Equal(t, prop.ID, props[0].ID)

	// 10. Unit contact — returns phone (Rule 3 happy path)
	w = doRequest(t, router, "GET", "/api/v1/units/"+unit.ID.String()+"/contact", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	var contact domain.ContactInfoResponse
	json.NewDecoder(w.Body).Decode(&contact)
	assert.Equal(t, landlord.User.Phone, contact.LandlordPhone)

	// 11. Admin revokes the landlord
	w = doRequest(t, router, "POST", "/api/v1/admin/verifications/"+landlordID.String()+"/revoke",
		domain.RevokeRequest{Reason: "Fraudulent listings"},
		admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// 12. Second audit log entry (Rule 4)
	logs, err = repo.GetAuditLogs(context.Background())
	require.NoError(t, err)
	require.Len(t, logs, 2)
	assert.Equal(t, "revoke_landlord", logs[0].Action)

	// 13. Public browse — property GONE (Rule 2 enforced at SQL layer)
	w = doRequest(t, router, "GET", "/api/v1/properties?location=Kilimani", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	json.NewDecoder(w.Body).Decode(&props)
	require.Len(t, props, 0)

	// 14. Unit contact — 403 (Rule 3: landlord no longer verified)
	w = doRequest(t, router, "GET", "/api/v1/units/"+unit.ID.String()+"/contact", nil, "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ----------------------------------------------------------------
// Test 2: Contact gating for occupied units (Rule 3)
// ----------------------------------------------------------------
func TestContactGatingOccupiedUnit(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// Set up: verified landlord with a property and unit
	landlord := registerUser(t, router, domain.RegisterRequest{
		Phone:            "+254733333333",
		Password:         "Landlord123!",
		Role:             domain.RoleLandlord,
		NationalIDNumber: "33333333",
	})
	admin := registerUser(t, router, domain.RegisterRequest{
		Phone:    "+254744444444",
		Password: "AdminPass123!",
		Role:     domain.RoleAdmin,
	})
	lp, _ := repo.GetLandlordProfileByUserID(context.Background(), landlord.User.ID)
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+lp.ID.String()+"/approve", nil, admin.Token)

	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Westlands Towers", Location: "Westlands"},
		landlord.Token)
	var prop domain.Property
	json.NewDecoder(w.Body).Decode(&prop)

	w = doRequest(t, router, "POST", "/api/v1/landlord/properties/"+prop.ID.String()+"/units",
		domain.CreateUnitRequest{UnitLabel: "B2", Bedrooms: 1, UnitType: "1br", RentAmount: 25000},
		landlord.Token)
	var unit domain.Unit
	json.NewDecoder(w.Body).Decode(&unit)
	assert.Equal(t, domain.UnitVacant, unit.Status)

	// Contact works while vacant
	w = doRequest(t, router, "GET", "/api/v1/units/"+unit.ID.String()+"/contact", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Landlord marks unit as occupied
	w = doRequest(t, router, "PATCH", "/api/v1/landlord/units/"+unit.ID.String(),
		domain.UpdateUnitRequest{Status: strPtr(domain.UnitOccupied)},
		landlord.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Contact now fails (Rule 3: non-vacant unit)
	w = doRequest(t, router, "GET", "/api/v1/units/"+unit.ID.String()+"/contact", nil, "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ----------------------------------------------------------------
// Test 3: Caretaker approval flow (Rule 5)
// ----------------------------------------------------------------
func TestCaretakerApprovalFlow(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// Create a verified landlord (will act as authorizer)
	authorizer := registerUser(t, router, domain.RegisterRequest{
		Phone:            "+254755555555",
		Password:         "Landlord123!",
		Role:             domain.RoleLandlord,
		NationalIDNumber: "55555555",
	})
	admin := registerUser(t, router, domain.RegisterRequest{
		Phone:    "+254766666666",
		Password: "AdminPass123!",
		Role:     domain.RoleAdmin,
	})
	authorizerLP, _ := repo.GetLandlordProfileByUserID(context.Background(), authorizer.User.ID)
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+authorizerLP.ID.String()+"/approve", nil, admin.Token)

	// Register a caretaker pointing to the verified landlord
	caretaker := registerUser(t, router, domain.RegisterRequest{
		Phone:                  "+254777777777",
		Password:               "Caretaker123!",
		Role:                   domain.RoleLandlord,
		NationalIDNumber:       "77777777",
		IsCaretaker:            true,
		AuthorizedByLandlordID: &authorizerLP.ID,
	})
	caretakerLP, _ := repo.GetLandlordProfileByUserID(context.Background(), caretaker.User.ID)

	// Admin approves caretaker — should succeed (Rule 5 happy path)
	w := doRequest(t, router, "POST", "/api/v1/admin/verifications/"+caretakerLP.ID.String()+"/approve", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify caretaker is now verified
	updated, _ := repo.GetLandlordProfileByID(context.Background(), caretakerLP.ID)
	assert.Equal(t, domain.StatusVerified, updated.VerificationStatus)

	// Register another caretaker pointing to an UNVERIFIED landlord
	unverified := registerUser(t, router, domain.RegisterRequest{
		Phone:            "+254788888888",
		Password:         "Landlord123!",
		Role:             domain.RoleLandlord,
		NationalIDNumber: "88888888",
	})
	unverifiedLP, _ := repo.GetLandlordProfileByUserID(context.Background(), unverified.User.ID)

	badCaretaker := registerUser(t, router, domain.RegisterRequest{
		Phone:                  "+254799999999",
		Password:               "Caretaker123!",
		Role:                   domain.RoleLandlord,
		NationalIDNumber:       "99999999",
		IsCaretaker:            true,
		AuthorizedByLandlordID: &unverifiedLP.ID,
	})
	badCaretakerLP, _ := repo.GetLandlordProfileByUserID(context.Background(), badCaretaker.User.ID)

	// Admin tries to approve — should fail (Rule 5: authorizer not verified)
	w = doRequest(t, router, "POST", "/api/v1/admin/verifications/"+badCaretakerLP.ID.String()+"/approve", nil, admin.Token)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]string
	json.NewDecoder(w.Body).Decode(&errResp)
	assert.Contains(t, errResp["error"], "authorizing landlord")
}

// ----------------------------------------------------------------
// Test 4: Tenant report flow (Rule 6)
// ----------------------------------------------------------------
func TestTenantReportProperty(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// Set up: verified landlord with a property
	landlord := registerUser(t, router, domain.RegisterRequest{
		Phone:            "+254711122233",
		Password:         "Landlord123!",
		Role:             domain.RoleLandlord,
		NationalIDNumber: "11112233",
	})
	admin := registerUser(t, router, domain.RegisterRequest{
		Phone:    "+254722233344",
		Password: "AdminPass123!",
		Role:     domain.RoleAdmin,
	})
	lp, _ := repo.GetLandlordProfileByUserID(context.Background(), landlord.User.ID)
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+lp.ID.String()+"/approve", nil, admin.Token)

	// Create property
	var prop domain.Property
	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Lavington Court", Location: "Lavington"},
		landlord.Token)
	json.NewDecoder(w.Body).Decode(&prop)

	// Register tenant
	tenant := registerUser(t, router, domain.RegisterRequest{
		Phone:    "+254733344455",
		Password: "Tenant123!",
		Role:     domain.RoleTenant,
	})

	// Tenant reports property
	w = doRequest(t, router, "POST", "/api/v1/properties/"+prop.ID.String()+"/report",
		map[string]string{"reason": "Suspicious listing — photos don't match"},
		tenant.Token)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Admin fetches unresolved reports
	w = doRequest(t, router, "GET", "/api/v1/admin/reports?resolved=false", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)
	var reports []domain.PropertyReport
	json.NewDecoder(w.Body).Decode(&reports)
	require.Len(t, reports, 1)
	assert.Equal(t, "Suspicious listing — photos don't match", reports[0].Reason)
	assert.False(t, reports[0].Resolved)

	// Admin resolves the report
	w = doRequest(t, router, "POST", "/api/v1/admin/reports/"+reports[0].ID.String()+"/resolve", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Reports query excludes resolved
	w = doRequest(t, router, "GET", "/api/v1/admin/reports?resolved=false", nil, admin.Token)
	json.NewDecoder(w.Body).Decode(&reports)
	assert.Len(t, reports, 0)
}

func strPtr(s string) *string { return &s }
