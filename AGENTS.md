# GoMon

Website uptime monitor deployed as a Cloudflare Worker. Written in Go, compiled to WASM via TinyGo, backed by D1.

## Build system

- **Not a standard Go project.** All `.go` files carry `//go:build js && wasm` — they only compile under TinyGo targeting WASM. Standard `go build` will produce empty binaries.
- Entrypoint: `main.go`. Router + cron wiring lives there.
- Five internal packages under `src/`: `handlers` (includes both API handlers and `ui.go` with the web dashboard), `models`, `storage`, `monitoring` (aliased as `workers` in go.mod imports).
- Storage is D1 via `sql.Open("d1", bindingName)` using `syumai/workers/cloudflare/d1`.

## API routes

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/` | `ServeUI` | Web dashboard |
| GET/POST/PUT/DELETE | `/api/websites` | `WebsiteHandler` | Website CRUD |
| GET | `/api/websites/badge` | `GetShieldsIoBadge` | Shields.io badge JSON |
| GET | `/api/webhook-deliveries` | `ListWebhookDeliveries` | Webhook delivery queue |
| GET | `/health` | inline | Health check |

## Commands

| Command | What it does |
|---|---|
| `npm run build` | `workers-assets-gen` then `tinygo build -o ./build/app.wasm -target wasm -no-debug ./...` |
| `npm start` / `npm run dev` | `wrangler dev --test-scheduled` |
| `npm run deploy` | `wrangler deploy` |
| Trigger cron locally | `curl "http://127.0.0.1:8787/__scheduled"` |
| Apply migrations | `npx wrangler d1 migrations apply gomon --local` |

## Architecture notes

- Cron runs every minute (`"crons": ["* * * * *"]`). Each tick checks all websites due (query: `last_checked_at = 0 OR (now - last_checked_at) >= frequency`). Checks run in parallel goroutines within a single cron invocation.
- Webhook delivery uses a retry queue with exponential backoff stored in D1. Per-website and global config in `wrangler.jsonc` `vars`.
- **No tests exist** in the repo.
- No lint/typecheck/formatter commands configured.

## Deployment

- CI/CD: GitHub Actions on push to `main`. Deploys to custom domain `gomon.rowentey.xyz`.
- PRs from `main` branches deploy as preview via `wrangler versions upload`.
- Secrets: `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN` set in GitHub Actions secrets.

## Environment & config

- Env vars loaded from `wrangler.jsonc` `vars` at runtime via `cloudflare.Getenv()`, NOT from `.env` at build time.
- `MONITOR_TIMEOUT_SEC` controls HTTP request timeout per website check (default 3s).
- `.env` / `.dev.vars*` are gitignored — only `.env.example` checked in.
- Migration files in `migrations/` — sequential SQL files.
