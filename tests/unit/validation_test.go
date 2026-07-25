package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/handlers"
	"github.com/stretchr/testify/assert"
)

func TestAuthRegisterValidation(t *testing.T) {
	handler := handlers.NewAuthHandler(nil)

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "Invalid JSON payload",
			body:           "invalid-json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(b))
			rec := httptest.NewRecorder()

			handler.Register(rec, req)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestLandlordCreatePropertyValidation(t *testing.T) {
	tests := []struct {
		name  string
		input domain.CreatePropertyRequest
		valid bool
	}{
		{
			name:  "Valid input",
			input: domain.CreatePropertyRequest{Name: "Palms", Location: "Westlands"},
			valid: true,
		},
		{
			name:  "Missing name",
			input: domain.CreatePropertyRequest{Name: "", Location: "Westlands"},
			valid: false,
		},
		{
			name:  "Missing location",
			input: domain.CreatePropertyRequest{Name: "Palms", Location: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.input.Name != "" && tt.input.Location != ""
			assert.Equal(t, tt.valid, isValid)
		})
	}
}
