package service

import (
	"context"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type AdminAnalyticsStore interface {
	Summary(ctx context.Context, start, end, now time.Time) (*model.AdminAnalyticsSummary, error)
}

type AdminAnalyticsService struct {
	store AdminAnalyticsStore
	now   func() time.Time
}

func NewAdminAnalyticsService(store AdminAnalyticsStore) *AdminAnalyticsService {
	return &AdminAnalyticsService{store: store, now: time.Now}
}

func (s *AdminAnalyticsService) WithClock(now func() time.Time) *AdminAnalyticsService {
	s.now = now
	return s
}

func (s *AdminAnalyticsService) Summary(ctx context.Context, from, to string) (*model.AdminAnalyticsSummary, error) {
	start, end, err := model.ValidateAnalyticsRange(from, to, s.now())
	if err != nil {
		return nil, err
	}
	summary, err := s.store.Summary(ctx, start, end, s.now().UTC())
	if err != nil {
		return nil, err
	}
	summary.From = start.Format(time.DateOnly)
	summary.To = end.AddDate(0, 0, -1).Format(time.DateOnly)
	if summary.Daily == nil {
		summary.Daily = []model.AdminAnalyticsDaily{}
	}
	return summary, nil
}

func (s *AdminAnalyticsService) MarketingSummary(ctx context.Context) (*model.MarketingSummary, error) {
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today.AddDate(0, 0, -29)
	end := today.AddDate(0, 0, 1)
	summary, err := s.store.Summary(ctx, start, end, now)
	if err != nil {
		return nil, err
	}
	return &model.MarketingSummary{
		MonthlyActiveUsers: summary.Users.MAU,
		ActiveAdvertisers:  summary.Advertising.Advertisers,
		ActiveCampaigns:    summary.Advertising.ActiveCampaigns,
		Impressions30d:     summary.Advertising.Impressions,
		Posts30d:           summary.Content.PostsCreated,
	}, nil
}
