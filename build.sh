#!/bin/sh
set -eu
cd "$(dirname "$0")"
export CGO_ENABLED=1
go test ./...
go build -trimpath -o bin/tk .
