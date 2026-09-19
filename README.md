# ad-service

Turborepo monorepo for an advertisement delivery service.

## Apps

- [apps/backend](apps/backend) — Go API
- [apps/web](apps/web) — Next.js
- [apps/mobile](apps/mobile) — Expo

## Setup

```bash
pnpm install
```

Run the stack (Postgres, API, web):

```bash
docker compose up --build
```

Or Postgres only, then every app on the host:

```bash
docker compose up -d postgres
pnpm dev
```

App-specific setup, APIs, and commands live in each app README.
