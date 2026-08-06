package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/config"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/handlers"
	"github.com/kenya-houses/backend/internal/middleware"
	"github.com/kenya-houses/backend/internal/repository"
	"github.com/kenya-houses/backend/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMailer records the OTP code sent to each email so the tests can verify it.
type testMailer struct {
	mu    sync.Mutex
	codes map[string]string
}

func newTestMailer() *testMailer {
	return &testMailer{codes: map[string]string{}}
}

func (m *testMailer) SendOTP(_ context.Context, toEmail, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[toEmail] = code
	return nil
}

func (m *testMailer) Code(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.codes[email]
}

var testMail = newTestMailer()

func connectDB(t *testing.T) *sql.DB {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://brian@/kenyahouses_test?host=/var/run/postgresql&port=5434&sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil || db.Ping() != nil {
		t.Skip("Postgres DB connection unavailable for integration testing")
	}
	return db
}

func setupTestRouter(repo repository.Repository) http.Handler {
	jwtSecret := "test-secret"

	cfg := config.Config{Environment: "development", OTPExpiryMinutes: 10, OTPCooldownSeconds: 60}
	authSvc := service.NewAuthService(repo, jwtSecret, testMail, cfg)
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
	r.Post("/api/v1/auth/verify-email", authH.VerifyEmail)
	r.Post("/api/v1/auth/resend-otp", authH.ResendOtp)
	r.Get("/api/v1/properties", propertyH.SearchProperties)
	r.Get("/api/v1/properties/{id}", propertyH.GetPropertyDetail)
	r.Get("/api/v1/categories/{id}/contact", propertyH.GetUnitContact)

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(jwtSecret))

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("tenant"))
			r.Post("/api/v1/properties/{id}/report", propertyH.ReportProperty)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("landlord"))
			r.Get("/api/v1/landlord/me", landlordH.GetMe)
			r.Post("/api/v1/landlord/profile", landlordH.SubmitVerification)
			r.Put("/api/v1/landlord/profile", landlordH.UpdateProfile)
			r.Post("/api/v1/landlord/properties", landlordH.CreateProperty)
			r.Get("/api/v1/landlord/properties", landlordH.GetProperties)
			r.Post("/api/v1/landlord/properties/{id}/categories", landlordH.AddCategory)
			r.Patch("/api/v1/landlord/categories/{id}", landlordH.UpdateCategory)
			r.Post("/api/v1/landlord/categories/{id}/quantity", landlordH.AdjustQuantity)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/api/v1/admin/verifications", adminH.GetVerifications)
			r.Post("/api/v1/admin/verifications/{landlord_id}/approve", adminH.ApproveLandlord)
			r.Post("/api/v1/admin/verifications/{landlord_id}/revoke", adminH.RevokeLandlord)
			r.Get("/api/v1/admin/reports", adminH.GetReports)
			r.Post("/api/v1/admin/reports/{id}/resolve", adminH.ResolveReport)
			r.Get("/api/v1/admin/audit-log", adminH.GetAuditLog)
			r.Get("/api/v1/admin/property-managers/{landlord_id}/profile", adminH.GetPropertyManagerProfile)
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

	email := *req.Email
	code := testMail.Code(email)
	require.NotEmpty(t, code, "OTP should have been sent to %s", email)

	return verifyUser(t, router, email, code)
}

