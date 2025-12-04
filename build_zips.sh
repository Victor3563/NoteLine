#!/usr/bin/env bash
set -euo pipefail

# Simple rebuild script for Noteline zips (Linux/MacOS/Windows)
# Usage:
#   ./rebuild_zips.sh            -> build linux, macos, windows with ARCH=amd64
#   ARCH=arm64 ./rebuild_zips.sh linux
#   ./rebuild_zips.sh linux macos
#
# Requirements: go, zip. rsync is optional (preferred). Works on Linux (bash).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Config
CMD_DIR="cmd/noteline"
I18N_SRC="internal/i18n"
RELEASE_DIR="release"
ZIPS_DIR="$SCRIPT_DIR/zips"

# Map the release subfolder name to GOOS
declare -A GOOS_MAP
GOOS_MAP[linux]=linux
GOOS_MAP[macos]=darwin
GOOS_MAP[windows]=windows

# Default arch
ARCH="${ARCH:-amd64}"

# Targets passed on CLI or default list
if [ $# -gt 0 ]; then
  TARGETS=("$@")
else
  TARGETS=(linux macos windows)
fi

# Check prerequisites
command -v go >/dev/null 2>&1 || { echo "ERROR: 'go' not found in PATH. Install Go." >&2; exit 1; }
command -v zip >/dev/null 2>&1 || { echo "ERROR: 'zip' not found in PATH. Install zip." >&2; exit 1; }

# Prepare dirs
mkdir -p "$ZIPS_DIR"

# Copy only jsons preserving directory tree
copy_json_tree() {
  src="$1"
  dest="$2"
  if [ ! -d "$src" ]; then
    echo "Warning: i18n source '$src' not found; creating empty '$dest'." >&2
    mkdir -p "$dest"
    return 0
  fi
  mkdir -p "$dest"
  if command -v rsync >/dev/null 2>&1; then
    rsync -avm --include='*/' --include='*.json' --exclude='*' "$src"/ "$dest"/
  else
    # fallback: find + cp (preserve dirs)
    (cd "$src"
      find . -type f -name '*.json' -print0 | while IFS= read -r -d $'\0' f; do
        d="$(dirname "$f")"
        mkdir -p "$dest/$d"
        cp -p -- "$f" "$dest/$d/"
      done
    )
  fi
}

for target in "${TARGETS[@]}"; do
  if [ -z "${GOOS_MAP[$target]+_}" ]; then
    echo "Skipping unknown target: $target"
    continue
  fi

  goos="${GOOS_MAP[$target]}"
  goarch="${ARCH}"

  echo
  echo "=== Building for target: $target (GOOS=$goos GOARCH=$goarch) ==="

  staging="$SCRIPT_DIR/$RELEASE_DIR/$target"
  echo " -> staging dir: $staging"

  rm -rf "$staging"
  mkdir -p "$staging"

  # Choose binary name so that install logic can find it:
  # - windows needs extension .exe and expected name like noteline-windows-<arch>.exe
  if [ "$target" = "windows" ]; then
    exe_name="noteline-windows-${goarch}.exe"
  else
    exe_name="noteline-${target}-${goarch}"
  fi

  echo " -> building binary: $exe_name"
  # Build in module-aware mode; ensure we run from repo root (SCRIPT_DIR)
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$staging/$exe_name" "./$CMD_DIR"
  # verify binary was produced
  if [ ! -f "$staging/$exe_name" ]; then
    echo "ERROR: build didn't produce $staging/$exe_name" >&2
    exit 2
  fi
  chmod +x "$staging/$exe_name" || true

  echo " -> copying JSON locales from $I18N_SRC to $staging/i18n"
  copy_json_tree "$I18N_SRC" "$staging/i18n"

  # Make sure the staging contains at least the binary and i18n (i18n may be empty)
  echo " -> staging contents (short):"
  ls -la "$staging" || true
  echo " -> i18n contents (short):"
  ls -la "$staging/i18n" || true

  zip_path="$ZIPS_DIR/noteline-$target.zip"
  echo " -> creating zip: $zip_path"
  # Remove previous zip if exists
  rm -f "$zip_path"
  (
    cd "$staging"
    # include everything in staging. Use -r . to ensure dirs are preserved.
    zip -r "$zip_path" . >/dev/null
  )
  if [ ! -f "$zip_path" ]; then
    echo "ERROR: zip was not created: $zip_path" >&2
    exit 3
  fi

  echo " -> wrote $zip_path"
  echo " -> preview of zip contents:"
  unzip -l "$zip_path" | sed -n '1,20p'
done

echo
echo "All done. Zips are in: $ZIPS_DIR"
