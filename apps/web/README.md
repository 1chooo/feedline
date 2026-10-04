# web

Next.js App Router frontend for Stream. Server components talk to the Go API.

## Run

The whole stack, with hot reload:

```bash
docker compose up --build
docker compose exec backend go run ./cmd/seed
```

Open [http://localhost:3000](http://localhost:3000) signed out to read the mock feed. Sign in to publish a real post. Wipe the database with `docker compose down -v`, then start and seed again.

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
