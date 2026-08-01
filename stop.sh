#!/usr/bin/env zsh
DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="$DIR/ob1.pid"

if [[ -f "$PID_FILE" ]]; then
  PID=$(cat "$PID_FILE")
  if kill "$PID" 2>/dev/null; then
    echo "ob1-go stopped (pid $PID)"
    rm -f "$PID_FILE"
    exit 0
  fi
  rm -f "$PID_FILE"
fi

if pkill -f "ob1-go/ob1-go" 2>/dev/null; then
  echo "ob1-go stopped"
else
  echo "ob1-go not running"
fi
