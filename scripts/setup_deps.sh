#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LIB_DIR="$ROOT_DIR/lib"
INCLUDE_DIR="$ROOT_DIR/include"
SO_FILE="$LIB_DIR/liblitert-lm.so"
RELEASE_TAG="v0.16.0"
ZIP_URL="https://github.com/google-ai-edge/LiteRT-LM/releases/download/${RELEASE_TAG}/litert_lm_c_api-0.1.0.zip"

if [ -f "$SO_FILE" ] && [ -d "$INCLUDE_DIR" ]; then
    echo "==> LiteRT-LM prebuilt C runtime library is ready in $LIB_DIR"
    exit 0
fi

echo "==> LiteRT-LM prebuilt C runtime library not found."
echo "==> Downloading official prebuilts from GitHub (${RELEASE_TAG})..."

TMP_ZIP="/tmp/litert_lm_c_api.zip"
curl -sSL "$ZIP_URL" -o "$TMP_ZIP"

mkdir -p "$LIB_DIR" "$INCLUDE_DIR"

echo "==> Extracting prebuilt libraries and headers..."
unzip -q -o -j "$TMP_ZIP" "lib/linux_x86_64/liblitert-lm.so" -d "$LIB_DIR/"
unzip -q -o -j "$TMP_ZIP" "include/*" -d "$INCLUDE_DIR/"

# Also provide symlink for liblitert_lm_runtime.so if needed
ln -sf liblitert-lm.so "$LIB_DIR/liblitert_lm_runtime.so"

rm -f "$TMP_ZIP"
echo "==> LiteRT-LM prebuilt runtime successfully installed to $LIB_DIR and $INCLUDE_DIR"
