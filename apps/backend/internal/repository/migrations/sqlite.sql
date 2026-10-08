-- SQLite's INTEGER PRIMARY KEY supplies generated IDs. All timestamps use
-- fixed-width UTC text; declared TIMESTAMP columns scan back into time.Time.
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name TEXT NOT NULL,
  bio TEXT NOT NULL DEFAULT '',
  age INTEGER,
  gender TEXT,
  country TEXT,
  role TEXT NOT NULL DEFAULT 'member',
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);

CREATE TABLE IF NOT EXISTS media (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key TEXT NOT NULL UNIQUE,
  public_url TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_media_owner_id ON media (owner_id);

CREATE TABLE IF NOT EXISTS companies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  credit_balance INTEGER NOT NULL DEFAULT 0 CHECK (credit_balance >= 0),
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_companies_owner_id ON companies (owner_id);

CREATE TABLE IF NOT EXISTS ads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  advertiser_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
  company_id INTEGER REFERENCES companies(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  image_url TEXT NOT NULL DEFAULT '',
  image_media_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
  landing_page_url TEXT NOT NULL DEFAULT '',
  bid REAL NOT NULL DEFAULT 0,
  daily_budget INTEGER,
  credit_budget INTEGER,
  credit_spent INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'active',
  start_at TIMESTAMP NOT NULL,
  end_at TIMESTAMP NOT NULL,
  conditions TEXT NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_ads_active ON ads (start_at, end_at);
CREATE INDEX IF NOT EXISTS idx_ads_status ON ads (status);
CREATE INDEX IF NOT EXISTS idx_ads_advertiser_id ON ads (advertiser_id);
CREATE INDEX IF NOT EXISTS idx_ads_company_id ON ads (company_id);

CREATE TABLE IF NOT EXISTS ad_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ad_id INTEGER NOT NULL REFERENCES ads(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  occurred_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ad_events_ad_id_occurred_at ON ad_events (ad_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_ad_events_occurred_at ON ad_events (occurred_at);

CREATE TABLE IF NOT EXISTS sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS user_activity (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  activity_type TEXT NOT NULL,
  activity_date TEXT NOT NULL,
  occurred_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now')),
  UNIQUE (user_id, activity_type, activity_date)
);
CREATE INDEX IF NOT EXISTS idx_user_activity_date_user_id ON user_activity (activity_date, user_id);

CREATE TABLE IF NOT EXISTS posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  image_url TEXT NOT NULL DEFAULT '',
  landing_page_url TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts (user_id);

CREATE TABLE IF NOT EXISTS credit_packages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
  currency TEXT NOT NULL DEFAULT 'USD',
  credits INTEGER NOT NULL CHECK (credits > 0),
  bonus_credits INTEGER NOT NULL DEFAULT 0 CHECK (bonus_credits >= 0),
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);

CREATE TABLE IF NOT EXISTS promotions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT UNIQUE,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  reward_type TEXT NOT NULL,
  reward_value INTEGER NOT NULL CHECK (reward_value > 0),
  starts_at TIMESTAMP,
  ends_at TIMESTAMP,
  max_redemptions INTEGER CHECK (max_redemptions > 0),
  total_redemptions INTEGER NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now')),
  CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at)
);
CREATE INDEX IF NOT EXISTS idx_promotions_active_window ON promotions (active, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS credit_purchases (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
  package_id INTEGER NOT NULL REFERENCES credit_packages(id) ON DELETE RESTRICT,
  status TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_reference TEXT NOT NULL,
  amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
  discount_cents INTEGER NOT NULL DEFAULT 0 CHECK (discount_cents >= 0),
  currency TEXT NOT NULL,
  credits INTEGER NOT NULL CHECK (credits > 0),
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_credit_purchases_company_created_at ON credit_purchases (company_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_purchases_provider_reference ON credit_purchases (provider, provider_reference);

CREATE TABLE IF NOT EXISTS promotion_redemptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  promotion_id INTEGER NOT NULL REFERENCES promotions(id) ON DELETE RESTRICT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
  purchase_id INTEGER REFERENCES credit_purchases(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now')),
  UNIQUE (promotion_id, company_id)
);

CREATE TABLE IF NOT EXISTS credit_transactions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE RESTRICT,
  type TEXT NOT NULL,
  delta_credits INTEGER NOT NULL,
  balance_after INTEGER NOT NULL CHECK (balance_after >= 0),
  reference TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_by_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f000000+00:00', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_credit_transactions_company_created_at ON credit_transactions (company_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_company_reference ON credit_transactions (company_id, reference) WHERE reference <> '';
