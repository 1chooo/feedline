package httpdelivery

import (
	"encoding/json"
	"net/http"
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

func (h *Handler) activateAdvertiser(w http.ResponseWriter, r *http.Request) {
	user, err := h.social.ActivateAdvertiser(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
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

	ad, err := h.svc.CreateAdForAdvertiser(r.Context(), user.ID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ad)
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
