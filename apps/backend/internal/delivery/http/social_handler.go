package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}

	resp, err := h.social.Register(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}

	resp, err := h.social.Login(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.social.Logout(r.Context(), bearerToken(r)); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.Me(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user.Public())
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.GetUser(r.Context(), chi.URLParam(r, "username"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.social.ListPosts(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) listUserPosts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.social.ListPostsByUsername(r.Context(), chi.URLParam(r, "username"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	var req model.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}

	resp, err := h.social.CreatePost(r.Context(), bearerToken(r), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	if h.media == nil {
		writeError(w, http.StatusServiceUnavailable, "MEDIA_UNAVAILABLE", "media storage is not configured")
		return
	}
	user, err := h.social.Me(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	mediaID, err := strconv.ParseInt(chi.URLParam(r, "mediaID"), 10, 64)
	if err != nil || mediaID < 1 {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "mediaID must be a positive integer")
		return
	}
	if err := h.media.DeleteImage(r.Context(), user.ID, mediaID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) activateAdvertiser(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.ActivateAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if h.billing != nil {
		if _, err := h.billing.EnsureDefaultCompany(r.Context(), user); err != nil {
			writeServiceError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, user.Public())
}

func (h *Handler) listAdvertiserAds(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp, err := h.svc.ListAdvertiserAds(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) createAdvertiserAd(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	var req model.CreateAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}
	if req.ImageMediaID != nil {
		if h.media == nil {
			writeError(w, http.StatusServiceUnavailable, "MEDIA_UNAVAILABLE", "media storage is not configured")
			return
		}
		media, err := h.media.OwnedImage(r.Context(), user.ID, *req.ImageMediaID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		req.ImageUrl = media.URL
	}

	key := ""
	if rawKey := strings.TrimSpace(r.Header.Get("Idempotency-Key")); rawKey != "" {
		key = strconv.FormatInt(user.ID, 10) + ":" + rawKey
		if cached, ok := h.idempotent.Get(key); ok {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	ad, err := h.svc.CreateAdForAdvertiser(r.Context(), user.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if key != "" {
		h.idempotent.Set(key, ad)
	}
	writeJSON(w, http.StatusCreated, ad)
}

func (h *Handler) bulkCreateAdvertiserAds(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	var req model.BulkCreateAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "request body must be valid JSON")
		return
	}

	resp := &model.BulkCreateAdResponse{Ads: make([]model.Ad, 0, len(req.Ads))}
	for index, adReq := range req.Ads {
		if adReq.ImageMediaID != nil {
			if h.media == nil {
				resp.Failures = append(resp.Failures, model.BulkCreateFail{Index: index, Error: "media storage is not configured"})
				continue
			}
			media, err := h.media.OwnedImage(r.Context(), user.ID, *adReq.ImageMediaID)
			if err != nil {
				resp.Failures = append(resp.Failures, model.BulkCreateFail{Index: index, Error: err.Error()})
				continue
			}
			adReq.ImageUrl = media.URL
		}
		ad, err := h.svc.CreateAdForAdvertiser(r.Context(), user.ID, adReq)
		if err != nil {
			resp.Failures = append(resp.Failures, model.BulkCreateFail{Index: index, Error: err.Error()})
			continue
		}
		resp.Ads = append(resp.Ads, *ad)
	}
	if len(resp.Failures) == 0 {
		resp.Failures = nil
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) advertiserAnalytics(w http.ResponseWriter, r *http.Request) {
	if h.analytics == nil {
		writeError(w, http.StatusServiceUnavailable, "ANALYTICS_UNAVAILABLE", "analytics are not configured")
		return
	}
	user, err := h.social.RequireAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	summary, err := h.analytics.Summary(r.Context(), user.ID, r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if len(header) < 8 || !strings.EqualFold(header[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}
