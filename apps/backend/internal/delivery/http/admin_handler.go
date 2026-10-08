package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) marketingSummary(w http.ResponseWriter, r *http.Request) {
	if h.admin == nil {
		writeError(w, http.StatusServiceUnavailable, "ANALYTICS_UNAVAILABLE", "marketing analytics are not configured")
		return
	}
	summary, err := h.admin.MarketingSummary(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) adminAnalytics(w http.ResponseWriter, r *http.Request) {
	if h.admin == nil {
		writeError(w, http.StatusServiceUnavailable, "ANALYTICS_UNAVAILABLE", "admin analytics are not configured")
		return
	}
	if _, err := h.social.RequireAdmin(r.Context(), bearerToken(r)); err != nil {
		writeServiceError(w, err)
		return
	}
	summary, err := h.admin.Summary(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) adminUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminOperations(w, r) {
		return
	}
	users, err := h.operations.Users(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (h *Handler) adminCompanies(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminOperations(w, r) {
		return
	}
	companies, err := h.operations.Companies(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": companies})
}

func (h *Handler) adminCampaigns(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminOperations(w, r) {
		return
	}
	campaigns, err := h.operations.Campaigns(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": campaigns})
}

func (h *Handler) updateAdminUserRole(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminOperations(w, r) {
		return
	}
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "userID must be a positive integer")
		return
	}
	var req model.UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	user, err := h.operations.SetUserRole(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) updateAdminCampaignStatus(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminOperations(w, r) {
		return
	}
	adID, err := strconv.ParseInt(chi.URLParam(r, "adID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "adID must be a positive integer")
		return
	}
	var req model.UpdateCampaignStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	campaign, err := h.operations.SetCampaignStatus(r.Context(), adID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, campaign)
}

func (h *Handler) adminPromotions(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminBilling(w, r) {
		return
	}
	promotions, err := h.billing.Promotions(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": promotions})
}

func (h *Handler) createAdminPromotion(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminBilling(w, r) {
		return
	}
	var req model.AdminPromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	promotion, err := h.billing.CreatePromotion(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, promotion)
}

func (h *Handler) updateAdminPromotion(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminBilling(w, r) {
		return
	}
	promotionID, err := strconv.ParseInt(chi.URLParam(r, "promotionID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "promotionID must be a positive integer")
		return
	}
	var req model.AdminPromotionStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	promotion, err := h.billing.SetPromotionActive(r.Context(), promotionID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, promotion)
}

func (h *Handler) requireAdminOperations(w http.ResponseWriter, r *http.Request) bool {
	if h.operations == nil {
		writeError(w, http.StatusServiceUnavailable, "ADMIN_UNAVAILABLE", "admin operations are not configured")
		return false
	}
	if _, err := h.social.RequireAdmin(r.Context(), bearerToken(r)); err != nil {
		writeServiceError(w, err)
		return false
	}
	return true
}

func (h *Handler) requireAdminBilling(w http.ResponseWriter, r *http.Request) bool {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return false
	}
	if _, err := h.social.RequireAdmin(r.Context(), bearerToken(r)); err != nil {
		writeServiceError(w, err)
		return false
	}
	return true
}
