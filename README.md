# Feedline

Self-hosted social feed. Anyone can read posts. Sign in to publish. Ads are placed by age, country, and platform.

Go API, Next.js, Expo, PostgreSQL or SQLite. The app is called Stream.

Signed-in members can activate an advertiser workspace at `/advertiser` to upload creative, buy and redeem advertising credits, fund targeted campaigns, and review campaign impressions and clicks. Internal staff use `/admin` for live platform and advertising analytics, roles, campaign controls, company credit adjustments, and promotions. The public [advertising services page](/advertise) is available at `/advertise`. See the [backend guide](apps/backend/README.md#advertiser-workflow) for API and storage configuration.

Members can publish photos directly from the composer. Advertiser and staff
reports support date filters and accessible daily data tables. The
[product review and engineering brief](docs/PRODUCT_REVIEW.md) records the
priorities, acceptance criteria, completed workflow checks, and next features.

## Apps

- [apps/backend](apps/backend) — Go API
- [apps/web](apps/web) — Next.js
- [apps/mobile](apps/mobile) — Expo

## Setup

```bash
pnpm install
```

Run the stack (Postgres, API with hot reload, web with hot reload):

```bash
docker compose up --build
docker compose exec backend go run ./cmd/seed
```

Open http://localhost:3000 signed out to read the mock feed. Sign in as `admin@stream.local` for administrative reporting, `jane@stream.local`, `kai@stream.local`, or `nova@stream.local` for advertiser data, or `miles@stream.local` as a regular member (password `password123` for every development fixture).

Wipe the database and start over:

```bash
docker compose down -v
```

Or Postgres only, then every app on the host:

```bash
docker compose up -d postgres
pnpm dev
pnpm --filter backend db:seed
```

App-specific setup, APIs, and commands live in each app README.

For local development with a file database and local image storage, use the
standalone SQLite stack:

```bash
docker compose -f docker-compose.sqlite.yml up --build
docker compose -f docker-compose.sqlite.yml exec backend go run ./cmd/seed
```

SQLite data and uploads persist in `apps/backend/data/`, which is excluded from
Git. PostgreSQL remains the default provider. See the
[database configuration guide](apps/backend/README.md#database-providers) for
running either provider directly on the host.
