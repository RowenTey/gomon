# GoMon 🔭

> Stay effortlessly updated on your website's status—never miss a beat!

## 🛠 Getting Started
> [!IMPORTANT] 
> Make sure you have `tinygo` installed as this project requires it to compile to **WASM** for Cloudflare Workers

1\. Install dependencies

```terminal
go mod tidy
```

2\. Create a `KV namespace`

```terminal
npx wrangler kv namespace create <YOUR-NAMESPACE> 
```

3\. Update the **BINDING_NAME** and **BINDING_ID** values in the `wrangler.jsonc` file

4\. Run the worker locally 

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

