package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminAnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAdminAnalyticsRepository(pool *pgxpool.Pool) *AdminAnalyticsRepository {
	return &AdminAnalyticsRepository{pool: pool}
}

func (r *AdminAnalyticsRepository) Summary(ctx context.Context, start, end, now time.Time) (*model.AdminAnalyticsSummary, error) {
	summary := &model.AdminAnalyticsSummary{Daily: []model.AdminAnalyticsDaily{}}
	now = now.UTC()
	start = start.UTC()
	end = end.UTC()

	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date >= $1::date AND activity_date < $2::date),
			(SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at < $2),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date = $3::date),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date >= ($3::date - 6) AND activity_date <= $3::date),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date >= ($3::date - 29) AND activity_date <= $3::date),
			(SELECT COUNT(*) FROM posts WHERE created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM users WHERE role = 'advertiser'),
			(SELECT COUNT(*) FROM companies),
			(SELECT COUNT(DISTINCT company_id) FROM credit_purchases WHERE status = 'completed'),
			(SELECT COUNT(*) FROM ads WHERE status = 'active' AND start_at <= $3 AND end_at > $3),
			(SELECT COUNT(*) FROM ads WHERE status = 'active' AND start_at > $3),
			(SELECT COUNT(*) FROM ads WHERE end_at <= $3 OR status = 'archived'),
			(SELECT COUNT(*) FROM ads WHERE status = 'canceled'),
			(SELECT COUNT(*) FROM ads WHERE status = 'paused'),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'impression') FROM ad_events WHERE occurred_at >= $1 AND occurred_at < $2),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'click') FROM ad_events WHERE occurred_at >= $1 AND occurred_at < $2),
			(SELECT COALESCE(SUM(amount_cents), 0) FROM credit_purchases WHERE status = 'completed' AND created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM credit_purchases WHERE status = 'completed' AND created_at >= $1 AND created_at < $2),
			(SELECT COALESCE(SUM(credits), 0) FROM credit_purchases WHERE status = 'completed' AND created_at >= $1 AND created_at < $2),
			(SELECT COALESCE(-SUM(delta_credits) FILTER (WHERE delta_credits < 0), 0) FROM credit_transactions WHERE created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM promotion_redemptions redemption JOIN promotions promotion ON promotion.id = redemption.promotion_id WHERE promotion.kind = 'coupon' AND redemption.created_at >= $1 AND redemption.created_at < $2),
			(SELECT COALESCE(SUM(delta_credits), 0) FROM credit_transactions WHERE type = 'promotion' AND created_at >= $1 AND created_at < $2)
	`, start, end, now).Scan(
		&summary.Users.Total,
		&summary.Users.ActiveInRange,
		&summary.Users.NewRegistrations,
		&summary.Users.DAU,
		&summary.Users.WAU,
		&summary.Users.MAU,
		&summary.Content.PostsCreated,
		&summary.Advertising.Advertisers,
		&summary.Advertising.Companies,
		&summary.Advertising.PayingCompanies,
		&summary.Advertising.ActiveCampaigns,
		&summary.Advertising.ScheduledCampaigns,
		&summary.Advertising.CompletedCampaigns,
		&summary.Advertising.CanceledCampaigns,
		&summary.Advertising.PausedCampaigns,
		&summary.Advertising.Impressions,
		&summary.Advertising.Clicks,
		&summary.Billing.RevenueCents,
		&summary.Billing.Purchases,
		&summary.Billing.CreditsPurchased,
		&summary.Billing.CreditsUsed,
		&summary.Billing.CouponRedemptions,
		&summary.Billing.PromotionalCredits,
	)
	if err != nil {
		return nil, fmt.Errorf("query admin analytics summary: %w", err)
	}
	if summary.Advertising.Impressions > 0 {
		summary.Advertising.EngagementRate = float64(summary.Advertising.Clicks) / float64(summary.Advertising.Impressions)
	}

	retention, err := r.retention(ctx, now)
	if err != nil {
		return nil, err
	}
	summary.Retention = *retention
	daily, err := r.daily(ctx, start, end)
	if err != nil {
		return nil, err
	}
	summary.Daily = daily
	return summary, nil
}

func (r *AdminAnalyticsRepository) retention(ctx context.Context, now time.Time) (*model.AdminRetentionMetrics, error) {
	metrics := &model.AdminRetentionMetrics{}
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(100.0 * COUNT(*) FILTER (WHERE (u.created_at AT TIME ZONE 'UTC')::date <= $1::date - 1 AND EXISTS (
				SELECT 1 FROM user_activity activity WHERE activity.user_id = u.id AND activity.activity_date = (u.created_at AT TIME ZONE 'UTC')::date + 1
			)) / NULLIF(COUNT(*) FILTER (WHERE (u.created_at AT TIME ZONE 'UTC')::date <= $1::date - 1), 0), 0),
			COALESCE(100.0 * COUNT(*) FILTER (WHERE (u.created_at AT TIME ZONE 'UTC')::date <= $1::date - 7 AND EXISTS (
				SELECT 1 FROM user_activity activity WHERE activity.user_id = u.id AND activity.activity_date = (u.created_at AT TIME ZONE 'UTC')::date + 7
			)) / NULLIF(COUNT(*) FILTER (WHERE (u.created_at AT TIME ZONE 'UTC')::date <= $1::date - 7), 0), 0),
			COALESCE(100.0 * COUNT(*) FILTER (WHERE (u.created_at AT TIME ZONE 'UTC')::date <= $1::date - 30 AND EXISTS (
				SELECT 1 FROM user_activity activity WHERE activity.user_id = u.id AND activity.activity_date = (u.created_at AT TIME ZONE 'UTC')::date + 30
			)) / NULLIF(COUNT(*) FILTER (WHERE (u.created_at AT TIME ZONE 'UTC')::date <= $1::date - 30), 0), 0)
		FROM users u
	`, now).Scan(&metrics.Day1Percent, &metrics.Day7Percent, &metrics.Day30Percent)
	if err != nil {
		return nil, fmt.Errorf("query retention metrics: %w", err)
	}
	return metrics, nil
}