// verifyUser submits an OTP via the verify-email endpoint and returns the
// auth response (token + user) issued on success.
func verifyUser(t *testing.T, router http.Handler, email, code string) *domain.AuthResponse {
	t.Helper()
	resp := doRequest(t, router, "POST", "/api/v1/auth/verify-email",
		domain.VerifyEmailRequest{Email: email, Code: code}, "")
	require.Equal(t, http.StatusOK, resp.Code)
	var auth domain.AuthResponse
	json.NewDecoder(resp.Body).Decode(&auth)
	require.NotEmpty(t, auth.Token)
	return &auth
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

// registerLandlord: registers a landlord user and submits verification, returning
// the auth response and the (pending) landlord profile ID.
func registerLandlord(t *testing.T, router http.Handler, repo repository.Repository, email, phone, name, nationalID string) (*domain.AuthResponse, uuid.UUID) {
	t.Helper()
	auth := registerUser(t, router, domain.RegisterRequest{
		Email:            strPtr(email),
		Phone:            strPtr(phone),
		Password:         "Landlord123!",
		Role:             domain.RoleLandlord,
		FullName:         name,
		NationalIDNumber: nationalID,
	})
	w := doRequest(t, router, "POST", "/api/v1/landlord/profile",
		domain.SubmitVerificationRequest{
			FullName:         name,
			Phone:            strPtr(phone),
			NationalIDNumber: nationalID,
		},
		auth.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	var profile domain.LandlordProfile
	json.NewDecoder(w.Body).Decode(&profile)
	return auth, profile.ID
}

func registerAdmin(t *testing.T, router http.Handler, email string) *domain.AuthResponse {
	t.Helper()
	return registerUser(t, router, domain.RegisterRequest{
		Email:    strPtr(email),
		Password: "AdminPass123!",
		Role:     domain.RoleAdmin,
	})
}

// ----------------------------------------------------------------
// Test 1: Full landlord lifecycle — register, verify, approve, create,
//
//	browse, revoke, verify gating (Rules 1, 2, 3, 4)
//
// ----------------------------------------------------------------
func TestFullLandlordWorkflowAndGating(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// 1. Register landlord + submit verification (status = pending)
	landlord, landlordID := registerLandlord(t, router, repo, "landlord1@test.com", "+254711111111", "Test Landlord", "11111111")

	// 2. Unverified landlord cannot create property (Rule 1)
	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Kilimani Heights", Location: "Kilimani"},
		landlord.Token)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 3. Register admin
	admin := registerAdmin(t, router, "admin1@test.com")

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

	// 8. Create a category under the property
	w = doRequest(t, router, "POST", "/api/v1/landlord/properties/"+prop.ID.String()+"/categories",
		domain.CreateCategoryRequest{Name: "1 Bedroom", RentAmount: 45000, QuantityAvailable: 3},
		landlord.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	var cat domain.UnitCategory
	json.NewDecoder(w.Body).Decode(&cat)
	assert.Equal(t, "1 Bedroom", cat.Name)
	assert.Equal(t, 3, cat.QuantityAvailable)

	// 9. Public browse — property appears (Rule 2: verified-only query)
	w = doRequest(t, router, "GET", "/api/v1/properties?q=Kilimani", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	var props []domain.Property
	json.NewDecoder(w.Body).Decode(&props)
	require.Len(t, props, 1)
	assert.Equal(t, prop.ID, props[0].ID)

	// 10. Category contact — returns phone (Rule 3 happy path)
	w = doRequest(t, router, "GET", "/api/v1/categories/"+cat.ID.String()+"/contact", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	var contact domain.ContactInfoResponse
	json.NewDecoder(w.Body).Decode(&contact)
	assert.Equal(t, "+254711111111", contact.LandlordPhone)

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
	w = doRequest(t, router, "GET", "/api/v1/properties?q=Kilimani", nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	json.NewDecoder(w.Body).Decode(&props)
	require.Len(t, props, 0)

	// 14. Category contact — 403 (Rule 3: landlord no longer verified)
	w = doRequest(t, router, "GET", "/api/v1/categories/"+cat.ID.String()+"/contact", nil, "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ----------------------------------------------------------------
// Test 2: Quantity adjustment flow
// ----------------------------------------------------------------
func TestAdjustCategoryQuantity(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	landlord, landlordID := registerLandlord(t, router, repo, "landlord2@test.com", "+254733333333", "Quantity Tester", "33333333")
	admin := registerAdmin(t, router, "admin2@test.com")
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+landlordID.String()+"/approve", nil, admin.Token)

	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Westlands Towers", Location: "Westlands"},
		landlord.Token)
	var prop domain.Property
	json.NewDecoder(w.Body).Decode(&prop)

	w = doRequest(t, router, "POST", "/api/v1/landlord/properties/"+prop.ID.String()+"/categories",
		domain.CreateCategoryRequest{Name: "Bedsitter", RentAmount: 25000, QuantityAvailable: 3},
		landlord.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	var cat domain.UnitCategory
	json.NewDecoder(w.Body).Decode(&cat)

	// Adjust quantity by -1
	w = doRequest(t, router, "POST", "/api/v1/landlord/categories/"+cat.ID.String()+"/quantity",
		map[string]int{"delta": -1},
		landlord.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	cats, err := repo.GetCategoriesByPropertyID(context.Background(), prop.ID)
	require.NoError(t, err)
	require.Len(t, cats, 1)
	assert.Equal(t, 2, cats[0].QuantityAvailable)
}

// ----------------------------------------------------------------
// Test 3: Caretaker approval flow (Rule 5) — validated at service layer
// ----------------------------------------------------------------
func TestCaretakerApprovalFlow(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// Create a verified landlord (will act as authorizer)
	_, authorizerLPID := registerLandlord(t, router, repo, "landlord3@test.com", "+254755555555", "Authorizer", "55555555")
	admin := registerAdmin(t, router, "admin3@test.com")
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+authorizerLPID.String()+"/approve", nil, admin.Token)

	// Register an unverified landlord (authorizer that never gets approved)
	_, unverifiedLPID := registerLandlord(t, router, repo, "landlord4@test.com", "+254788888888", "Unverified", "88888888")

	// Register caretakers pointing at each authorizer
	registerCaretaker := func(email, phone, nationalID string, authorizerID uuid.UUID) *domain.LandlordProfile {
		auth := registerUser(t, router, domain.RegisterRequest{
			Email:            strPtr(email),
			Phone:            strPtr(phone),
			Password:         "Caretaker123!",
			Role:             domain.RoleLandlord,
			FullName:         "Caretaker",
			NationalIDNumber: nationalID,
		})
		profile := &domain.LandlordProfile{
			ID:                     uuid.New(),
			UserID:                 auth.User.ID,
			FullName:               "Caretaker",
			NationalIDNumber:       nationalID,
			IsCaretaker:            true,
			AuthorizedByLandlordID: &authorizerID,
			VerificationStatus:     domain.StatusPending,
			CreatedAt:              now(),
		}
		require.NoError(t, repo.CreateLandlordProfile(context.Background(), profile))
		return profile
	}

	good := registerCaretaker("caretaker1@test.com", "+254777777777", "77777777", authorizerLPID)
	bad := registerCaretaker("caretaker2@test.com", "+254799999999", "99999999", unverifiedLPID)

	// Approve caretaker authorized by verified landlord — succeeds (Rule 5 happy path)
	w := doRequest(t, router, "POST", "/api/v1/admin/verifications/"+good.ID.String()+"/approve", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	updated, _ := repo.GetLandlordProfileByID(context.Background(), good.ID)
	assert.Equal(t, domain.StatusVerified, updated.VerificationStatus)

	// Approve caretaker authorized by UNVERIFIED landlord — fails (Rule 5)
	w = doRequest(t, router, "POST", "/api/v1/admin/verifications/"+bad.ID.String()+"/approve", nil, admin.Token)
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
	landlord, landlordID := registerLandlord(t, router, repo, "landlord5@test.com", "+254711122233", "Report Target", "11112233")
	admin := registerAdmin(t, router, "admin4@test.com")
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+landlordID.String()+"/approve", nil, admin.Token)

	// Create property
	var prop domain.Property
	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Lavington Court", Location: "Lavington"},
		landlord.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	json.NewDecoder(w.Body).Decode(&prop)

	// Register tenant
	tenant := registerUser(t, router, domain.RegisterRequest{
		Email:    strPtr("tenant1@test.com"),
		Phone:    strPtr("+254733344455"),
		Password: "Tenant123!",
		Role:     domain.RoleTenant,
		FullName: "Tenant One",
	})

	// Tenant reports property with a message
	w = doRequest(t, router, "POST", "/api/v1/properties/"+prop.ID.String()+"/report",
		map[string]string{"reason": "Suspicious listing", "details": "Agent asked for M-Pesa deposit before viewing"},
		tenant.Token)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Admin fetches unresolved reports
	w = doRequest(t, router, "GET", "/api/v1/admin/reports?resolved=false", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)
	var reports []domain.PropertyReport
	json.NewDecoder(w.Body).Decode(&reports)
	require.Len(t, reports, 1)
	assert.Equal(t, "Suspicious listing", reports[0].Reason)
	assert.Equal(t, "Agent asked for M-Pesa deposit before viewing", reports[0].Details)
	assert.False(t, reports[0].Resolved)
	assert.Equal(t, "Lavington Court", reports[0].PropertyName)
	assert.Equal(t, landlordID, reports[0].LandlordID)

	// Admin resolves the report
	w = doRequest(t, router, "POST", "/api/v1/admin/reports/"+reports[0].ID.String()+"/resolve", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Reports query excludes resolved
	w = doRequest(t, router, "GET", "/api/v1/admin/reports?resolved=false", nil, admin.Token)
	json.NewDecoder(w.Body).Decode(&reports)
	assert.Len(t, reports, 0)
}

// ----------------------------------------------------------------
// Test 5: Resolving a report auto-restores a revoked property manager (Rule 6 + 2)
// ----------------------------------------------------------------
func TestResolveReportAutoRestoresPropertyManager(t *testing.T) {
	db := connectDB(t)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback()

	repo := repository.NewPostgresRepo(tx)
	router := setupTestRouter(repo)

	// Set up: verified landlord with a property
	landlord, landlordID := registerLandlord(t, router, repo, "landlord6@test.com", "+254700111222", "Restore Target", "70011122")
	admin := registerAdmin(t, router, "admin5@test.com")
	doRequest(t, router, "POST", "/api/v1/admin/verifications/"+landlordID.String()+"/approve", nil, admin.Token)

	var prop domain.Property
	w := doRequest(t, router, "POST", "/api/v1/landlord/properties",
		domain.CreatePropertyRequest{Name: "Runda Gate", Location: "Runda"},
		landlord.Token)
	require.Equal(t, http.StatusCreated, w.Code)
	json.NewDecoder(w.Body).Decode(&prop)

	// Tenant reports it
	tenant := registerUser(t, router, domain.RegisterRequest{
		Email:    strPtr("tenant2@test.com"),
		Phone:    strPtr("+254700111333"),
		Password: "Tenant123!",
		Role:     domain.RoleTenant,
		FullName: "Tenant Two",
	})
	w = doRequest(t, router, "POST", "/api/v1/properties/"+prop.ID.String()+"/report",
		map[string]string{"reason": "Scam deposit", "details": "Asked for cash before viewing"},
		tenant.Token)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Admin revokes the property manager — property disappears from search
	w = doRequest(t, router, "POST", "/api/v1/admin/verifications/"+landlordID.String()+"/revoke",
		domain.RevokeRequest{Reason: "Scam deposit report"},
		admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	w = doRequest(t, router, "GET", "/api/v1/properties?q=Runda", nil, "")
	var props []domain.Property
	json.NewDecoder(w.Body).Decode(&props)
	require.Len(t, props, 0)

	// Admin resolves the report — manager and property are auto-restored
	var reports []domain.PropertyReport
	w = doRequest(t, router, "GET", "/api/v1/admin/reports?resolved=false", nil, admin.Token)
	json.NewDecoder(w.Body).Decode(&reports)
	require.Len(t, reports, 1)

	w = doRequest(t, router, "POST", "/api/v1/admin/reports/"+reports[0].ID.String()+"/resolve", nil, admin.Token)
	assert.Equal(t, http.StatusOK, w.Code)

	// Manager is verified again
	restored, err := repo.GetLandlordProfileByID(context.Background(), landlordID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusVerified, restored.VerificationStatus)

	// Property is visible in search again
	w = doRequest(t, router, "GET", "/api/v1/properties?q=Runda", nil, "")
	json.NewDecoder(w.Body).Decode(&props)
	require.Len(t, props, 1)
	assert.Equal(t, prop.ID, props[0].ID)
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func now() time.Time { return time.Now() }
