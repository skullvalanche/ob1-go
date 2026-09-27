# ob1-go

Lightweight local AI memory server. Stores and searches thoughts via MCP, backed by SQLite with in-memory cosine similarity.

## Requirements

- Go 1.23+
- An OpenAI-compatible API for embeddings and chat (Ollama, OpenAI, vLLM, etc.)

## Setup

```bash
./setup.sh
```

This checks Go, writes a `.env` with a generated `MCP_ACCESS_KEY`, builds the binary, and prints the `claude mcp add` command with your key filled in. Re-running is safe — an existing `.env` is left alone.

By hand instead:

```bash
cp .env.example .env      # then set MCP_ACCESS_KEY, e.g. openssl rand -hex 32
./build.sh
```

## Usage

```bash
./build.sh   # compile ./ob1-go
./start.sh   # start on PORT (default 8081)
./stop.sh    # stop
```

`start.sh` requires an existing binary — run `./build.sh` after changing code.

Running the binary directly, `.env` is not read unless you ask for it:

```bash
./ob1-go --help
./ob1-go --env .env --db /path/to/openbrain.db
```

Flags: `--db`, `--mcp-access-key`, `--env`. Precedence is flag, then
environment, then `--env` file.

## Connecting clients

All clients connect to the same HTTP endpoint with an auth header:
- **URL:** `http://localhost:8081`
- **Header:** `x-brain-key: YOUR_KEY`

### Claude Code

```bash
claude mcp add --transport http open-brain http://localhost:8081 \
  --header "x-brain-key: YOUR_KEY" --scope user
```

### JSON config clients

Most clients use a JSON config file. Add the appropriate block to your config:

| Client | Config path |
|--------|-------------|
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS)<br>`%APPDATA%\Claude\claude_desktop_config.json` (Windows) |
| Cursor | `.cursor/mcp.json` (project) or `~/.cursor/mcp.json` (global) |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` |
| Antigravity | `~/.gemini/config/mcp_config.json` (global) or `.agents/mcp_config.json` (workspace) |

**Claude Desktop / Cursor** (use `url`):
```json
{
  "mcpServers": {
    "open-brain": {
      "url": "http://localhost:8081",
      "headers": { "x-brain-key": "YOUR_KEY" }
    }
  }
}
```

**Windsurf / Antigravity** (use `serverUrl`):
```json
{
  "mcpServers": {
    "open-brain": {
      "serverUrl": "http://localhost:8081",
      "headers": { "x-brain-key": "YOUR_KEY" }
    }
  }
}
```

### VS Code (Copilot Chat)

Add to your VS Code `settings.json` (different structure):

```json
{
  "mcp": {
    "servers": {
      "open-brain": {
        "url": "http://localhost:8081",
        "headers": { "x-brain-key": "YOUR_KEY" }
      }
    }
  }
}
```

### ChatGPT (Actions)

Add as a custom GPT action (requires public endpoint — ChatGPT cannot reach localhost):
- **Server URL:** Your public endpoint
- **Authentication:** API Key in header `x-brain-key`

The `search` and `fetch` tools return citation URLs for ChatGPT compatibility.

## Tools

| Tool | Description |
|------|-------------|
| `capture_thought` | Store a thought with embedding + metadata |
| `delete_thought` | Permanently delete one thought by ID |
| `search_thoughts` | Semantic search with similarity threshold |
| `list_thoughts` | List recent thoughts, filterable by type/topic/person/days |
| `thought_stats` | Summary counts by type, topic, people |
| `search` | ChatGPT-compatible search (same as search_thoughts) |
| `fetch` | ChatGPT-compatible fetch by ID |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_ACCESS_KEY` | — | **Required.** Auth key for all clients |
| `DB_PATH` | see below | SQLite file path |
| `PORT` | `8081` | HTTP port |
| `EMBEDDING_API_BASE` | `http://127.0.0.1:11434/v1` | OpenAI-compatible embedding endpoint |
| `EMBEDDING_API_KEY` | `ollama` | API key for embedding endpoint |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model |
| `CHAT_API_BASE` | same as embedding | Chat endpoint for metadata extraction |
| `CHAT_API_KEY` | same as embedding | API key for chat endpoint |
| `CHAT_MODEL` | `qwen2.5:1.5b` | Model for metadata extraction |
| `OPEN_BRAIN_CITATION_BASE_URL` | `https://openbrain.local/thoughts` | Base URL for `search`/`fetch` citation links |

**DB_PATH:** `start.sh` defaults to `~/.local/share/open-brain/openbrain.db` (honors `XDG_DATA_HOME`). The binary itself defaults to `/data/openbrain.db` if run directly without `--db`.

Any OpenAI-compatible API works for embeddings and chat. Switching embedding models requires re-embedding existing data.
