# web

Next.js App Router frontend for Stream. Server components talk to the Go API.

## Run

The whole stack, with hot reload:

```bash
docker compose up --build
docker compose exec backend go run ./cmd/seed
```

Open [http://localhost:3000](http://localhost:3000) signed out to read the mock feed. Sign in to publish a real post. Wipe the database with `docker compose down -v`, then start and seed again.

The frontend uses port 3000. Ensure another service is not using that port;
the repository root `.env` should contain `WEB_PORT=3000`. Recreate the web
container from the root with
`docker compose up -d --build --force-recreate --no-deps web` if necessary.
Docker uses Webpack with `WATCHPACK_POLLING` for hot reload and a separate
`.next` volume. Host development continues to use Turbopack.

The advertiser dashboard is at [http://localhost:3000/advertiser](http://localhost:3000/advertiser).
The admin dashboard is at [http://localhost:3000/admin](http://localhost:3000/admin).
Both redirect signed-out visitors to sign-in. Administration is available to
admin accounts, and its navigation link appears after signing in as an admin.
Use `admin@stream.local` / `password123` for the seeded admin, or
`jane@stream.local` / `password123` for an advertiser.

Postgres only, then the apps on the host:

```bash
docker compose up -d postgres
pnpm --filter backend dev
pnpm --filter backend db:seed
pnpm --filter web dev
```

Or from this directory, after the API is running:

```bash
pnpm dev
```

`API_URL` defaults to `http://localhost:8080` on the host. Compose sets `API_URL=http://backend:8080` so the Next server can reach the API container.

Seed accounts (`password123`): `jane@stream.local`, `kai@stream.local`, `nova@stream.local`, `miles@stream.local`. Signing out leaves the public feed in place. Signing in shows the compose box, and a new post uses the same card as the seeded ones.
