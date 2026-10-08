package service

import (
	"context"
	"testing"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type memoryAnalyticsStore struct {
	start time.Time
	end   time.Time
}

func (m *memoryAnalyticsStore) AnalyticsForAdvertiser(_ context.Context, _ int64, start, end time.Time) (*model.AnalyticsSummary, error) {
	m.start = start
	m.end = end
	return &model.AnalyticsSummary{Impressions: 200, Clicks: 7}, nil
}

func TestAnalyticsSummaryUsesDefaultRangeAndCTR(t *testing.T) {
	t.Parallel()

	store := &memoryAnalyticsStore{}
	svc := NewAnalyticsService(store).WithClock(func() time.Time {
		return time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	})

	summary, err := svc.Summary(context.Background(), 3, "", "")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.From != "2026-09-09" || summary.To != "2026-10-08" {
		t.Fatalf("unexpected range: %+v", summary)
	}
	if summary.CTR != 0.035 {
		t.Fatalf("ctr = %v, want 0.035", summary.CTR)
	}
	if store.end.Sub(store.start) != 30*24*time.Hour {
		t.Fatalf("unexpected query span %v", store.end.Sub(store.start))
	}
}
