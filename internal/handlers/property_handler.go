package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kenya-houses/backend/internal/domain"
	"github.com/kenya-houses/backend/internal/middleware"
	"github.com/kenya-houses/backend/internal/service"
)

type PropertyHandler struct {
	propertyService service.PropertyService
}

func NewPropertyHandler(propertyService service.PropertyService) *PropertyHandler {
	return &PropertyHandler{propertyService: propertyService}
}

func (h *PropertyHandler) SearchProperties(w http.ResponseWriter, r *http.Request) {
	filter := domain.PropertyFilter{
		Query:  r.URL.Query().Get("q"),
		County: r.URL.Query().Get("county"),
	}

	if minRentStr := r.URL.Query().Get("min_rent"); minRentStr != "" {
		if v, err := strconv.ParseFloat(minRentStr, 64); err == nil {
			filter.MinRent = &v
		}
	}
	if maxRentStr := r.URL.Query().Get("max_rent"); maxRentStr != "" {
		if v, err := strconv.ParseFloat(maxRentStr, 64); err == nil {
			filter.MaxRent = &v
		}
	}

	properties, err := h.propertyService.SearchProperties(r.Context(), filter)
	if err != nil {
		http.Error(w, `{"error":"failed to search properties"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(properties)
}

func (h *PropertyHandler) GetPropertyDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid property id"}`, http.StatusBadRequest)
		return
	}

	property, categories, err := h.propertyService.GetPropertyDetail(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"property not found or unverified"}`, http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"property":   property,
		"categories": categories,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *PropertyHandler) GetUnitContact(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid category id"}`, http.StatusBadRequest)
		return
	}

	contact, err := h.propertyService.GetUnitContact(r.Context(), categoryID)
	if err != nil {
		if err == domain.ErrContactNotAvailable {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "contact details only available for verified landlords"})
			return
		}
		http.Error(w, `{"error":"category not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contact)
}

func (h *PropertyHandler) ReportProperty(w http.ResponseWriter, r *http.Request) {
	propertyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid property id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		http.Error(w, `{"error":"report reason is required"}`, http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if err := h.propertyService.ReportProperty(r.Context(), userID, propertyID, req.Reason); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "report submitted"})
}
