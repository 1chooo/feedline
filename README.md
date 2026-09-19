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

First boot seeds Stream users `jane@stream.local`, `kai@stream.local`, `nova@stream.local`, and `miles@stream.local` with password `password123`, plus sample posts and ads.

App-specific setup, APIs, and commands live in each app README.
