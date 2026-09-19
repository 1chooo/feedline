# ad-service

A Turborepo monorepo for an **Advertisement Delivery Service**. The Go backend exposes an admin API to create targeted ads and a public API to list matching active ads. `apps/web` (Next.js) and `apps/mobile` (Expo) are starter templates.

## Architecture

```text
Client (web / mobile)
  |  POST /api/v1/ad  (create)
  |  GET  /api/v1/ad  (list matching)
  v
HTTP handlers (chi)
  v
AdService (validation, filtering, sorting, pagination)
  v
AdRepository
  |-- PostgreSQL  (durable writes, active-ad reload)
  '-- In-memory cache  (read path for high throughput)
```

### Design choices

- **PostgreSQL via Docker** stores all ads durably (~3,000 creates/day).
- **In-memory active-ad cache** serves the public list API without hitting the DB on every request. With fewer than 1,000 concurrent active ads, in-memory filter + sort + paginate is sufficient for **10k+ RPS**.
- A **1-second background refresher** reloads active ads from PostgreSQL so newly started ads appear and expired ads are evicted without requiring a restart.

## Project layout

```text
apps/backend/cmd/server/main.go              Application entry point
apps/backend/internal/delivery/http/         HTTP handlers and routing
apps/backend/internal/model/ad.go            Data structures, validation, matching
apps/backend/internal/repository/ad_repo.go  PostgreSQL + active-ad cache
apps/backend/internal/service/ad_service.go  Business logic
apps/web/                                    Next.js App Router template
apps/mobile/                                 Expo (React Native) template
docs/SPEC.md                                 Full API specification
```

## Prerequisites

- Node.js 22+ and [pnpm](https://pnpm.io)
- Go 1.25+ (for host backend development)
- Docker and Docker Compose

## Run locally

Install workspace dependencies:

```bash
pnpm install
```

### Full stack in Docker

Starts PostgreSQL, the Go API, and the Next.js web app:

```bash
docker compose up --build
```

- Web: [http://localhost:3000](http://localhost:3000)
- API: [http://localhost:8080](http://localhost:8080)

Expo/mobile is not part of Compose. Run it on the host so it can reach a simulator, Expo Go, or a physical device:

```bash
pnpm --filter mobile dev
```

Point the mobile app at `http://localhost:8080` (or your machine IP from a device).

### Host development

Start PostgreSQL only, then run every app with Turbo:

```bash
docker compose up -d postgres
pnpm dev
```

Or start one app:

```bash
pnpm --filter backend dev
pnpm --filter web dev
pnpm --filter mobile dev
```

The backend listens on `:8080` by default. Configure with environment variables:

| Variable | Default |
|----------|---------|
| `PORT` | `8080` |
| `DATABASE_URL` | `postgres://ad:ad@localhost:5432/ad_service?sslmode=disable` |

## API examples

### Create an ad

```bash
curl -X POST -H "Content-Type: application/json" \
  "http://localhost:8080/api/v1/ad" \
  --data '{
    "title": "AD 55",
    "startAt": "2026-06-10T03:00:00.000Z",
    "endAt": "2026-06-30T16:00:00.000Z",
    "conditions": {
      "ageStart": 20,
      "ageEnd": 30,
      "country": ["TW", "JP"],
      "platform": ["android", "ios"]
    }
  }'
```

### List matching ads

```bash
curl -X GET \
  "http://localhost:8080/api/v1/ad?offset=0&limit=3&age=24&gender=F&country=TW&platform=ios"
```

Example response:

```json
{
  "items": [
    { "title": "AD 1", "endAt": "2026-06-22T01:00:00.000Z" },
    { "title": "AD 31", "endAt": "2026-06-30T12:00:00.000Z" }
  ]
}
```

### Error format

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "limit must be between 1 and 100"
  }
}
```

## Testing

```bash
pnpm test
# or
pnpm --filter backend test
```

From `apps/backend` you can still run `go test ./...`.

Unit tests cover validation, matching logic, sorting, pagination, and service behavior.

## Trade-offs and extensions

| Area | Current approach | Possible extension |
|------|------------------|--------------------|
| Reads | In-process cache | Redis shared cache across replicas |
| Writes | Single PostgreSQL | Read replicas, connection pooling at scale |
| Matching | In-memory scan | Pre-indexed segments by country/platform |
| Auth | None (per spec) | API keys or mTLS for admin routes |

See [docs/SPEC.md](docs/SPEC.md) for the full specification.
