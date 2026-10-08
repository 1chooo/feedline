package httpdelivery

import (
	"encoding/json"
	"net/http"

	"github.com/1chooo/ad-service/internal/model"
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
