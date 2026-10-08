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
