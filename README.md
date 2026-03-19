# GoMon 🔭

> Stay effortlessly updated on your website's status—never miss a beat!

## 🛠 Getting Started

> [!IMPORTANT]  
> Make sure you have [`tinygo`](https://tinygo.org/getting-started/install/) installed as this project requires it to compile to **WASM** for Cloudflare Workers

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
  "MIN_FREQUENCY": "", // in seconds
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
├── src/                  # go packages
├── main.go               # entrypoint
├── wrangler.jsonc        # cloudflare worker configuration
```

## 🔔 Webhook Notifications

GoMon can send webhook callbacks when a website status transitions to `degraded` or `down`, and also on recovery to `up`.

Set webhook fields in the create or update request body:

```json
{
	"url": "https://example.com",
	"frequency": 300,
	"webhookEnabled": true,
	"webhookUrl": "https://your-endpoint.example/webhook",
	"webhookPayloadTemplate": "{\"id\":\"{{eventId}}\",\"url\":\"{{websiteUrl}}\",\"from\":\"{{previousStatus}}\",\"to\":\"{{currentStatus}}\",\"statusCode\":{{statusCode}},\"responseTime\":{{responseTime}},\"error\":\"{{error}}\",\"timestamp\":{{timestamp}}}"
}
```

Sample request:

```bash
curl -X POST "http://127.0.0.1:8787/api/websites" \
  -H "Content-Type: application/json" \
  --data-raw '{
    "url": "https://non-existent-website213131.com",
    "frequency": 300,
    "webhookEnabled": true,
    "webhookUrl": "https://teybot.rowentey.xyz/webhook",
    "webhookPayloadTemplate": "{\"chat_id\":-1002500967655,\"message_thread_id\":8,\"title\":\"Alert for {{websiteUrl}}\",\"message\":\"from={{previousStatus}}, to={{currentStatus}}, statusCode={{statusCode}}, responseTime={{responseTime}}, error={{error}}, timestamp={{timestamp}}\"}"
  }'
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
