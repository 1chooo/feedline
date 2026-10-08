package service

import (
	"context"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type AnalyticsStore interface {
	AnalyticsForAdvertiser(ctx context.Context, advertiserID int64, start, end time.Time) (*model.AnalyticsSummary, error)
}

type AnalyticsService struct {
	store AnalyticsStore
	now   func() time.Time
}

func NewAnalyticsService(store AnalyticsStore) *AnalyticsService {
	return &AnalyticsService{store: store, now: time.Now}
}

func (s *AnalyticsService) WithClock(now func() time.Time) *AnalyticsService {
	s.now = now
	return s
}

func (s *AnalyticsService) Summary(ctx context.Context, advertiserID int64, from, to string) (*model.AnalyticsSummary, error) {
	start, end, err := model.ValidateAnalyticsRange(from, to, s.now())
	if err != nil {
		return nil, err
	}
	summary, err := s.store.AnalyticsForAdvertiser(ctx, advertiserID, start, end)
	if err != nil {
		return nil, err
	}
	summary.From = start.Format(time.DateOnly)
	summary.To = end.AddDate(0, 0, -1).Format(time.DateOnly)
	if summary.Impressions > 0 {
		summary.CTR = float64(summary.Clicks) / float64(summary.Impressions)
	}
	if summary.Daily == nil {
		summary.Daily = []model.AnalyticsDaily{}
	}
	return summary, nil
}
