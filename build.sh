#!/usr/bin/env sh
set -eu

# Build the local NSQLite container image from the repository root.
# Usage:
#   ./build.sh [image-tag] [extra docker build args...]
# Examples:
#   ./build.sh
#   ./build.sh nsqlite:test --no-cache

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to build the local image" >&2
  exit 1
fi

image_tag="${1:-nsqlite:local}"

if [ "$#" -gt 0 ]; then
  shift
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

docker build \
  --tag "${image_tag}" \
  "$@" \
  "${script_dir}"

echo "Built ${image_tag}"
