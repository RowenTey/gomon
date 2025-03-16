# GoMon 🔭

> Stay effortlessly updated on your website's status—never miss a beat!

## 🛠 Getting Started

> [!IMPORTANT]  
> Make sure you have [`tinygo`](https://tinygo.org/getting-started/install/) installed as this project requires it to compile to **WASM** for Cloudflare Workers

1\. Install dependencies

```terminal
go mod tidy
```

2\. Create a KV `namespace`

```terminal
npx wrangler kv namespace create <YOUR-NAMESPACE>
```

3\. Update the **BINDING_NAME** and **BINDING_ID** values in the `wrangler.jsonc` file

```
"kv_namespaces": [
  {
    "binding": "<BINDING_NAME>",
    "id": "<BINDING_ID>"
  }
]
```

4\. Fill in env variables in `wrangler.jsonc` file

```
"vars": {
  "KV_NAMESPACE": "",
  "MIN_FREQUENCY": "" // in seconds
}
```

5\. Run the worker locally

```terminal
npm start
```

## 📂 Project Folder Structure

#### Top Level Directory Layout

```terminal
.
├── src/                  # go packages
├── main.go               # entrypoint
├── wrangler.jsonc        # cloudflare worker configuration
├── .gitignore
└── README.md
```

## ❗ Constraints

> [!NOTE]  
> Default monitoring frequency is set to `5 minutes` to avoid going over limit of **1000** List and Write operations under Cloudflare Free Plan.
