# GoMon 🔭

> Stay effortlessly updated on your website's status—never miss a beat!

## 🛠 Getting Started
> Make sure you have `tinygo` installed as this project requires it to compile to WASM for Cloudflare Workers

1\. Install dependencies

```terminal
go mod tidy
```

2\. Run the worker locally 

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