func (r *AdminAnalyticsRepository) daily(ctx context.Context, start, end time.Time) ([]model.AdminAnalyticsDaily, error) {
	rows, err := r.pool.Query(ctx, `
		WITH days AS (
			SELECT generate_series($1::date, ($2::date - 1), INTERVAL '1 day')::date AS day
		)
		SELECT
			day,
			(SELECT COUNT(*) FROM users WHERE (created_at AT TIME ZONE 'UTC')::date = day),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date = day),
			(SELECT COUNT(*) FROM posts WHERE (created_at AT TIME ZONE 'UTC')::date = day),
			(SELECT COALESCE(SUM(amount_cents), 0) FROM credit_purchases WHERE status = 'completed' AND (created_at AT TIME ZONE 'UTC')::date = day),
			(SELECT COUNT(*) FROM credit_purchases WHERE status = 'completed' AND (created_at AT TIME ZONE 'UTC')::date = day),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'impression') FROM ad_events WHERE (occurred_at AT TIME ZONE 'UTC')::date = day),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'click') FROM ad_events WHERE (occurred_at AT TIME ZONE 'UTC')::date = day)
		FROM days
		ORDER BY day
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("query admin analytics daily series: %w", err)
	}
	defer rows.Close()

	daily := []model.AdminAnalyticsDaily{}
	for rows.Next() {
		var day time.Time
		var point model.AdminAnalyticsDaily
		if err := rows.Scan(&day, &point.NewUsers, &point.ActiveUsers, &point.PostsCreated, &point.RevenueCents, &point.Purchases, &point.Impressions, &point.Clicks); err != nil {
			return nil, fmt.Errorf("scan admin analytics daily series: %w", err)
		}
		point.Date = day.Format(time.DateOnly)
		daily = append(daily, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin analytics daily series: %w", err)
	}
	return daily, nil
}
