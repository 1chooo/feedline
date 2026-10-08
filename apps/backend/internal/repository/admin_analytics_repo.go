package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

type AdminAnalyticsRepository struct {
	db database.DB
}

func NewAdminAnalyticsRepository(db database.DB) *AdminAnalyticsRepository {
	return &AdminAnalyticsRepository{db: db}
}

func (r *AdminAnalyticsRepository) Summary(ctx context.Context, start, end, now time.Time) (*model.AdminAnalyticsSummary, error) {
	summary := &model.AdminAnalyticsSummary{Daily: []model.AdminAnalyticsDaily{}}
	now = now.UTC()
	start = start.UTC()
	end = end.UTC()

	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date >= $4 AND activity_date < $5),
			(SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at < $2),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date = $6),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date >= $7 AND activity_date <= $6),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date >= $8 AND activity_date <= $6),
			(SELECT COUNT(*) FROM posts WHERE created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM users WHERE role = 'advertiser'),
			(SELECT COUNT(*) FROM companies),
			(SELECT COUNT(DISTINCT company_id) FROM credit_purchases WHERE status = 'completed'),
			(SELECT COUNT(*) FROM ads WHERE status = 'active' AND start_at <= $3 AND end_at > $3),
			(SELECT COUNT(*) FROM ads WHERE status = 'active' AND start_at > $3),
			(SELECT COUNT(*) FROM ads WHERE status <> 'canceled' AND (end_at <= $3 OR status = 'archived')),
			(SELECT COUNT(*) FROM ads WHERE status = 'canceled'),
			(SELECT COUNT(*) FROM ads WHERE status = 'paused' AND end_at > $3),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'impression') FROM ad_events WHERE occurred_at >= $1 AND occurred_at < $2),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'click') FROM ad_events WHERE occurred_at >= $1 AND occurred_at < $2),
			(SELECT COALESCE(SUM(amount_cents), 0) FROM credit_purchases WHERE status = 'completed' AND created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM credit_purchases WHERE status = 'completed' AND created_at >= $1 AND created_at < $2),
			(SELECT COALESCE(SUM(credits), 0) FROM credit_purchases WHERE status = 'completed' AND created_at >= $1 AND created_at < $2),
			(SELECT COALESCE(-SUM(delta_credits) FILTER (WHERE delta_credits < 0), 0) FROM credit_transactions WHERE type = 'campaign_spend' AND created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*) FROM promotion_redemptions redemption JOIN promotions promotion ON promotion.id = redemption.promotion_id WHERE promotion.kind = 'coupon' AND redemption.created_at >= $1 AND redemption.created_at < $2),
			(SELECT COALESCE(SUM(delta_credits), 0) FROM credit_transactions WHERE type = 'promotion' AND created_at >= $1 AND created_at < $2)
	`, start, end, now, start.Format(time.DateOnly), end.Format(time.DateOnly), now.Format(time.DateOnly), now.AddDate(0, 0, -6).Format(time.DateOnly), now.AddDate(0, 0, -29).Format(time.DateOnly)).Scan(
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

// utcDate is used only with fixed column expressions in repository SQL.
func utcDate(db database.Executor, column string) string {
	if db.Dialect() == database.SQLite {
		return "date(" + column + ")"
	}
	return "(" + column + " AT TIME ZONE 'UTC')::date"
}

func (r *AdminAnalyticsRepository) retention(ctx context.Context, now time.Time) (*model.AdminRetentionMetrics, error) {
	metrics := &model.AdminRetentionMetrics{}
	cohort := utcDate(r.db, "u.created_at")
	clauses := make([]string, 0, 3)
	for _, days := range []int{1, 7, 30} {
		eligible := fmt.Sprintf("($1::date - %d)", days)
		returned := fmt.Sprintf("(%s + %d)", cohort, days)
		if r.db.Dialect() == database.SQLite {
			eligible = fmt.Sprintf("date($1, '-%d days')", days)
			returned = fmt.Sprintf("date(u.created_at, '+%d days')", days)
		}
		clauses = append(clauses, fmt.Sprintf(`
			COALESCE(100.0 * COUNT(*) FILTER (WHERE %s <= %s AND EXISTS (
				SELECT 1 FROM user_activity activity WHERE activity.user_id = u.id AND activity.activity_date = %s
			)) / NULLIF(COUNT(*) FILTER (WHERE %s <= %s), 0), 0)
		`, cohort, eligible, returned, cohort, eligible))
	}
	err := r.db.QueryRow(ctx, "SELECT "+strings.Join(clauses, ",")+" FROM users u", now).Scan(&metrics.Day1Percent, &metrics.Day7Percent, &metrics.Day30Percent)
	if err != nil {
		return nil, fmt.Errorf("query retention metrics: %w", err)
	}
	return metrics, nil
}

func (r *AdminAnalyticsRepository) daily(ctx context.Context, start, end time.Time) ([]model.AdminAnalyticsDaily, error) {
	days := `WITH days AS (
		SELECT generate_series($1::date, ($2::date - 1), INTERVAL '1 day')::date AS day
	)`
	if r.db.Dialect() == database.SQLite {
		days = `WITH RECURSIVE days(day) AS (
			SELECT date($1) WHERE date($1) < date($2)
			UNION ALL SELECT date(day, '+1 day') FROM days WHERE date(day, '+1 day') < date($2)
		)`
	}
	rows, err := r.db.Query(ctx, days+`
		SELECT CAST(day AS TEXT),
			(SELECT COUNT(*) FROM users WHERE `+utcDate(r.db, "created_at")+` = day),
			(SELECT COUNT(DISTINCT user_id) FROM user_activity WHERE activity_date = day),
			(SELECT COUNT(*) FROM posts WHERE `+utcDate(r.db, "created_at")+` = day),
			(SELECT COALESCE(SUM(amount_cents), 0) FROM credit_purchases WHERE status = 'completed' AND `+utcDate(r.db, "created_at")+` = day),
			(SELECT COUNT(*) FROM credit_purchases WHERE status = 'completed' AND `+utcDate(r.db, "created_at")+` = day),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'impression') FROM ad_events WHERE `+utcDate(r.db, "occurred_at")+` = day),
			(SELECT COUNT(*) FILTER (WHERE event_type = 'click') FROM ad_events WHERE `+utcDate(r.db, "occurred_at")+` = day)
		FROM days ORDER BY day
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("query admin analytics daily series: %w", err)
	}
	defer rows.Close()
	daily := []model.AdminAnalyticsDaily{}
	for rows.Next() {
		var point model.AdminAnalyticsDaily
		if err := rows.Scan(&point.Date, &point.NewUsers, &point.ActiveUsers, &point.PostsCreated, &point.RevenueCents, &point.Purchases, &point.Impressions, &point.Clicks); err != nil {
			return nil, fmt.Errorf("scan admin analytics daily series: %w", err)
		}
		daily = append(daily, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin analytics daily series: %w", err)
	}
	return daily, nil
}
