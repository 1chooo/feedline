package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

//go:embed migrations/postgres.sql
var migrationSQL string

//go:embed migrations/sqlite.sql
var sqliteMigrationSQL string

type AdRepository struct {
	db    database.DB
	cache *ActiveAdCache
}

func NewAdRepository(db database.DB) *AdRepository {
	return &AdRepository{db: db, cache: NewActiveAdCache()}
}

func Migrate(ctx context.Context, db database.DB) error {
	query := migrationSQL
	if db.Dialect() == database.SQLite {
		query = sqliteMigrationSQL
	}
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, query); err != nil {
		return fmt.Errorf("run %s migration: %w", db.Dialect(), err)
	}
	if db.Dialect() == database.SQLite {
		// Existing SQLite files predate uploaded images in social posts. Add the
		// nullable reference without rewriting or replacing existing records.
		var present bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('posts') WHERE name = 'image_media_id')`).Scan(&present); err != nil {
			return err
		}
		if !present {
			if _, err := tx.Exec(ctx, `ALTER TABLE posts ADD COLUMN image_media_id INTEGER REFERENCES media(id) ON DELETE RESTRICT`); err != nil {
				return fmt.Errorf("add post media reference: %w", err)
			}
		}
	}
	return tx.Commit(ctx)
}

func CountUsers(ctx context.Context, db database.DB) (int, error) {
	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *AdRepository) Create(ctx context.Context, ad *model.Ad) error {
	conditionsJSON, err := json.Marshal(ad.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		INSERT INTO ads (advertiser_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, status, start_at, end_at, conditions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`, ad.AdvertiserID, ad.Title, ad.Description, ad.ImageUrl, ad.ImageMediaID, ad.LandingPageUrl, ad.Bid, ad.DailyBudget, ad.Status, ad.StartAt, ad.EndAt, conditionsJSON).Scan(&ad.ID, &ad.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert ad: %w", err)
	}

	return nil
}

func (r *AdRepository) ListActive(ctx context.Context, now time.Time) ([]model.Ad, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, advertiser_id, company_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, credit_budget, credit_spent, status, start_at, end_at, conditions, created_at
		FROM ads
		WHERE start_at < $1 AND end_at > $1 AND status = 'active'
		ORDER BY end_at ASC
	`, now)
	if err != nil {
		return nil, fmt.Errorf("query active ads: %w", err)
	}
	defer rows.Close()

	var ads []model.Ad
	for rows.Next() {
		ad, err := scanAd(rows)
		if err != nil {
			return nil, err
		}
		ads = append(ads, ad)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active ads: %w", err)
	}

	return ads, nil
}

func (r *AdRepository) ListByAdvertiser(ctx context.Context, advertiserID int64) ([]model.Ad, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, advertiser_id, company_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, credit_budget, credit_spent, status, start_at, end_at, conditions, created_at
		FROM ads
		WHERE advertiser_id = $1
		ORDER BY created_at DESC
	`, advertiserID)
	if err != nil {
		return nil, fmt.Errorf("query advertiser ads: %w", err)
	}
	defer rows.Close()

	ads := []model.Ad{}
	for rows.Next() {
		ad, err := scanAd(rows)
		if err != nil {
			return nil, err
		}
		ads = append(ads, ad)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate advertiser ads: %w", err)
	}
	return ads, nil
}

