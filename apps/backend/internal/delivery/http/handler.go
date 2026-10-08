package httpdelivery

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/1chooo/ad-service/internal/service"
	"github.com/1chooo/ad-service/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc         *service.AdService
	social      *service.SocialService
	media       *service.MediaService
	analytics   *service.AnalyticsService
	billing     *service.BillingService
	localMedia  *storage.Local
	rateLimiter *RateLimiter
	idempotent  *IdempotencyStore
}

func NewHandler(svc *service.AdService, social *service.SocialService, media *service.MediaService, analytics *service.AnalyticsService, billing *service.BillingService, objectStorage storage.ObjectStorage) *Handler {
	localMedia, _ := objectStorage.(*storage.Local)
	return &Handler{
		svc:         svc,
		social:      social,
		media:       media,
		analytics:   analytics,
		billing:     billing,
		localMedia:  localMedia,
		rateLimiter: NewRateLimiter(100, time.Minute),
		idempotent:  NewIdempotencyStore(5 * time.Minute),
	}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", h.health)
	if h.localMedia != nil {
		r.Get("/media/*", h.serveLocalMedia)
	}
	r.MethodFunc(http.MethodGet, "/api/v1/ad", h.listAds)
	r.With(h.adminRateLimit).MethodFunc(http.MethodPost, "/api/v1/ad", h.createAdvertiserAd)
	r.With(h.adminRateLimit).MethodFunc(http.MethodPost, "/api/v1/ads", h.bulkCreateAdvertiserAds)
	r.MethodFunc(http.MethodPut, "/api/v1/ad", methodNotAllowed)
	r.MethodFunc(http.MethodPatch, "/api/v1/ad", methodNotAllowed)
	r.MethodFunc(http.MethodDelete, "/api/v1/ad", methodNotAllowed)
	r.Get("/api/v1/ads/{adID}/click", h.trackAdClick)
	r.Get("/api/v1/advertising/pricing", h.pricing)

	r.Post("/api/v1/auth/register", h.register)
	r.Post("/api/v1/auth/login", h.login)
	r.Post("/api/v1/auth/logout", h.logout)
	r.Get("/api/v1/me", h.me)
	r.Get("/api/v1/users/{username}", h.getUser)
	r.Get("/api/v1/users/{username}/posts", h.listUserPosts)
	r.Get("/api/v1/posts", h.listPosts)
	r.Post("/api/v1/posts", h.createPost)
	r.Post("/api/v1/media/images", h.uploadImage)
	r.Post("/api/v1/advertiser/activate", h.activateAdvertiser)
	r.Get("/api/v1/advertiser/ads", h.listAdvertiserAds)
	r.Get("/api/v1/advertiser/analytics", h.advertiserAnalytics)
	r.Get("/api/v1/advertiser/billing", h.billingOverview)
	r.Post("/api/v1/advertiser/company", h.renameCompany)
	r.Post("/api/v1/advertiser/credits/purchases", h.purchaseCredits)
	r.Post("/api/v1/advertiser/promo-codes/redeem", h.redeemPromoCode)
	r.With(h.adminRateLimit).Post("/api/v1/advertiser/ads", h.createAdvertiserAd)
	r.With(h.adminRateLimit).Post("/api/v1/admin/credits/adjustments", h.adjustCredits)
	return r
}

func (h *Handler) trackAdClick(w http.ResponseWriter, r *http.Request) {
	adID, err := strconv.ParseInt(chi.URLParam(r, "adID"), 10, 64)
	if err != nil || adID < 1 {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "adID must be a positive integer")
		return
	}
	destination, err := h.svc.TrackClick(r.Context(), adID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	http.Redirect(w, r, destination, http.StatusFound)
}

func (h *Handler) serveLocalMedia(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)
	if filename == "." || filename == "/" || filename == "" {
		writeError(w, http.StatusNotFound, model.ErrCodeNotFound, "media not found")
		return
	}
	http.StripPrefix("/media/", http.FileServer(http.Dir(h.localMedia.Directory()))).ServeHTTP(w, r)
}

