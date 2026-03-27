# web-tools

Local-first web search and reading CLI for AI agents.

Zero cost. No API keys. No third-party dependencies.

## What it does

- **web-search**: Search the web via a local SearXNG instance (aggregates Google, Bing, DuckDuckGo)
- **web-reader**: Extract readable content from URLs or convert local files (PDF, DOCX, PPTX, XLSX) to Markdown

## Quick start

### 1. Build

```bash
cd ~/projects/go/web-tools
go build -o ~/go/bin/web-tools .
```

### 2. Start SearXNG (for web-search)

```bash
cd ~/projects/go/web-tools/docker
docker compose up -d
```

Verify: `curl -s http://localhost:8888/search?q=test&format=json | head -c 200`

### 3. Install optional dependencies

```bash
# For file conversion (PDF, DOCX, PPTX, XLSX)
pip install markitdown

# For browser fallback (JS-rendered pages)
npm i -g agent-browser
```

### 4. Use

```bash
# Search
web-tools web-search "latest AI news"
web-tools web-search "人工智能" --locale zh-CN --limit 3

# Read a URL
web-tools web-reader https://example.com/article

# Convert a file
web-tools web-reader ./report.pdf
```

## Architecture

```
web-tools
├── cmd/web-reader/     # web-reader CLI entry point
├── cmd/web-search/     # web-search CLI entry point
├── internal/
│   ├── config/         # Configuration loading (file + env + defaults)
│   ├── errors/         # Structured error handling for agent consumption
│   ├── reader/         # HTTP fetch, readability extraction, cache, converter, browser fallback
│   └── search/         # SearXNG client, result parsing, output formatting
├── docker/             # SearXNG docker-compose.yml + settings
└── skill/              # Agent skill documentation
```

## License

Private project.
