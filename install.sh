#!/usr/bin/env bash
set -euo pipefail

REPO="Victor3563/NoteLine"
REL_BASE="https://github.com/${REPO}/releases/latest/download"
TMPDIR="$(mktemp -d)"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)   ZIP="noteline-linux.zip" ;;
  darwin*)  ZIP="noteline-macos.zip" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

URL="${REL_BASE}/${ZIP}"
echo "Downloading ${ZIP} ..."
curl -L "$URL" -o "$TMPDIR/$ZIP"

echo "Extracting..."
unzip -o "$TMPDIR/$ZIP" -d "$TMPDIR/extracted"

# Paths we will use
BIN_TARGET_DIR="/usr/local/lib/noteline"       # где храним сам бинарник
WRAPPER="/usr/local/bin/noteline"              # исполняемый wrapper в PATH
DATA_DIR="/usr/local/share/noteline"           # где будут локали (i18n)

# Создаём каталоги (потребуется sudo для системных директорий)
sudo mkdir -p "$BIN_TARGET_DIR"
sudo mkdir -p "$DATA_DIR"

# Найдём бинарник и i18n внутри распакованного архива
BIN_FOUND="$(find "$TMPDIR/extracted" -maxdepth 2 -type f -iname "noteline*" ! -iname "*.zip" | head -n 1)"
if [ -z "$BIN_FOUND" ]; then
  echo "Binary not found in archive."
  exit 1
fi

# Копируем бинарник (в bin target под именем noteline.bin)
sudo cp "$BIN_FOUND" "$BIN_TARGET_DIR/noteline.bin"
sudo chmod +x "$BIN_TARGET_DIR/noteline.bin"

# Копируем папку i18n (если есть) в DATA_DIR
if [ -d "$TMPDIR/extracted/i18n" ]; then
  sudo rm -rf "$DATA_DIR/i18n" || true
  sudo cp -r "$TMPDIR/extracted/i18n" "$DATA_DIR/i18n"
fi

# Создаём wrapper в /usr/local/bin, который экспортирует переменную и exec'ит бинарник
sudo tee "$WRAPPER" >/dev/null <<'EOF'
#!/bin/sh
export NOTELINE_I18N_DIR="/usr/local/share/noteline/i18n"
exec /usr/local/lib/noteline/noteline.bin "$@"
EOF

sudo chmod +x "$WRAPPER"

# cleanup
rm -rf "$TMPDIR"

echo "Installed!"
echo "Binary: /usr/local/lib/noteline/noteline.bin"
echo "Locales: /usr/local/share/noteline/i18n"
echo "Wrapper: /usr/local/bin/noteline"
echo "Теперь можно запускать: noteline -h"
