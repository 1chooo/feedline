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
  |  POST /api/v1/media/images
  |  POST /api/v1/advertiser/activate
  |  GET/POST /api/v1/advertiser/ads
  |  GET  /api/v1/advertiser/analytics
  v
HTTP handlers (chi)
  v
AdService / SocialService / MediaService
  v
PostgreSQL (ads, users, sessions, posts, media, ad events)
  + local disk, S3, or Cloudflare R2 media storage
  + in-memory active-ad cache
```

- **PostgreSQL via Docker** stores all ads durably (~3,000 creates/day).
- **In-memory active-ad cache** serves the public list API without hitting the DB on every request. With fewer than 1,000 concurrent active ads, in-memory filter + sort + paginate is sufficient for **10k+ RPS**.
- A **1-second background refresher** reloads active ads from PostgreSQL so newly started ads appear and expired ads are evicted without requiring a restart.
- **Advertiser writes** are session-authenticated and owned by the advertiser who created them. A signed-in member explicitly activates advertiser access before creating campaigns.
- **Media metadata** (owner, object key, URL, MIME type, byte size) is stored in PostgreSQL. Image bytes are stored in local disk for development or an S3-compatible object store for production.
- **Analytics events** retain only ad ID, event type, and timestamp. A bounded in-process buffer keeps writes off the public ad-delivery path; advertiser reports are eventually consistent by up to the flush interval.

## Layout

```text
cmd/server/main.go              Application entry point
internal/delivery/http/         HTTP handlers and routing
internal/model/                 Ads, users, posts, validation
internal/repository/            PostgreSQL, cache, local seed
internal/service/               Ad targeting and social/auth logic
```

Startup creates tables. Dev data is a separate command: two administrators (`admin`, `ops`), three advertisers (`jane`, `kai`, `nova`), and regular members—all with password `password123`. It includes companies, credit packages, completed purchases, coupon and event promotions, ledger entries, campaigns, delivery events, posts, and daily activity cohorts. Seed records use stable identities and are safe to rerun without truncating local data. The command refuses `APP_ENV=production` unless `ALLOW_PRODUCTION_SEED=true` is explicitly set.

## Run

From the repo root, with hot reload:

```bash
docker compose up --build
docker compose exec backend go run ./cmd/seed
```

Postgres only, then the API on the host:

```bash
docker compose up -d postgres
pnpm --filter backend dev
pnpm --filter backend db:seed
```

Or from this directory, with Postgres already on `localhost:5432`:

```bash
go run ./cmd/server
go run ./cmd/seed
```

Listens on `:8080` by default.

| Variable | Default |
|----------|---------|
| `PORT` | `8080` |
| `DATABASE_URL` | `postgres://ad:ad@localhost:5432/ad_service?sslmode=disable` |
| `STORAGE_DRIVER` | `local` (`local`, `s3`, or `r2`) |
| `MEDIA_LOCAL_DIR` | `./data/media` when using local storage |
| `MEDIA_PUBLIC_BASE_URL` | `http://localhost:8080/media`; public origin for media URLs |
| `S3_BUCKET` | Required for `s3` or `r2` storage |
| `S3_REGION` | `auto`; use the AWS region for S3 or `auto` for R2 |
| `S3_ENDPOINT` | Optional S3-compatible endpoint; required for most R2 setups |
| `S3_ACCESS_KEY_ID` | Required for `s3` or `r2` storage |
| `S3_SECRET_ACCESS_KEY` | Required for `s3` or `r2` storage |
| `PAYMENTS_DRIVER` | `manual` in development only; production requires a payment provider integration |
| `APP_ENV` | `development`; the seed command blocks `production` by default |

## API examples

### Advertiser workflow

Ad creation is authenticated. A user first signs in, then activates the advertiser capability. The web app manages the bearer token in an HTTP-only session cookie; direct clients can use the token returned from login.

```bash
curl -X POST -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/advertiser/activate"
```

Upload a creative image. The API accepts JPEG, PNG, WebP, and GIF files up to 5 MB and returns its durable metadata. The response `id` is the media reference to use when creating an ad.

```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -F "image=@creative.png" \
  "http://localhost:8080/api/v1/media/images"
```

Create a campaign owned by the current advertiser:

```bash
curl -X POST \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  "http://localhost:8080/api/v1/advertiser/ads" \
  --data '{
    "title": "AD 55",
    "imageMediaId": 12,
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

The compatibility write routes `POST /api/v1/ad` and `POST /api/v1/ads` now enforce the same advertiser authorization and ownership rules. A single write still accepts `Idempotency-Key`; keys are scoped to the advertiser account.

List the current advertiser's ads and its default 30-day results:

```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/advertiser/ads"

curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/advertiser/analytics?from=2026-10-01&to=2026-10-31"
```

The analytics response reports impressions, clicks, click-through rate, and a daily breakdown. Date ranges are inclusive and limited to 90 days. Public delivery responses now include an ad `id`; send people through `GET /api/v1/ads/{adID}/click` to record the click and redirect safely to the campaign landing page.

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

Social writes, media uploads, and advertiser routes require `Authorization: Bearer <token>`. Only an activated advertiser may create or list owned campaigns, view results, or use the ad write compatibility routes.

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
| Media uploads | Server-mediated object write | Presigned uploads for larger creative files |
| Media providers | Local disk, AWS S3, Cloudflare R2 | Provider-specific lifecycle and image transformation |
| Analytics writes | In-process buffered event batches | Durable queue and warehouse for multi-replica delivery |
| Social auth | Session tokens + bcrypt | Cookie issued by the Next.js BFF |

See [docs/SPEC.md](../../docs/SPEC.md) for the full specification.

## Object storage configuration

Development defaults to local storage at `apps/backend/data/media`. The API serves those objects from `/media/*`; the database only retains their metadata and public URL.

For AWS S3, configure a public bucket or CDN origin and set:

```bash
STORAGE_DRIVER=s3
MEDIA_PUBLIC_BASE_URL=https://media.example.com
S3_BUCKET=stream-media
S3_REGION=us-west-2
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
```

For Cloudflare R2, use the S3-compatible endpoint and public custom domain:

```bash
STORAGE_DRIVER=r2
MEDIA_PUBLIC_BASE_URL=https://media.example.com
S3_BUCKET=stream-media
S3_REGION=auto
S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
```

The application does not change bucket ACLs. Configure public read access through an S3 bucket policy, CloudFront, or an R2 custom domain before setting `MEDIA_PUBLIC_BASE_URL`.
