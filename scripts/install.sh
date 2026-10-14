#!/usr/bin/env bash
# agent-ecosystem installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.sh | bash
#
# Methods, in order: Homebrew (macOS), pre-built binary from GitHub Releases
# (checksum verified), go install.
set -euo pipefail

GITHUB_OWNER="iesedobleac"
GITHUB_REPO="agent-ecosystem"
BINARY_NAME="agent-ecosystem"
BREW_TAP="iesedobleac/tap"
GO_MODULE="github.com/${GITHUB_OWNER}/${GITHUB_REPO}"

setup_colors() {
    if [ -t 1 ] && [ "${TERM:-}" != "dumb" ]; then
        RED='\033[0;31m' GREEN='\033[0;32m' YELLOW='\033[1;33m' BLUE='\033[0;34m' CYAN='\033[0;36m' BOLD='\033[1m' DIM='\033[2m' NC='\033[0m'
    else
        RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' DIM='' NC=''
    fi
}
info()    { echo -e "${BLUE}[info]${NC}    $*"; }
success() { echo -e "${GREEN}[ok]${NC}      $*"; }
warn()    { echo -e "${YELLOW}[warn]${NC}    $*"; }
fatal()   { echo -e "${RED}[error]${NC}   $*" >&2; exit 1; }
step()    { echo -e "\n${CYAN}${BOLD}==>${NC} ${BOLD}$*${NC}"; }

show_help() {
    cat <<EOF
agent-ecosystem installer

Usage: install.sh [OPTIONS]

Options:
  --method METHOD   brew, binary or go (default: brew on macOS when available, else binary)
  --dir DIR         Install directory for the binary method
  --insecure        Skip checksum verification (not recommended)
  -h, --help        Show this help
EOF
}

detect_platform() {
    case "$(uname -s)" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux" ;;
        *) fatal "Unsupported OS: $(uname -s). On Windows use scripts/install.ps1." ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) fatal "Unsupported architecture: $(uname -m)" ;;
    esac
    success "Platform: ${OS}/${ARCH}"
}

check_prerequisites() {
    step "Checking prerequisites"
    local missing=()
    command -v curl >/dev/null 2>&1 || missing+=("curl")
    command -v git  >/dev/null 2>&1 || missing+=("git")
    [ ${#missing[@]} -eq 0 ] || fatal "Missing required tools: ${missing[*]}"
    success "curl and git are available"
    if ! command -v node >/dev/null 2>&1; then
        warn "Node.js not found: CodeGraph needs it (agent-ecosystem doctor shows how to install it)"
    fi
}

detect_method() {
    if [ -n "${FORCE_METHOD}" ]; then
        case "${FORCE_METHOD}" in brew|binary|go) METHOD="${FORCE_METHOD}" ;; *) fatal "Unknown method: ${FORCE_METHOD}" ;; esac
        info "Using method: ${METHOD}"
        return
    fi
    if [ "${OS}" = "darwin" ] && command -v brew >/dev/null 2>&1; then
        METHOD="brew"
        success "Homebrew found: installing from ${BREW_TAP}"
    else
        METHOD="binary"
        info "Downloading the pre-built binary from GitHub Releases"
    fi
}

install_brew() {
    step "Installing with Homebrew"
    brew tap "${BREW_TAP}" >/dev/null
    if brew list --cask "${BINARY_NAME}" >/dev/null 2>&1; then
        brew upgrade --cask "${BREW_TAP}/${BINARY_NAME}" || success "${BINARY_NAME} is already up to date"
    else
        brew install --cask "${BREW_TAP}/${BINARY_NAME}"
    fi
    success "Installed ${BINARY_NAME} with Homebrew"
}

latest_version() {
    local body
    body="$(curl -fsSL "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest")" \
        || fatal "Could not query the latest release (rate limited? try --method go)"
    LATEST="$(printf '%s\n' "$body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
    [ -n "${LATEST}" ] || fatal "Could not determine the latest version"
    VERSION="${LATEST#v}"
    success "Latest version: ${LATEST}"
}

