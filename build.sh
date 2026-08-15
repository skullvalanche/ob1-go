#!/usr/bin/env zsh
DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$DIR" && go build -o ob1-go . && echo "built $DIR/ob1-go"
