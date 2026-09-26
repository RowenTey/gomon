# GoMon 🔭

> Stay effortlessly updated on your website's status—never miss a beat!

## 🖥 Web Dashboard

GoMon ships with a built-in web dashboard served at the root path (`/`). It is a self-contained HTML/CSS/JS page served as a Workers Static Asset from `public/index.html`, so it stays out of the Wasm bundle.

**Features:**

- **Website management** — View all monitored sites with live status indicators (green = up, red = down, orange = degraded), response times, status codes, and last-checked timestamps. Add, edit, and delete websites through modal forms.
- **Webhook delivery queue** — Inspect webhook notification history: pending retries, delivered events, and failed attempts with error messages.
- **Auto-refresh** — Both tabs poll the API every 15 seconds.

## 🛠 Getting Started

> [!IMPORTANT]  
> You need **TinyGo 0.42.x** and **Go 1.27.x** installed. TinyGo 0.42 compiles the project to **WASM** for Cloudflare Workers. Its `wasm_exec.js` requires the `runtime.getRandomData` import, which `syumai/workers` provides from v0.34.0 onward — keep that dependency at v0.34.0 or newer.

1\. Install dependencies

```terminal
go mod tidy
```

2\. Create a D1 database

```terminal
npx wrangler d1 create gomon
```

3\. Update the D1 binding values in the `wrangler.jsonc` file

```
"d1_databases": [
  {
    "binding": "DB",
    "database_name": "gomon",
    "database_id": "<YOUR_D1_DATABASE_ID>"
  }
]
```

4\. Fill in env variables in `wrangler.jsonc` file

```
"vars": {
  "D1_BINDING": "DB",
  "MIN_FREQUENCY": "",      // in seconds (default 240, keep below the cron interval)
  "MONITOR_TIMEOUT_SEC": "", // HTTP timeout per check (default 3)
  "WEBHOOK_NOTIFY_ON_RECOVERY": "true",
  "WEBHOOK_MAX_ATTEMPTS": "3",
  "WEBHOOK_INITIAL_DELAY_SEC": "30",
  "WEBHOOK_MAX_DELAY_SEC": "300",
  "WEBHOOK_BACKOFF_FACTOR": "2"
}
```

5\. Run local migrations

```terminal
npx wrangler d1 migrations apply gomon --local
```

6\. Run the worker locally

```terminal
npm start
```

7\. Test cron scheduling locally

```terminal
curl "http://127.0.0.1:8787/__scheduled"
```

## 📂 Project Folder Structure

### Top Level Directory Layout

```terminal
.
├── public/               # static dashboard assets (Workers Static Assets)
├── src/                  # go packages
│   ├── handlers/         # HTTP API handlers
│   ├── models/           # data types & api contracts
│   ├── storage/          # D1 persistence layer
│   └── workers/          # monitoring & webhook logic
├── main.go               # entrypoint + router
├── wrangler.jsonc        # cloudflare worker configuration
```

## 🔔 Webhook Notifications

GoMon can send webhook callbacks when a website status transitions to `degraded` or `down`, and also on recovery to `up`.

Set webhook fields in the create or update request body:

```json
{
	"url": "https://example.com",
	"frequency": 300,
	"customHeaders": {
		"Authorization": "Bearer your-token",
		"X-Monitor-Source": "gomon"
	},
	"webhookEnabled": true,
	"webhookUrl": "https://your-endpoint.example/webhook",
	"webhookPayloadTemplate": "{\"id\":\"{{eventId}}\",\"url\":\"{{websiteUrl}}\",\"from\":\"{{previousStatus}}\",\"to\":\"{{currentStatus}}\",\"statusCode\":{{statusCode}},\"responseTime\":{{responseTime}},\"error\":\"{{error}}\",\"timestamp\":{{timestamp}}}"
}
```


Webhook retry/recovery behavior is global and configured through env vars in `wrangler.jsonc`.

Supported payload template placeholders:

- `{{eventId}}`
- `{{websiteUrl}}`
- `{{timestamp}}`
- `{{previousStatus}}`
- `{{currentStatus}}`
- `{{responseTime}}`
- `{{statusCode}}`
- `{{error}}`

Webhook payload shape:

```json
{
	"eventId": "20260318120000.000000000-a1b2c3d4",
	"websiteUrl": "https://example.com",
	"timestamp": 1710777600,
	"previousStatus": "up",
	"currentStatus": "down",
	"responseTime": 1534,
	"statusCode": 502,
	"error": ""
}
```
