# Feedline

Self-hosted social feed. Anyone can read posts. Sign in to publish. Ads are placed by age, country, and platform.

Go API, Next.js, Expo, Postgres. The app is called Stream.

Signed-in members can activate an advertiser workspace at `/advertiser` to upload creative, launch targeted campaigns, and review campaign impressions and clicks. See the [backend guide](apps/backend/README.md#advertiser-workflow) for API and S3/R2 storage configuration.

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
