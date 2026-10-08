package service

import (
	"context"
	"testing"
	"time"

	"github.com/1chooo/ad-service/internal/model"
)

type mockAdminAnalyticsStore struct {
	start time.Time
	end   time.Time
	now   time.Time
}

func (m *mockAdminAnalyticsStore) Summary(_ context.Context, start, end, now time.Time) (*model.AdminAnalyticsSummary, error) {
	m.start, m.end, m.now = start, end, now
	return &model.AdminAnalyticsSummary{}, nil
}

func TestAdminAnalyticsSummaryNormalizesRange(t *testing.T) {
	store := &mockAdminAnalyticsStore{}
	svc := NewAdminAnalyticsService(store).WithClock(func() time.Time {
		return time.Date(2026, 10, 8, 13, 0, 0, 0, time.UTC)
	})

	summary, err := svc.Summary(context.Background(), "2026-10-01", "2026-10-07")
	if err != nil {
		t.Fatalf("Summary() error = %v", err)
	}
	if summary.From != "2026-10-01" || summary.To != "2026-10-07" {
		t.Fatalf("summary range = %q to %q", summary.From, summary.To)
	}
	if !store.end.Equal(time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("store end = %v", store.end)
	}
	if summary.Daily == nil {
		t.Fatal("Summary() returned nil daily series")
	}
}