func (r *AdRepository) GetByID(ctx context.Context, id int64) (*model.Ad, error) {
	ad, err := scanAd(r.db.QueryRow(ctx, `
		SELECT id, advertiser_id, company_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, credit_budget, credit_spent, status, start_at, end_at, conditions, created_at
		FROM ads
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ad, nil
}

func (r *AdRepository) InsertAdEvents(ctx context.Context, events []model.AdEvent) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin ad event batch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Bound parameter count and statement size for either driver, retaining a
	// single atomic batch rather than a DB round trip for each event.
	for start := 0; start < len(events); start += 500 {
		end := min(start+500, len(events))
		values := make([]string, 0, end-start)
		args := make([]any, 0, 3*(end-start))
		for i, event := range events[start:end] {
			values = append(values, fmt.Sprintf("($%d,$%d,$%d)", 3*i+1, 3*i+2, 3*i+3))
			args = append(args, event.AdID, event.EventType, event.OccurredAt)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO ad_events (ad_id, event_type, occurred_at) VALUES `+strings.Join(values, ","), args...); err != nil {
			return fmt.Errorf("insert ad event batch: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *AdRepository) AnalyticsForAdvertiser(ctx context.Context, advertiserID int64, start, end time.Time) (*model.AnalyticsSummary, error) {
	summary := &model.AnalyticsSummary{Daily: []model.AnalyticsDaily{}}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE event_type = 'impression'),
			COUNT(*) FILTER (WHERE event_type = 'click')
		FROM ad_events event
		JOIN ads ad ON ad.id = event.ad_id
		WHERE ad.advertiser_id = $1 AND event.occurred_at >= $2 AND event.occurred_at < $3
	`, advertiserID, start, end).Scan(&summary.Impressions, &summary.Clicks)
	if err != nil {
		return nil, fmt.Errorf("query advertiser analytics: %w", err)
	}

	dateExpression := "CAST((event.occurred_at AT TIME ZONE 'UTC')::date AS TEXT)"
	if r.db.Dialect() == database.SQLite {
		dateExpression = "date(event.occurred_at)"
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			`+dateExpression+`,
			COUNT(*) FILTER (WHERE event.event_type = 'impression'),
			COUNT(*) FILTER (WHERE event.event_type = 'click')
		FROM ad_events event
		JOIN ads ad ON ad.id = event.ad_id
		WHERE ad.advertiser_id = $1 AND event.occurred_at >= $2 AND event.occurred_at < $3
		GROUP BY 1
		ORDER BY 1
	`, advertiserID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query advertiser analytics by day: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var daily model.AnalyticsDaily
		if err := rows.Scan(&daily.Date, &daily.Impressions, &daily.Clicks); err != nil {
			return nil, fmt.Errorf("scan advertiser analytics: %w", err)
		}
		summary.Daily = append(summary.Daily, daily)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate advertiser analytics: %w", err)
	}
	return summary, nil
}

func (r *AdRepository) RefreshCache(ctx context.Context, now time.Time) error {
	ads, err := r.ListActive(ctx, now)
	if err != nil {
		return err
	}
	r.cache.Refresh(ads)
	return nil
}

func (r *AdRepository) ActiveAds() []model.Ad {
	return r.cache.Active()
}

func (r *AdRepository) UpsertCache(ad model.Ad) {
	r.cache.Upsert(ad)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAd(row rowScanner) (model.Ad, error) {
	var ad model.Ad
	var conditionsJSON []byte

	if err := row.Scan(&ad.ID, &ad.AdvertiserID, &ad.CompanyID, &ad.Title, &ad.Description, &ad.ImageUrl, &ad.ImageMediaID, &ad.LandingPageUrl, &ad.Bid, &ad.DailyBudget, &ad.CreditBudget, &ad.CreditSpent, &ad.Status, &ad.StartAt, &ad.EndAt, &conditionsJSON, &ad.CreatedAt); err != nil {
		return model.Ad{}, fmt.Errorf("scan ad: %w", err)
	}

	if len(conditionsJSON) > 0 {
		if err := json.Unmarshal(conditionsJSON, &ad.Conditions); err != nil {
			return model.Ad{}, fmt.Errorf("unmarshal conditions: %w", err)
		}
	}

	if ad.Status == "" {
		ad.Status = model.StatusActive
	}

	return ad, nil
}

type ActiveAdCache struct {
	mu   sync.RWMutex
	byID map[int64]model.Ad
}

func NewActiveAdCache() *ActiveAdCache {
	return &ActiveAdCache{
		byID: make(map[int64]model.Ad),
	}
}

func (c *ActiveAdCache) Active() []model.Ad {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ads := make([]model.Ad, 0, len(c.byID))
	for _, ad := range c.byID {
		ads = append(ads, ad)
	}
	return ads
}

func (c *ActiveAdCache) Upsert(ad model.Ad) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byID[ad.ID] = ad
}

func (c *ActiveAdCache) Refresh(ads []model.Ad) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byID = make(map[int64]model.Ad, len(ads))
	for _, ad := range ads {
		c.byID[ad.ID] = ad
	}
}
