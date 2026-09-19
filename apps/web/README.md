# web

Next.js App Router template for the ad-service frontend.

## Run

From the repo root:

```bash
pnpm --filter web dev
```

Or from this directory:

```bash
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000).

The Compose stack also builds this app. From the repo root:

```bash
docker compose up --build
```

`NEXT_PUBLIC_API_URL` defaults to `http://localhost:8080` in Compose. See [apps/backend](../backend) for the API.
