#!/usr/bin/env bash
# Builds displayswitch.exe and the release zip from Linux or macOS (cross-compiles for Windows).
# Usage: ./scripts/build.sh
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
plugin="$root/com.marco.displayswitch.sdPlugin"
version="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["Version"])' "$plugin/manifest.json")"

cd "$root"
go test ./...
GOOS=windows GOARCH=amd64 go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o "$plugin/displayswitch.exe" ./cmd/displayswitch

stage="$root/build/display-switch-ss552-v$version"
rm -rf "$stage" "$stage.zip"
mkdir -p "$stage"
cp -r "$plugin" install.ps1 install.cmd uninstall.ps1 uninstall.cmd README.md LICENSE "$stage/"
(cd "$plugin" && sha256sum displayswitch.exe) > "$stage/SHA256SUMS.txt"
(cd "$root/build" && zip -qr "display-switch-ss552-v$version.zip" "display-switch-ss552-v$version")

echo "Built $stage.zip"
cat "$stage/SHA256SUMS.txt"
