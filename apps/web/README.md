# web

Next.js App Router frontend for Stream. Server components talk to the Go API.

## Run

Postgres and the API need to be up first:

```bash
docker compose up -d postgres
pnpm --filter backend dev
pnpm --filter web dev
```

Or from this directory, after the API is running:

```bash
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000).

`API_URL` defaults to `http://localhost:8080` on the host. Compose sets `API_URL=http://backend:8080` so the Next server can reach the API container.

Seed accounts (`password123`): `jane@stream.local`, `kai@stream.local`, `nova@stream.local`, `miles@stream.local`.
