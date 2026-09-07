#!/usr/bin/env bash
# ==============================================================================
# LiteRT-LM System-Wide Dependency Installer for Go
# Usage:
#   curl -sSL https://raw.githubusercontent.com/benmizrahi/adk-litert/main/scripts/install.sh | sudo bash
#   or:
#   sudo ./scripts/install.sh
# ==============================================================================
set -euo pipefail

RELEASE_TAG="v0.16.0"
ZIP_URL="https://github.com/google-ai-edge/LiteRT-LM/releases/download/${RELEASE_TAG}/litert_lm_c_api-0.1.0.zip"
TMP_ZIP="/tmp/litert_lm_c_api.zip"

INSTALL_LIB_DIR="/usr/local/lib"
INSTALL_INC_DIR="/usr/local/include/litert"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        NORM_ARCH="x86_64"
        ;;
    aarch64|arm64)
        NORM_ARCH="arm64"
        ;;
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

echo "==> Detected OS: $OS, Architecture: $ARCH ($LIB_SUBDIR)"

if [ ! -f "$TMP_ZIP" ]; then
    echo "==> Downloading LiteRT-LM prebuilts (${RELEASE_TAG})..."
    curl -f -sSL "$ZIP_URL" -o "$TMP_ZIP"
else
    echo "==> Using cached archive $TMP_ZIP"
fi

echo "==> Installing $LIB_NAME to $INSTALL_LIB_DIR..."
mkdir -p "$INSTALL_LIB_DIR" "$INSTALL_INC_DIR"

unzip -q -o -j "$TMP_ZIP" "lib/${LIB_SUBDIR}/${LIB_NAME}" -d "$INSTALL_LIB_DIR/"
unzip -q -o -j "$TMP_ZIP" "include/*" -d "$INSTALL_INC_DIR/"

# Also copy headers to /usr/local/include for easy compiler resolution
cp -f "$INSTALL_INC_DIR"/* /usr/local/include/ 2>/dev/null || true

if [ "$OS" = "linux" ]; then
    if command -v ldconfig >/dev/null 2>&1; then
        echo "==> Updating library cache with ldconfig..."
        ldconfig
    fi
fi

if [ "$OS" = "darwin" ]; then
    LIB_PATH="$INSTALL_LIB_DIR/$LIB_NAME"

    # The upstream macOS build ships with the Bazel-generated install name
    # "@rpath/liblitert-lm.so", which does not match the .dylib file name and
    # is not resolvable from a Go module cache. Rewrite it to the absolute
    # install path so consumers link without needing any rpath entry.
    if command -v install_name_tool >/dev/null 2>&1; then
        echo "==> Rewriting install name to $LIB_PATH..."
        chmod u+w "$LIB_PATH"
        install_name_tool -id "$LIB_PATH" "$LIB_PATH"
        # install_name_tool invalidates the ad-hoc signature on arm64.
        if command -v codesign >/dev/null 2>&1; then
            codesign --force --sign - "$LIB_PATH" >/dev/null 2>&1 || true
        fi
        chmod a-w "$LIB_PATH"
    fi

    # Compatibility alias for binaries already linked against
    # "@rpath/liblitert-lm.so" before the install name was corrected.
    ln -sf "$LIB_NAME" "$INSTALL_LIB_DIR/liblitert-lm.so"
fi

echo "==> Success! LiteRT-LM C runtime installed to $INSTALL_LIB_DIR"
echo "==> Go packages can now import 'github.com/benmizrahi/adk-litert' and build with standard 'go build'."
