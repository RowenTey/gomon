# GoMon

Website uptime monitor deployed as a Cloudflare Worker. Written in Go, compiled to WASM via TinyGo, backed by D1.

## Build system

- **Not a standard Go project.** All `.go` files carry `//go:build js && wasm` — they only compile under TinyGo targeting WASM. Standard `go build` will produce empty binaries. Uses TinyGo 0.42.x with Go 1.27.x. TinyGo 0.42's `wasm_exec.js` requires the `runtime.getRandomData` import, provided by `syumai/workers` v0.34.0+; keep that dependency at v0.34.0 or newer. (The `syumai/workers` module is deprecated in favor of `syumai/workers-go`, but v0.34.0 still ships `workers-assets-gen` used by the build.)
- Entrypoint: `main.go`. Router + cron wiring lives there.
- Five internal packages under `src/`: `handlers` (API handlers), `models`, `storage`, `monitoring` (aliased as `workers` in go.mod imports), `httpclient` (outbound fetch with a real `AbortSignal.timeout`).
- The web dashboard is a static asset at `public/index.html`, served by Workers Static Assets — not embedded in the Wasm binary.
- Storage is D1 via `sql.Open("d1", bindingName)` using `syumai/workers/cloudflare/d1`.

## API routes

`GET /` is served by Workers Static Assets from `public/index.html` and never invokes the worker.

| Method | Path | Handler | Description |
|---|---|---|---|
| GET/POST/PUT/DELETE | `/api/websites` | `WebsiteHandler` | Website CRUD |
| GET | `/api/websites/badge` | `GetShieldsIoBadge` | Shields.io badge JSON |
| GET | `/api/webhook-deliveries` | `ListWebhookDeliveries` | Webhook delivery queue |
| GET | `/health` | inline | Health check |

## Commands

| Command | What it does |
|---|---|
| `npm run build` | `workers-assets-gen -mode=tinygo` then `tinygo build -o ./build/app.wasm -target wasm -no-debug .` |
| `npm start` / `npm run dev` | `wrangler dev --test-scheduled` |
| `npm run deploy` | `wrangler deploy` |
| Trigger cron locally | `curl "http://127.0.0.1:8787/__scheduled"` |
| Apply migrations | `npx wrangler d1 migrations apply gomon --local` |

## Architecture notes

- Cron runs every 15 minutes (`"crons": ["*/15 * * * *"]`). Each tick checks all websites due (query: `last_checked_at = 0 OR (now - last_checked_at) >= frequency`). Checks run in parallel goroutines within a single cron invocation. `MIN_FREQUENCY` (240s) must stay below the cron interval (900s), otherwise ticks land a second or two short of `frequency` and skip every check. The cadence was raised from `*/5` because Workers Free caps cron CPU at 10ms while the worker needs ~50ms, so most 5-minute ticks were killed with `exceededCpu`.
- Webhook delivery uses a retry queue with exponential backoff stored in D1. Per-website and global config in `wrangler.jsonc` `vars`.
- Requests matching a file in `public/` are served by Workers Static Assets before the worker runs (`assets.run_worker_first` is false). All other paths (API, `/health`, 404s) fall through to the Go worker.
- **No tests exist** in the repo.
- No lint/typecheck/formatter commands configured.

## Deployment

- CI/CD: GitHub Actions on push to `main`. Deploys to custom domain `gomon.rowentey.xyz`.
- PRs from `main` branches deploy as preview via `wrangler versions upload`.
- Secrets: `CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN` set in GitHub Actions secrets.

## Environment & config

- Env vars loaded from `wrangler.jsonc` `vars` at runtime via `cloudflare.Getenv()`, NOT from `.env` at build time.
- `MONITOR_TIMEOUT_SEC` bounds the HTTP request for each website check (deployed as `2`, code default `3`). It is enforced in `src/httpclient` via the platform's `AbortSignal.timeout()`, **not** via `context.WithTimeout` — `syumai/workers`'s `fetch` client discards the request context (no AbortSignal, and its promise wait never selects on `ctx.Done()`), so a context deadline there is inert. Webhook deliveries use a fixed 15s. Do not "simplify" this back to a context timeout, and do not reintroduce an `AbortController` driven by `setTimeout`/`time.Timer`: passing a `js.Func` to `setTimeout` never returns in this runtime, and a Go-timer abort makes workerd resolve the fetch with a synthetic `200`, so timeouts would be recorded as healthy checks.
- `.env` / `.dev.vars*` are gitignored — only `.env.example` checked in.
- Migration files in `migrations/` — sequential SQL files.
