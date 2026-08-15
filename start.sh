#!/usr/bin/env zsh
DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ -f "$DIR/ob1.pid" ]] && kill -0 "$(cat "$DIR/ob1.pid")" 2>/dev/null; then
  echo "ob1-go already running (pid $(cat "$DIR/ob1.pid"))"
  exit 0
fi

if [[ ! -x "$DIR/ob1-go" ]]; then
  echo "no binary at $DIR/ob1-go — run ./build.sh first" >&2
  exit 1
fi

DB_PATH="${DB_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/open-brain/openbrain.db}"
mkdir -p "$(dirname "$DB_PATH")"

"$DIR/ob1-go" -env "$DIR/.env" -db "$DB_PATH" >> "$DIR/ob1.log" 2>&1 &

echo $! > "$DIR/ob1.pid"
echo "ob1-go started (pid $!)"
