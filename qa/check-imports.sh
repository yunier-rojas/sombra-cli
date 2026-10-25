#!/usr/bin/env bash
#
# Replace the __MODULE__ placeholder in qa/import-*.yaml with this repository's
# module path, then run go-import-checks on each config.

set -euo pipefail

module="$(GOWORK=off go list -m)"
if [ -z "$module" ]; then
	echo "check-imports: cannot resolve Go module" >&2
	exit 1
fi

shopt -s nullglob
configs=(qa/import-*.yaml)
if [ "${#configs[@]}" -eq 0 ]; then
	echo "check-imports: no qa/import-*.yaml configs found" >&2
	exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

for config in "${configs[@]}"; do
	name="$(basename "$config")"
	sed "s|__MODULE__|${module}|g" "$config" > "$tmp/$name"
	go run github.com/quantumcycle/go-import-checks@latest --config "$tmp/$name"
done
