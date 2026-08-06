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

type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) GetVerifications(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	list, err := h.adminService.GetPendingVerifications(r.Context(), status)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch verifications"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *AdminHandler) ApproveLandlord(w http.ResponseWriter, r *http.Request) {
	landlordID, err := uuid.Parse(chi.URLParam(r, "landlord_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid landlord_id"}`, http.StatusBadRequest)
		return
	}

	adminID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	err = h.adminService.ApproveLandlord(r.Context(), adminID, landlordID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "landlord profile approved successfully"})
}

func (h *AdminHandler) RevokeLandlord(w http.ResponseWriter, r *http.Request) {
	landlordID, err := uuid.Parse(chi.URLParam(r, "landlord_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid landlord_id"}`, http.StatusBadRequest)
		return
	}

	var req domain.RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Reason == "" {
		http.Error(w, `{"error":"revoke reason is required"}`, http.StatusBadRequest)
		return
	}

	adminID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	err = h.adminService.RevokeLandlord(r.Context(), adminID, landlordID, req.Reason)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "landlord profile revoked successfully"})
}

func (h *AdminHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	var resolvedPtr *bool
	if resStr := r.URL.Query().Get("resolved"); resStr != "" {
		resVal, _ := strconv.ParseBool(resStr)
		resolvedPtr = &resVal
	}

	reports, err := h.adminService.GetReports(r.Context(), resolvedPtr)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch reports"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func (h *AdminHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, `{"error":"invalid report id"}`, http.StatusBadRequest)
		return
	}

	adminID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	if err := h.adminService.ResolveReport(r.Context(), adminID, id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "report resolved successfully"})
}

func (h *AdminHandler) GetCustomers(w http.ResponseWriter, r *http.Request) {
	customers, err := h.adminService.GetCustomers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to fetch customers"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customers)
}

func (h *AdminHandler) GetCustomerProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "customer_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid customer_id"}`, http.StatusBadRequest)
		return
	}

	profile, err := h.adminService.GetCustomerProfile(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "customer not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *AdminHandler) GetAllPropertyManagers(w http.ResponseWriter, r *http.Request) {
	managers, err := h.adminService.GetAllPropertyManagers(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to fetch property managers"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(managers)
}

func (h *AdminHandler) GetPropertyManagerProperties(w http.ResponseWriter, r *http.Request) {
	landlordID, err := uuid.Parse(chi.URLParam(r, "landlord_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid landlord_id"}`, http.StatusBadRequest)
		return
	}

	properties, err := h.adminService.GetPropertyManagerProperties(r.Context(), landlordID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch property manager properties"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(properties)
}

func (h *AdminHandler) GetPropertyManagerProfile(w http.ResponseWriter, r *http.Request) {
	landlordID, err := uuid.Parse(chi.URLParam(r, "landlord_id"))
	if err != nil {
		http.Error(w, `{"error":"invalid landlord_id"}`, http.StatusBadRequest)
		return
	}

	profile, err := h.adminService.GetPropertyManagerProfile(r.Context(), landlordID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "property manager not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *AdminHandler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	logs, err := h.adminService.GetAuditLogs(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to fetch audit log"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
