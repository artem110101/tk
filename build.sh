#!/bin/sh
set -eu
cd "$(dirname "$0")"
target_os=$(go env GOOS)
case "$target_os" in
    linux|darwin) ;;
    *) echo 'tk builds are supported on Linux and macOS only.' >&2; exit 1 ;;
esac
export CGO_ENABLED=1
go test ./...
go build -trimpath -o bin/tk .