install_binary() {
    step "Installing the pre-built binary"
    latest_version
    local archive="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
    local base="https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${LATEST}"
    local tmp
    tmp="$(mktemp -d)"
    # Expand now: the trap runs after this function's locals are gone.
    trap "rm -rf '${tmp}'" EXIT

    info "Downloading ${archive}"
    curl -fsSL -o "${tmp}/${archive}" "${base}/${archive}" || fatal "Download failed: ${base}/${archive}"

    if [ "${INSECURE}" = "true" ]; then
        warn "Checksum verification skipped (--insecure)"
    else
        curl -fsSL -o "${tmp}/checksums.txt" "${base}/checksums.txt" || fatal "Could not download checksums.txt"
        local expected actual
        expected="$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')"
        [ -n "${expected}" ] || fatal "${archive} is not listed in checksums.txt"
        if command -v sha256sum >/dev/null 2>&1; then
            actual="$(sha256sum "${tmp}/${archive}" | awk '{print $1}')"
        else
            actual="$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')"
        fi
        [ "${actual}" = "${expected}" ] || fatal "Checksum mismatch for ${archive}"
        success "Checksum verified"
    fi

    tar -xzf "${tmp}/${archive}" -C "${tmp}"
    [ -f "${tmp}/${BINARY_NAME}" ] || fatal "${BINARY_NAME} not found in the archive"

    local dir="${INSTALL_DIR}"
    if [ -z "${dir}" ]; then
        if [ -w /usr/local/bin ]; then dir="/usr/local/bin"; else dir="${HOME}/.local/bin"; fi
    fi
    mkdir -p "${dir}"
    # Stage next to the destination and rename, so an existing binary is replaced atomically.
    install -m 0755 "${tmp}/${BINARY_NAME}" "${dir}/${BINARY_NAME}.staging.$$"
    mv -f "${dir}/${BINARY_NAME}.staging.$$" "${dir}/${BINARY_NAME}"
    success "Installed ${dir}/${BINARY_NAME}"
    case ":${PATH}:" in
        *":${dir}:"*) ;;
        *) warn "${dir} is not in your PATH. Add: export PATH=\"\$PATH:${dir}\"" ;;
    esac
}

install_go() {
    step "Installing with go install"
    command -v go >/dev/null 2>&1 || fatal "Go is not installed. Install it from https://go.dev/dl/ or use --method binary."
    go install "${GO_MODULE}/cmd/${BINARY_NAME}@latest" || fatal "go install failed"
    local gobin
    gobin="$(go env GOBIN)"
    [ -n "${gobin}" ] || gobin="$(go env GOPATH)/bin"
    case ":${PATH}:" in
        *":${gobin}:"*) ;;
        *) warn "${gobin} is not in your PATH. Add: export PATH=\"\$PATH:${gobin}\"" ;;
    esac
    success "Installed ${BINARY_NAME} with go install"
}

main() {
    setup_colors
    FORCE_METHOD="" INSTALL_DIR="" INSECURE="false"
    while [ $# -gt 0 ]; do
        case "$1" in
            --method) [ $# -ge 2 ] || fatal "--method requires a value"; FORCE_METHOD="$2"; shift 2 ;;
            --dir) [ $# -ge 2 ] || fatal "--dir requires a value"; INSTALL_DIR="$2"; shift 2 ;;
            --insecure) INSECURE="true"; shift ;;
            -h|--help) show_help; exit 0 ;;
            *) fatal "Unknown option: $1" ;;
        esac
    done

    echo -e "${CYAN}${BOLD}agent-ecosystem${NC} ${DIM}— agents, skills and MCP servers for your AI assistants${NC}"
    step "Detecting platform"
    detect_platform
    check_prerequisites
    detect_method
    case "${METHOD}" in
        brew) install_brew ;;
        binary) install_binary ;;
        go) install_go ;;
    esac

    hash -r 2>/dev/null || true
    echo
    echo -e "${GREEN}${BOLD}Installation complete.${NC}"
    echo -e "Run ${BOLD}${BINARY_NAME}${NC} to open the interactive installer, or ${BOLD}${BINARY_NAME} install --help${NC}."
}

# Run main when executed or piped to bash; not when sourced.
if [ -z "${BASH_SOURCE[0]:-}" ] || [ "${BASH_SOURCE[0]}" = "$0" ]; then
    main "$@"
fi