func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request) {
	if h.media == nil {
		writeError(w, http.StatusServiceUnavailable, "MEDIA_UNAVAILABLE", "media storage is not configured")
		return
	}
	user, err := h.social.Me(r.Context(), bearerToken(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, model.MaxImageBytes+1024)
	if err := r.ParseMultipartForm(model.MaxImageBytes + 1024); err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "image must be at most 5 MB")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "image file is required")
		return
	}
	defer file.Close()

	body, err := io.ReadAll(io.LimitReader(file, model.MaxImageBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "could not read image")
		return
	}
	contentType := http.DetectContentType(body)
	if contentType == "application/octet-stream" && header.Header.Get("Content-Type") != "" {
		contentType = header.Header.Get("Content-Type")
	}
	media, err := h.media.UploadImage(r.Context(), user.ID, contentType, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

func (h *Handler) adminRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !h.rateLimiter.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (h *Handler) listAds(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	offset := model.DefaultOffset
	if raw := q.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "offset must be a non-negative integer")
			return
		}
		offset = parsed
	}

	limit := model.DefaultLimit
	if raw := q.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}

	var age *int
	if raw := q.Get("age"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, model.ErrCodeInvalidArgument, "age must be between 1 and 100")
			return
		}
		age = &parsed
	}

	var gender, country, platform *string
	if raw := strings.TrimSpace(q.Get("gender")); raw != "" {
		gender = &raw
	}
	if raw := strings.TrimSpace(q.Get("country")); raw != "" {
		country = &raw
	}
	if raw := strings.TrimSpace(q.Get("platform")); raw != "" {
		platform = &raw
	}

	query, err := model.ValidateListQuery(offset, limit, age, gender, country, platform)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp, err := h.svc.ListAds(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeServiceError(w http.ResponseWriter, err error) {
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		writeError(w, http.StatusBadRequest, validationErr.Code, validationErr.Message)
		return
	}
	var authErr *model.AuthError
	if errors.As(err, &authErr) {
		writeError(w, http.StatusUnauthorized, authErr.Code, authErr.Message)
		return
	}
	var forbiddenErr *model.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		writeError(w, http.StatusForbidden, forbiddenErr.Code, forbiddenErr.Message)
		return
	}
	var notFoundErr *model.NotFoundError
	if errors.As(err, &notFoundErr) {
		writeError(w, http.StatusNotFound, notFoundErr.Code, notFoundErr.Message)
		return
	}
	var conflictErr *model.ConflictError
	if errors.As(err, &conflictErr) {
		writeError(w, http.StatusConflict, conflictErr.Code, conflictErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected server error")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func extractIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.SplitN(fwd, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rateEntry
	limit    int
	window   time.Duration
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rateEntry),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, ok := rl.visitors[key]

	if !ok || now.After(entry.resetAt) {
		rl.visitors[key] = &rateEntry{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true
	}

	if entry.count >= rl.limit {
		return false
	}

	entry.count++
	return true
}

type IdempotencyStore struct {
	mu   sync.Mutex
	data map[string]*idempotentEntry
	ttl  time.Duration
}

type idempotentEntry struct {
	response  any
	expiresAt time.Time
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	s := &IdempotencyStore{
		data: make(map[string]*idempotentEntry),
		ttl:  ttl,
	}
	go s.cleanup()
	return s
}

func (s *IdempotencyStore) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.data[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(s.data, key)
		}
		return nil, false
	}

	return entry.response, true
}

func (s *IdempotencyStore) Set(key string, response any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	h := sha256.Sum256([]byte(fmt.Sprintf("%v", response)))
	_ = h

	s.data[key] = &idempotentEntry{
		response:  response,
		expiresAt: time.Now().Add(s.ttl),
	}
}

func (s *IdempotencyStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, entry := range s.data {
			if now.After(entry.expiresAt) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}
