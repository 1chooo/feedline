package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS ads (
  id               BIGSERIAL PRIMARY KEY,
  title            TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',
  image_url        TEXT NOT NULL DEFAULT '',
  landing_page_url TEXT NOT NULL DEFAULT '',
  bid              DOUBLE PRECISION NOT NULL DEFAULT 0,
  daily_budget     BIGINT,
  status           TEXT NOT NULL DEFAULT 'active',
  start_at         TIMESTAMPTZ NOT NULL,
  end_at           TIMESTAMPTZ NOT NULL,
  conditions       JSONB NOT NULL DEFAULT '{}',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE indexname = 'idx_ads_active'
  ) THEN
    CREATE INDEX idx_ads_active ON ads (start_at, end_at);
  END IF;
END $$;

	DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_indexes WHERE indexname = 'idx_ads_status'
  ) THEN
    CREATE INDEX idx_ads_status ON ads (status);
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS users (
  id            BIGSERIAL PRIMARY KEY,
  username      TEXT NOT NULL UNIQUE,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name  TEXT NOT NULL,
  bio           TEXT NOT NULL DEFAULT '',
  age           INT,
  gender        TEXT,
  country       TEXT,
  role          TEXT NOT NULL DEFAULT 'member',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'member';

CREATE TABLE IF NOT EXISTS media (
  id           BIGSERIAL PRIMARY KEY,
  owner_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key  TEXT NOT NULL UNIQUE,
  public_url   TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes   BIGINT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_owner_id ON media (owner_id);

ALTER TABLE ads ADD COLUMN IF NOT EXISTS advertiser_id BIGINT REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE ads ADD COLUMN IF NOT EXISTS image_media_id BIGINT REFERENCES media(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_ads_advertiser_id ON ads (advertiser_id);

CREATE TABLE IF NOT EXISTS ad_events (
  id          BIGSERIAL PRIMARY KEY,
  ad_id       BIGINT NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
  event_type  TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ad_events_ad_id_occurred_at ON ad_events (ad_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_ad_events_occurred_at ON ad_events (occurred_at);

CREATE TABLE IF NOT EXISTS sessions (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS user_activity (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  activity_type TEXT NOT NULL,
  activity_date DATE NOT NULL,
  occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (user_id, activity_type, activity_date)
);

CREATE INDEX IF NOT EXISTS idx_user_activity_date_user_id ON user_activity (activity_date, user_id);

CREATE TABLE IF NOT EXISTS posts (
  id               BIGSERIAL PRIMARY KEY,
  user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title            TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',
  image_url        TEXT NOT NULL DEFAULT '',
  landing_page_url TEXT NOT NULL DEFAULT '',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts (user_id);

CREATE TABLE IF NOT EXISTS companies (
  id             BIGSERIAL PRIMARY KEY,
  owner_id       BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  name           TEXT NOT NULL,
  credit_balance BIGINT NOT NULL DEFAULT 0 CHECK (credit_balance >= 0),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_companies_owner_id ON companies (owner_id);

CREATE TABLE IF NOT EXISTS credit_packages (
  id            BIGSERIAL PRIMARY KEY,
  code          TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL,
  price_cents   BIGINT NOT NULL CHECK (price_cents >= 0),
  currency      TEXT NOT NULL DEFAULT 'USD',
  credits       BIGINT NOT NULL CHECK (credits > 0),
  bonus_credits BIGINT NOT NULL DEFAULT 0 CHECK (bonus_credits >= 0),
  active        BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS promotions (
  id               BIGSERIAL PRIMARY KEY,
  code             TEXT UNIQUE,
  name             TEXT NOT NULL,
  kind             TEXT NOT NULL,
  reward_type      TEXT NOT NULL,
  reward_value     BIGINT NOT NULL CHECK (reward_value > 0),
  starts_at        TIMESTAMPTZ,
  ends_at          TIMESTAMPTZ,
  max_redemptions  BIGINT CHECK (max_redemptions > 0),
  total_redemptions BIGINT NOT NULL DEFAULT 0,
  active           BOOLEAN NOT NULL DEFAULT TRUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_promotions_active_window ON promotions (active, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS credit_purchases (
  id                 BIGSERIAL PRIMARY KEY,
  company_id         BIGINT NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
  package_id         BIGINT NOT NULL REFERENCES credit_packages(id) ON DELETE RESTRICT,
  status             TEXT NOT NULL,
  provider           TEXT NOT NULL,
  provider_reference TEXT NOT NULL,
  amount_cents       BIGINT NOT NULL CHECK (amount_cents >= 0),
  discount_cents     BIGINT NOT NULL DEFAULT 0 CHECK (discount_cents >= 0),
  currency           TEXT NOT NULL,
  credits            BIGINT NOT NULL CHECK (credits > 0),
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_purchases_company_created_at ON credit_purchases (company_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_purchases_provider_reference ON credit_purchases (provider, provider_reference);

CREATE TABLE IF NOT EXISTS promotion_redemptions (
  id           BIGSERIAL PRIMARY KEY,
  promotion_id BIGINT NOT NULL REFERENCES promotions(id) ON DELETE RESTRICT,
  company_id   BIGINT NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
  purchase_id  BIGINT REFERENCES credit_purchases(id) ON DELETE SET NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (promotion_id, company_id)
);

CREATE TABLE IF NOT EXISTS credit_transactions (
  id            BIGSERIAL PRIMARY KEY,
  company_id    BIGINT NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
  type          TEXT NOT NULL,
  delta_credits BIGINT NOT NULL,
  balance_after BIGINT NOT NULL CHECK (balance_after >= 0),
  reference     TEXT NOT NULL DEFAULT '',
  note          TEXT NOT NULL DEFAULT '',
  created_by_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_transactions_company_created_at ON credit_transactions (company_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_company_reference ON credit_transactions (company_id, reference) WHERE reference <> '';

ALTER TABLE ads ADD COLUMN IF NOT EXISTS company_id BIGINT REFERENCES companies(id) ON DELETE SET NULL;
ALTER TABLE ads ADD COLUMN IF NOT EXISTS credit_budget BIGINT;
ALTER TABLE ads ADD COLUMN IF NOT EXISTS credit_spent BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_ads_company_id ON ads (company_id);
`

type AdRepository struct {
	pool  *pgxpool.Pool
	cache *ActiveAdCache
}

func NewAdRepository(ctx context.Context, databaseURL string) (*AdRepository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if err := Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &AdRepository{
		pool:  pool,
		cache: NewActiveAdCache(),
	}, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}
	return nil
}

func CountUsers(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *AdRepository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *AdRepository) Close() {
	r.pool.Close()
}

func (r *AdRepository) Create(ctx context.Context, ad *model.Ad) error {
	conditionsJSON, err := json.Marshal(ad.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
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
	rows, err := r.pool.Query(ctx, `
		SELECT id, advertiser_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, status, start_at, end_at, conditions, created_at
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
	rows, err := r.pool.Query(ctx, `
		SELECT id, advertiser_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, status, start_at, end_at, conditions, created_at
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
	ad, err := scanAd(r.pool.QueryRow(ctx, `
		SELECT id, advertiser_id, title, description, image_url, image_media_id, landing_page_url, bid, daily_budget, status, start_at, end_at, conditions, created_at
		FROM ads
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	adIDs := make([]int64, len(events))
	eventTypes := make([]string, len(events))
	occurredAt := make([]time.Time, len(events))
	for i, event := range events {
		adIDs[i] = event.AdID
		eventTypes[i] = event.EventType
		occurredAt[i] = event.OccurredAt
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ad_events (ad_id, event_type, occurred_at)
		SELECT * FROM unnest($1::bigint[], $2::text[], $3::timestamptz[])
	`, adIDs, eventTypes, occurredAt)
	if err != nil {
		return fmt.Errorf("insert ad events: %w", err)
	}
	return nil
}

func (r *AdRepository) AnalyticsForAdvertiser(ctx context.Context, advertiserID int64, start, end time.Time) (*model.AnalyticsSummary, error) {
	summary := &model.AnalyticsSummary{Daily: []model.AnalyticsDaily{}}
	err := r.pool.QueryRow(ctx, `
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

	rows, err := r.pool.Query(ctx, `
		SELECT
			(event.occurred_at AT TIME ZONE 'UTC')::date,
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
		var date time.Time
		var daily model.AnalyticsDaily
		if err := rows.Scan(&date, &daily.Impressions, &daily.Clicks); err != nil {
			return nil, fmt.Errorf("scan advertiser analytics: %w", err)
		}
		daily.Date = date.Format(time.DateOnly)
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

	if err := row.Scan(&ad.ID, &ad.AdvertiserID, &ad.Title, &ad.Description, &ad.ImageUrl, &ad.ImageMediaID, &ad.LandingPageUrl, &ad.Bid, &ad.DailyBudget, &ad.Status, &ad.StartAt, &ad.EndAt, &conditionsJSON, &ad.CreatedAt); err != nil {
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
