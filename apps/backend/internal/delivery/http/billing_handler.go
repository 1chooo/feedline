package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) pricing(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	packages, err := h.billing.Pricing(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": packages})
}

func (h *Handler) fundCampaign(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	adID, err := strconv.ParseInt(chi.URLParam(r, "adID"), 10, 64)
	if err != nil || adID < 1 {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "adID must be a positive integer")
		return
	}
	var req model.FundCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	response, err := h.billing.FundCampaign(r.Context(), user.ID, adID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.svc.CacheCampaign(response.Campaign)
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) billingOverview(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	overview, err := h.billing.Overview(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (h *Handler) renameCompany(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var req model.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	company, err := h.billing.RenameCompany(r.Context(), user.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, company)
}

func (h *Handler) purchaseCredits(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var req model.PurchaseCreditsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	response, err := h.billing.PurchaseCredits(r.Context(), user.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) redeemPromoCode(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var req model.RedeemPromoCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	transaction, promotion, err := h.billing.RedeemPromoCode(r.Context(), user.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"transaction": transaction, "promotion": promotion})
}

func (h *Handler) adjustCredits(w http.ResponseWriter, r *http.Request) {
	if h.billing == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing is not configured")
		return
	}
	admin, err := h.social.RequireAdmin(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	var req model.AdminCreditAdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	transaction, err := h.billing.AdjustCredits(r.Context(), admin.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, transaction)
}
