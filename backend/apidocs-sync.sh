#!/usr/bin/env sh
# apidocs-sync.sh copies the canonical OpenAPI contract into backend/apidocs/
# so the go:embed in apidocs.go resolves. CI runs the same copy before both
# `go test` and the image build; run this locally after editing the contract so
# the committed copy and the served /docs spec stay identical to the source.
#
# Lives in backend/ rather than backend/apidocs/ on purpose: a script inside the
# embedded directory could be pulled into the binary. Not here.
set -eu

# Resolve paths relative to this script so it works from any working directory.
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

SRC="$REPO_ROOT/docs/001-capacity-exchange-marketplace/contracts/openapi.yaml"
DST="$SCRIPT_DIR/apidocs/openapi.yaml"

if [ ! -f "$SRC" ]; then
	echo "sumber kontrak tidak ditemukan: $SRC" >&2
	exit 1
fi

mkdir -p "$SCRIPT_DIR/apidocs"
cp "$SRC" "$DST"
echo "tersinkron: $DST"
