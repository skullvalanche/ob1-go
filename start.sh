#!/usr/bin/env zsh
DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ -f "$DIR/ob1.pid" ]] && kill -0 "$(cat "$DIR/ob1.pid")" 2>/dev/null; then
  echo "ob1-go already running (pid $(cat "$DIR/ob1.pid"))"
  exit 0
fi

source "$DIR/.env"
mkdir -p "$DIR/data"

(cd "$DIR" && go build -o ob1-go .)

DB_PATH="$DIR/data/openbrain.db" \
MCP_ACCESS_KEY="$MCP_ACCESS_KEY" \
PORT="${PORT:-8081}" \
"$DIR/ob1-go" >> "$DIR/ob1.log" 2>&1 &

echo $! > "$DIR/ob1.pid"
echo "ob1-go started (pid $!)"
