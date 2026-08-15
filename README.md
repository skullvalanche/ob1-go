# ob1-go

Lightweight local AI memory server. Stores and searches thoughts via MCP, backed by SQLite with in-memory cosine similarity.

## Requirements

- Go 1.23+
- [Ollama](https://ollama.com) with `nomic-embed-text` and `qwen2.5:1.5b`

```bash
ollama pull nomic-embed-text
ollama pull qwen2.5:1.5b
```

## Setup

```bash
cp .env.example .env
```

Edit `.env` and set `MCP_ACCESS_KEY` to a unique value.

If you wish, use `openssl rand -hex 32` to generate a key

## Usage

```bash
./build.sh   # compile ./ob1-go
./start.sh   # start on PORT (default 8081)
./stop.sh    # stop
```

`start.sh` requires an existing binary — run `./build.sh` after changing code.

Data lives at `~/.local/share/open-brain/openbrain.db` (honors `XDG_DATA_HOME`).
Set `DB_PATH` to override.

Running the binary directly, `.env` is not read unless you ask for it:

```bash
./ob1-go --help
./ob1-go --env .env --db /path/to/openbrain.db
```

Flags: `--db`, `--mcp-access-key`, `--env`. Precedence is flag, then
environment, then `--env` file.

## Connecting clients

All clients use the same endpoint and key:

**Claude Code**
```bash
claude mcp add --transport http open-brain http://localhost:8081 \
  --header "x-brain-key: YOUR_KEY" --scope user
```

**Claude Desktop / ChatGPT / Cursor**
- URL: `http://localhost:8081`
- Header: `x-brain-key: YOUR_KEY`

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
| `DB_PATH` | `/data/openbrain.db` | SQLite file path |
| `PORT` | `8081` | HTTP port |
| `EMBEDDING_API_BASE` | `http://127.0.0.1:11434/v1` | OpenAI-compatible embedding endpoint |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model |
| `CHAT_API_BASE` | same as embedding | Chat endpoint for metadata extraction |
| `CHAT_MODEL` | `qwen2.5:1.5b` | Model for metadata extraction |
| `OPEN_BRAIN_CITATION_BASE_URL` | `https://openbrain.local/thoughts` | Base URL for `search`/`fetch` citation links |

Any OpenAI-compatible API works for embeddings and chat — Ollama, OpenAI, Anthropic-compatible proxies, etc. Switching embedding models requires re-embedding existing data.
