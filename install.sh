#!/usr/bin/env bash
# l2h-client one-click install script
# Usage: curl -fsSL https://raw.githubusercontent.com/Kaiyuan/l2h-client/main/install.sh | bash

set -euo pipefail

REPO="Kaiyuan/l2h-client"
INSTALL_DIR="/usr/local/bin"
BIN_NAME="l2h-cli"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# Root check
[ "$(id -u)" -eq 0 ] || error "Please run as root (sudo) to install to /usr/local/bin."

# Detect OS and Arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${ARCH}" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7l) ARCH="armv7" ;;
  *) error "Unsupported architecture: ${ARCH}" ;;
esac

if [[ "${OS}" != "linux" && "${OS}" != "darwin" ]]; then
  error "This script currently only supports Linux and macOS. For Windows, download from GitHub Releases."
fi

info "Detecting latest release from GitHub..."
LATEST_TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" | grep -Po '"tag_name": "\K[^"]*')
if [[ -z "${LATEST_TAG}" ]]; then
  error "Failed to fetch latest tag for ${REPO}."
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/l2h-cli-${OS}-${ARCH}"

info "Downloading ${BIN_NAME} (${LATEST_TAG}) for ${OS}-${ARCH}..."
if ! curl -fsSL -o "${INSTALL_DIR}/${BIN_NAME}" "${DOWNLOAD_URL}"; then
    error "Failed to download from ${DOWNLOAD_URL}. The file for your architecture might not exist for this tag."
fi

chmod +x "${INSTALL_DIR}/${BIN_NAME}"

info "====================================="
info "l2h-client installed to ${INSTALL_DIR}/${BIN_NAME}"
info "Try: ${BIN_NAME} -h"
info "====================================="
