#!/usr/bin/env zsh
DIR="$(cd "$(dirname "$0")" && pwd)"
set -e

EMBEDDING_MODEL="${EMBEDDING_MODEL:-nomic-embed-text}"
CHAT_MODEL="${CHAT_MODEL:-qwen2.5:1.5b}"
OLLAMA_HOST_URL="${OLLAMA_HOST_URL:-http://127.0.0.1:11434}"

echo "==> Checking Go"
if ! command -v go >/dev/null; then
  echo "go not found. Install it (brew install go), then re-run." >&2
  exit 1
fi
go version

echo "==> Checking Ollama"
if ! command -v ollama >/dev/null; then
  echo "ollama not found. Install it, then re-run:" >&2
  echo "  brew install ollama        # or see https://ollama.com/download" >&2
  exit 1
fi

if ! curl -sf --max-time 3 "$OLLAMA_HOST_URL/api/version" >/dev/null; then
  echo "Ollama is installed but not serving on $OLLAMA_HOST_URL." >&2
  echo "Start it, then re-run:" >&2
  echo "  brew services start ollama    # or: ollama serve" >&2
  exit 1
fi
echo "serving on $OLLAMA_HOST_URL"

echo "==> Pulling models"
for model in "$EMBEDDING_MODEL" "$CHAT_MODEL"; do
  if ollama list | grep -q "^${model%%:*}"; then
    echo "$model already present"
  else
    ollama pull "$model"
  fi
done

echo "==> Configuring .env"
if [[ -f "$DIR/.env" ]]; then
  echo ".env already exists, leaving it alone"
else
  cp "$DIR/.env.example" "$DIR/.env"
  chmod 600 "$DIR/.env"
  KEY="$(openssl rand -hex 32)"
  # BSD and GNU sed disagree on -i, so rewrite the file instead.
  TMP="$(mktemp)"
  sed "s|^MCP_ACCESS_KEY=.*|MCP_ACCESS_KEY=$KEY|" "$DIR/.env" > "$TMP"
  mv "$TMP" "$DIR/.env"
  chmod 600 "$DIR/.env"
  echo "wrote .env with a generated MCP_ACCESS_KEY"
fi

echo "==> Building"
"$DIR/build.sh"

KEY="$(grep '^MCP_ACCESS_KEY=' "$DIR/.env" | cut -d= -f2-)"
PORT="${PORT:-8081}"

cat <<EOF

Setup complete. Start the server:

  ./start.sh

Then register it with Claude Code:

  claude mcp add --transport http open-brain http://localhost:$PORT \\
    --header "x-brain-key: $KEY" --scope user

EOF
