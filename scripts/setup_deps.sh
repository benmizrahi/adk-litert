#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LIB_DIR="$ROOT_DIR/lib"
INCLUDE_DIR="$ROOT_DIR/include"
RELEASE_TAG="v0.16.0"
ZIP_URL="https://github.com/google-ai-edge/LiteRT-LM/releases/download/${RELEASE_TAG}/litert_lm_c_api-0.1.0.zip"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)   NORM_ARCH="x86_64" ;;
    aarch64|arm64)  NORM_ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        LIB_SUBDIR="linux_${NORM_ARCH}"
        LIB_NAME="liblitert-lm.so"
        ;;
    darwin)
        if [ "$NORM_ARCH" != "arm64" ]; then
            echo "Error: LiteRT-LM macOS builds currently support arm64 (Apple Silicon) only."
            exit 1
        fi
        LIB_SUBDIR="macos_arm64"
        LIB_NAME="liblitert-lm.dylib"
        ;;
    *)
        echo "Error: Unsupported operating system: $OS (expected linux or darwin)"
        exit 1
        ;;
esac

LIB_FILE="$LIB_DIR/$LIB_NAME"

if [ -f "$LIB_FILE" ] && [ -d "$INCLUDE_DIR" ]; then
    echo "==> LiteRT-LM prebuilt C runtime library is ready in $LIB_DIR"
    exit 0
fi

echo "==> LiteRT-LM prebuilt C runtime library not found."
echo "==> Downloading official prebuilts from GitHub (${RELEASE_TAG}) for ${LIB_SUBDIR}..."

TMP_ZIP="/tmp/litert_lm_c_api.zip"
curl -f -sSL "$ZIP_URL" -o "$TMP_ZIP"

mkdir -p "$LIB_DIR" "$INCLUDE_DIR"

echo "==> Extracting prebuilt libraries and headers..."
unzip -q -o -j "$TMP_ZIP" "lib/${LIB_SUBDIR}/${LIB_NAME}" -d "$LIB_DIR/"
unzip -q -o -j "$TMP_ZIP" "include/*" -d "$INCLUDE_DIR/"

if [ "$OS" = "darwin" ]; then
    # The upstream macOS build carries the Bazel-generated install name
    # "@rpath/liblitert-lm.so", which does not match the .dylib file name.
    # Point it at the checkout so in-tree builds resolve it at run time.
    if command -v install_name_tool >/dev/null 2>&1; then
        chmod u+w "$LIB_FILE"
        install_name_tool -id "$LIB_FILE" "$LIB_FILE"
        # install_name_tool invalidates the ad-hoc signature on arm64.
        if command -v codesign >/dev/null 2>&1; then
            codesign --force --sign - "$LIB_FILE" >/dev/null 2>&1 || true
        fi
    fi
    # Compatibility alias for binaries linked before the install name was fixed.
    ln -sf "$LIB_NAME" "$LIB_DIR/liblitert-lm.so"
else
    # Symlink for consumers expecting the runtime's alternate SONAME.
    ln -sf "$LIB_NAME" "$LIB_DIR/liblitert_lm_runtime.so"
fi

rm -f "$TMP_ZIP"
echo "==> LiteRT-LM prebuilt runtime successfully installed to $LIB_DIR and $INCLUDE_DIR"
