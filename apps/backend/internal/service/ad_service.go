package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type AdStore interface {
	Create(ctx context.Context, ad *model.Ad) error
	ListByAdvertiser(ctx context.Context, advertiserID int64) ([]model.Ad, error)
	GetByID(ctx context.Context, id int64) (*model.Ad, error)
	ListActive(ctx context.Context, now time.Time) ([]model.Ad, error)
	RefreshCache(ctx context.Context, now time.Time) error
	ActiveAds() []model.Ad
	UpsertCache(ad model.Ad)
}

type AdService struct {
	store     AdStore
	now       func() time.Time
	spend     sync.Map
	mu        sync.Mutex
	spendDate string
	events    EventSink
}

func NewAdService(store AdStore) *AdService {
	return &AdService{
		store:     store,
		now:       time.Now,
		spendDate: todayDate(time.Now()),
		events:    noOpEventSink{},
	}
}

func (s *AdService) WithEventSink(events EventSink) *AdService {
	if events != nil {
		s.events = events
	}
	return s
}

func (s *AdService) WithClock(now func() time.Time) *AdService {
	s.now = now
	return s
}

func (s *AdService) CreateAd(ctx context.Context, req model.CreateAdRequest) (*model.Ad, error) {
	return s.createAd(ctx, nil, req)
}

func (s *AdService) CreateAdForAdvertiser(ctx context.Context, advertiserID int64, req model.CreateAdRequest) (*model.Ad, error) {
	return s.createAd(ctx, &advertiserID, req)
}

func (s *AdService) createAd(ctx context.Context, advertiserID *int64, req model.CreateAdRequest) (*model.Ad, error) {
	title, startAt, endAt, conditions, description, imageUrl, landingPageUrl, bid, dailyBudget, status, err := model.ValidateCreateRequest(req)
	if err != nil {
		return nil, err
	}

	ad := &model.Ad{
		AdvertiserID:   advertiserID,
		Title:          title,
		Description:    description,
		ImageUrl:       imageUrl,
		ImageMediaID:   req.ImageMediaID,
		LandingPageUrl: landingPageUrl,
		Bid:            bid,
		DailyBudget:    dailyBudget,
		Status:         status,
		StartAt:        startAt,
		EndAt:          endAt,
		Conditions:     conditions,
	}

	if err := s.store.Create(ctx, ad); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	if model.IsActive(*ad, now) {
		s.store.UpsertCache(*ad)
	}

	return ad, nil
}

func (s *AdService) ListAdvertiserAds(ctx context.Context, advertiserID int64) (*model.ListAdvertiserAdsResponse, error) {
	ads, err := s.store.ListByAdvertiser(ctx, advertiserID)
	if err != nil {
		return nil, err
	}
	return &model.ListAdvertiserAdsResponse{Items: ads}, nil
}

func (s *AdService) BulkCreateAds(ctx context.Context, req model.BulkCreateAdRequest) (*model.BulkCreateAdResponse, error) {
	resp := &model.BulkCreateAdResponse{
		Ads:      make([]model.Ad, 0, len(req.Ads)),
		Failures: []model.BulkCreateFail{},
	}

	for i, adReq := range req.Ads {
		ad, err := s.CreateAd(ctx, adReq)
		if err != nil {
			resp.Failures = append(resp.Failures, model.BulkCreateFail{
				Index: i,
				Error: err.Error(),
			})
			continue
		}
		resp.Ads = append(resp.Ads, *ad)
	}

	if len(resp.Failures) == 0 {
		resp.Failures = nil
	}

	return resp, nil
}

func (s *AdService) ListAds(ctx context.Context, query model.ListAdsQuery) (*model.ListAdsResponse, error) {
	now := s.now().UTC()
	ads := s.store.ActiveAds()

	s.resetDailySpendIfNeeded(now)

	matched := make([]model.Ad, 0, len(ads))
	for _, ad := range ads {
		if !model.IsActive(ad, now) {
			continue
		}
		if !ad.Conditions.Matches(query.Profile, now) {
			continue
		}
		if !s.hasBudget(ad, now) {
			continue
		}
		matched = append(matched, ad)
	}

	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Bid != matched[j].Bid {
			return matched[i].Bid > matched[j].Bid
		}
		return matched[i].EndAt.Before(matched[j].EndAt)
	})

	if query.Offset >= len(matched) {
		return &model.ListAdsResponse{Items: []model.AdListItem{}}, nil
	}

	end := query.Offset + query.Limit
	if end > len(matched) {
		end = len(matched)
	}

	page := matched[query.Offset:end]
	for _, ad := range page {
		s.trackImpression(ad.ID, now)
		s.events.Track(model.AdEvent{AdID: ad.ID, EventType: model.AdEventImpression, OccurredAt: now})
	}

	items := make([]model.AdListItem, len(page))
	for i, ad := range page {
		items[i] = model.AdListItem{
			ID:             ad.ID,
			Title:          ad.Title,
			Description:    ad.Description,
			ImageUrl:       ad.ImageUrl,
			LandingPageUrl: ad.LandingPageUrl,
			EndAt:          ad.EndAt,
		}
	}

	return &model.ListAdsResponse{Items: items}, nil
}

func (s *AdService) TrackClick(ctx context.Context, adID int64) (string, error) {
	ad, err := s.store.GetByID(ctx, adID)
	if err != nil {
		return "", err
	}
	if ad == nil || ad.LandingPageUrl == "" {
		return "", model.NotFound("ad destination not found")
	}
	s.events.Track(model.AdEvent{AdID: ad.ID, EventType: model.AdEventClick, OccurredAt: s.now().UTC()})
	return ad.LandingPageUrl, nil
}

func (s *AdService) RefreshCache(ctx context.Context) error {
	return s.store.RefreshCache(ctx, s.now().UTC())
}

func (s *AdService) hasBudget(ad model.Ad, now time.Time) bool {
	if ad.DailyBudget == nil {
		return true
	}

	today := todayDate(now)
	key := spendKey(ad.ID, today)

	val, ok := s.spend.Load(key)
	if !ok {
		return true
	}

	spent := val.(int64)
	return spent < *ad.DailyBudget
}

func (s *AdService) trackImpression(adID int64, now time.Time) {
	today := todayDate(now)
	key := spendKey(adID, today)

	val, _ := s.spend.LoadOrStore(key, int64(0))
	s.spend.Store(key, val.(int64)+1)
}

func (s *AdService) resetDailySpendIfNeeded(now time.Time) {
	today := todayDate(now)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spendDate != today {
		s.spend = sync.Map{}
		s.spendDate = today
	}
}

func spendKey(adID int64, date string) string {
	return date + ":" + fmt.Sprintf("%d", adID)
}

func todayDate(now time.Time) string {
	return now.Format("2006-01-02")
}
