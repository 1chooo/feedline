package httpdelivery

import "net/http"

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
