#!/bin/sh
# GitSketch 1-Command Installer
set -e

OWNER="Wian47"
REPO="GitSketch"

# ─── OS/Arch Detection ───────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    linux)   OS="linux" ;;
    darwin)  OS="darwin" ;;
    *)       echo "✗ Unsupported OS: ${OS}"; exit 1 ;;
esac

ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)       echo "✗ Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

# ─── Resolve Download URL ───────────────────────────────────────────────────

echo "🔍 Checking latest release of GitSketch..."
API_URL="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"

# Get release JSON
JSON=$(curl -fsSL "${API_URL}" || wget -qO- "${API_URL}")

# Extract release tag name
TAG=$(echo "${JSON}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "${TAG}" ]; then
    echo "✗ Failed to resolve latest release tag."
    exit 1
fi

echo "📦 Found version: ${TAG}"

# Extract matching download URL
DOWNLOAD_URL=$(echo "${JSON}" | grep '"browser_download_url":' | grep "${OS}" | grep "${ARCH}" | grep '\.tar\.gz' | sed -E 's/.*"([^"]+)".*/\1/' | head -n 1)

if [ -z "${DOWNLOAD_URL}" ]; then
    echo "✗ No pre-built release package found for ${OS}_${ARCH}."
    exit 1
fi

# ─── Download and Extract ────────────────────────────────────────────────────

TMP_DIR=$(mktemp -d)
clean_up() {
    rm -rf "${TMP_DIR}"
}
trap clean_up EXIT

echo "📥 Downloading from ${DOWNLOAD_URL}..."
curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/gitsketch.tar.gz"

echo "📦 Extracting package..."
tar -xzf "${TMP_DIR}/gitsketch.tar.gz" -C "${TMP_DIR}"

# ─── Install to Path ─────────────────────────────────────────────────────────

BINARY_PATH="${TMP_DIR}/gitsketch"
if [ ! -f "${BINARY_PATH}" ]; then
    echo "✗ Binary not found in extracted archive."
    exit 1
fi

INSTALL_DIR="/usr/local/bin"
FALLBACK_DIR="${HOME}/.local/bin"

install_to() {
    local dest_dir="$1"
    local use_sudo="$2"

    if [ "${use_sudo}" = "true" ]; then
        echo "🔐 Elevated permissions needed to write to ${dest_dir}. Running sudo..."
        sudo cp "${BINARY_PATH}" "${dest_dir}/gitsketch"
        sudo chmod +x "${dest_dir}/gitsketch"
    else
        cp "${BINARY_PATH}" "${dest_dir}/gitsketch"
        chmod +x "${dest_dir}/gitsketch"
    fi
    echo "✓ GitSketch successfully installed to ${dest_dir}/gitsketch!"
}

# Check write access to /usr/local/bin
if [ -w "${INSTALL_DIR}" ]; then
    install_to "${INSTALL_DIR}" "false"
else
    # Prompt for sudo fallback or choose user-space fallback
    echo "⚠️  No write access to ${INSTALL_DIR}."
    printf "Would you like to install globally using sudo? (y/n): "
    read -r choice < /dev/tty
    if [ "${choice}" = "y" ] || [ "${choice}" = "Y" ]; then
        install_to "${INSTALL_DIR}" "true"
    else
        # Try user-space fallback
        mkdir -p "${FALLBACK_DIR}"
        install_to "${FALLBACK_DIR}" "false"

        # Check if fallback dir is in PATH
        case ":${PATH}:" in
            *:"${FALLBACK_DIR}":*) ;;
            *)
                echo ""
                echo "⚠️  Warning: ${FALLBACK_DIR} is not in your PATH."
                echo "Please add the following line to your ~/.bashrc, ~/.zshrc, or profile:"
                echo "  export PATH=\"\$PATH:${FALLBACK_DIR}\""
                ;;
        esac
    fi
fi
