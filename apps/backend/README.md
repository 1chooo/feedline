# backend

Go advertisement delivery API. Admin create and public list of targeted ads.

## Architecture

```text
Client
  |  POST /api/v1/ad          (create ads)
  |  GET  /api/v1/ad          (list matching ads)
  |  POST /api/v1/auth/*      (register, login, logout)
  |  GET  /api/v1/me
  |  GET  /api/v1/users/:name
  |  GET/POST /api/v1/posts
  v
HTTP handlers (chi)
  v
AdService / SocialService
  v
PostgreSQL (ads, users, sessions, posts)
  + in-memory active-ad cache
```

- **PostgreSQL via Docker** stores all ads durably (~3,000 creates/day).
- **In-memory active-ad cache** serves the public list API without hitting the DB on every request. With fewer than 1,000 concurrent active ads, in-memory filter + sort + paginate is sufficient for **10k+ RPS**.
- A **1-second background refresher** reloads active ads from PostgreSQL so newly started ads appear and expired ads are evicted without requiring a restart.

## Layout

```text
cmd/server/main.go              Application entry point
internal/delivery/http/         HTTP handlers and routing
internal/model/                 Ads, users, posts, validation
internal/repository/            PostgreSQL, cache, local seed
internal/service/               Ad targeting and social/auth logic
```

On first boot, if tables are empty, the API seeds four Stream users (`jane`, `kai`, `nova`, `miles`) with password `password123`, their posts, and a few targeted ads.

## Run

From the repo root:

```bash
docker compose up -d postgres
pnpm --filter backend dev
```

Or from this directory:

```bash
go run ./cmd/server
```

Listens on `:8080` by default.

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

### Auth and posts

```bash
curl -X POST -H "Content-Type: application/json" \
  "http://localhost:8080/api/v1/auth/login" \
  --data '{"email":"jane@stream.local","password":"password123"}'

curl -X GET -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/me"

curl -X GET "http://localhost:8080/api/v1/posts"
```

Ad admin routes stay unauthenticated per the spec. Social write routes require `Authorization: Bearer <token>`.

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
pnpm --filter backend test
# or
go test ./...
```

Unit tests cover validation, matching logic, sorting, pagination, and service behavior.

## Trade-offs and extensions

| Area | Current approach | Possible extension |
|------|------------------|--------------------|
| Reads | In-process cache | Redis shared cache across replicas |
| Writes | Single PostgreSQL | Read replicas, connection pooling at scale |
| Matching | In-memory scan | Pre-indexed segments by country/platform |
| Ad admin auth | None (per spec) | API keys or mTLS for admin routes |
| Social auth | Session tokens + bcrypt | Cookie issued by the Next.js BFF |

See [docs/SPEC.md](../../docs/SPEC.md) for the full specification.
