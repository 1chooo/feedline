package model

import (
	"time"
)

const (
	AdEventImpression = "impression"
	AdEventClick      = "click"
)

type AdEvent struct {
	AdID       int64
	EventType  string
	OccurredAt time.Time
}

type AnalyticsDaily struct {
	Date        string `json:"date"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
}

type AnalyticsSummary struct {
	From        string           `json:"from"`
	To          string           `json:"to"`
	Impressions int64            `json:"impressions"`
	Clicks      int64            `json:"clicks"`
	CTR         float64          `json:"ctr"`
	Daily       []AnalyticsDaily `json:"daily"`
}

func ValidateAnalyticsRange(from, to string, now time.Time) (time.Time, time.Time, error) {
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if to == "" {
		to = today.Format(time.DateOnly)
	}
	end, err := time.Parse(time.DateOnly, to)
	if err != nil {
		return time.Time{}, time.Time{}, invalid("to must be a date in YYYY-MM-DD format")
	}
	if from == "" {
		from = end.AddDate(0, 0, -29).Format(time.DateOnly)
	}
	start, err := time.Parse(time.DateOnly, from)
	if err != nil {
		return time.Time{}, time.Time{}, invalid("from must be a date in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, invalid("to must be on or after from")
	}
	if end.Sub(start) > 89*24*time.Hour {
		return time.Time{}, time.Time{}, invalid("analytics range must be at most 90 days")
	}
	return start, end.AddDate(0, 0, 1), nil
}
