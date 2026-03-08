#!/usr/bin/env bash
# linux-install-deps.sh — Install runtime dependencies required by iptw on Linux.
#
# iptw is built with CGO and links against:
#   • GTK 3                            (libgtk-3.so.0)
#   • Ayatana AppIndicator 3           (libayatana-appindicator3.so.1)  ← system tray
#   • Legacy AppIndicator 3 (fallback) (libappindicator3.so.1)
#
# Run as root or with sudo:
#   sudo ./linux-install-deps.sh

set -euo pipefail

# ── helpers ──────────────────────────────────────────────────────────────────

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()    { echo -e "${BOLD}[iptw]${NC} $*"; }
success() { echo -e "${GREEN}[iptw]${NC} $*"; }
warn()    { echo -e "${YELLOW}[iptw]${NC} $*"; }
error()   { echo -e "${RED}[iptw]${NC} $*" >&2; }

require_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root (or with sudo)."
        echo "  sudo $0"
        exit 1
    fi
}

# Detect distro from /etc/os-release
detect_distro() {
    if [[ -f /etc/os-release ]]; then
        # shellcheck source=/dev/null
        source /etc/os-release
        DISTRO_ID="${ID:-unknown}"
        DISTRO_ID_LIKE="${ID_LIKE:-}"
        DISTRO_NAME="${NAME:-Linux}"
        DISTRO_VERSION="${VERSION_ID:-}"
    else
        DISTRO_ID="unknown"
        DISTRO_ID_LIKE=""
        DISTRO_NAME="Linux"
        DISTRO_VERSION=""
    fi
}

# Returns true if any of the given IDs match DISTRO_ID or DISTRO_ID_LIKE
is_distro() {
    for id in "$@"; do
        [[ "$DISTRO_ID" == "$id" ]] && return 0
        [[ "$DISTRO_ID_LIKE" == *"$id"* ]] && return 0
    done
    return 1
}

# ── per-family install functions ─────────────────────────────────────────────

install_fedora_rhel() {
    info "Detected Fedora/RHEL family — using dnf (or yum)"
    local pm="dnf"
    command -v dnf >/dev/null 2>&1 || pm="yum"

    # Always install gtk3 first.
    info "Installing: gtk3"
    "$pm" install -y gtk3

    # Try to install Ayatana AppIndicator directly; the package is available in
    # Fedora 34+ and RHEL 9+ repos.  We skip the 'dnf info' pre-check because
    # it can return a false negative inside a non-interactive script environment.
    info "Installing: libayatana-appindicator (provides libayatana-appindicator3.so.1)"
    if "$pm" install -y libayatana-appindicator 2>/dev/null; then
        return 0
    fi

    # Try the alternate package name used on some RHEL/CentOS streams.
    if "$pm" install -y libayatana-appindicator3 2>/dev/null; then
        return 0
    fi

    # Last resort: the legacy Canonical fork.  NOTE: this provides
    # libappindicator3.so.1, which is a DIFFERENT library from
    # libayatana-appindicator3.so.1; the binary will still fail to start.
    # Kept here only to avoid a hard error on very old distros while the user
    # is pointed to the proper fix.
    error "libayatana-appindicator is not available in the currently enabled repos."
    error "The iptw binary requires: libayatana-appindicator3.so.1"
    echo ""
    echo "  On Fedora, enable the package and retry:"
    echo "    sudo dnf install libayatana-appindicator"
    echo ""
    echo "  On RHEL/CentOS you may need to enable EPEL first:"
    echo "    sudo dnf install epel-release && sudo dnf install libayatana-appindicator"
    echo ""
    exit 1
}

install_debian_ubuntu() {
    info "Detected Debian/Ubuntu family — using apt"
    apt-get update -qq

    # Prefer Ayatana; fall back to the Canonical legacy package.
    local pkgs=(libgtk-3-0)

    if apt-cache show libayatana-appindicator3-1 &>/dev/null 2>&1; then
        pkgs+=(libayatana-appindicator3-1)
    else
        warn "libayatana-appindicator3-1 not found — falling back to libappindicator3-1"
        pkgs+=(libappindicator3-1)
    fi

    info "Installing: ${pkgs[*]}"
    DEBIAN_FRONTEND=noninteractive apt-get install -y "${pkgs[@]}"
}

install_arch() {
    info "Detected Arch Linux family — using pacman"
    local pkgs=(gtk3 libayatana-appindicator)
    info "Installing: ${pkgs[*]}"
    pacman -Sy --noconfirm "${pkgs[@]}"
}

install_opensuse() {
    info "Detected openSUSE family — using zypper"
    local pkgs=(libgtk-3-0)

    if zypper search -x libayatana-appindicator3-1 &>/dev/null 2>&1; then
        pkgs+=(libayatana-appindicator3-1)
    else
        warn "libayatana-appindicator3-1 not found — falling back to libappindicator3"
        pkgs+=(libappindicator3)
    fi

    info "Installing: ${pkgs[*]}"
    zypper --non-interactive install "${pkgs[@]}"
}

install_alpine() {
    info "Detected Alpine Linux — using apk"
    # Alpine uses musl; the binary must have been built against musl to work.
    warn "The iptw binary is built against glibc (Debian Bookworm)."
    warn "Running it on Alpine requires gcompat or a musl-targeted build."
    local pkgs=(gtk+3.0 libayatana-appindicator)
    info "Installing: ${pkgs[*]}"
    apk add --no-cache "${pkgs[@]}"
}

# ── verify linked libraries ────────────────────────────────────────────────

verify_binary() {
    local binary="${1:-}"
    [[ -z "$binary" ]] && return 0
    [[ -x "$binary" ]] || { warn "Binary not found or not executable: $binary"; return 0; }

    info "Verifying shared library links for: $binary"
    if ldd "$binary" 2>&1 | grep -q "not found"; then
        warn "Some libraries are still missing:"
        ldd "$binary" 2>&1 | grep "not found" | sed 's/^/    /'
        warn "Check the output above and install the corresponding packages manually."
    else
        success "All shared libraries resolved correctly."
    fi
}

# ── main ──────────────────────────────────────────────────────────────────────

main() {
    local binary="${1:-}"

    echo ""
    echo -e "${BOLD}iptw — Linux Dependency Installer${NC}"
    echo "===================================="
    echo ""

    require_root
    detect_distro

    info "Distribution : $DISTRO_NAME ${DISTRO_VERSION}"
    info "ID           : $DISTRO_ID"
    echo ""

    if is_distro fedora rhel centos rocky almalinux ol; then
        install_fedora_rhel
    elif is_distro debian ubuntu linuxmint pop kali raspbian; then
        install_debian_ubuntu
    elif is_distro arch manjaro endeavouros; then
        install_arch
    elif is_distro opensuse suse; then
        install_opensuse
    elif is_distro alpine; then
        install_alpine
    else
        error "Unsupported distribution: $DISTRO_NAME"
        echo ""
        echo "Please install the following libraries manually using your package manager:"
        echo "  • GTK 3 runtime            (provides libgtk-3.so.0)"
        echo "  • Ayatana AppIndicator 3   (provides libayatana-appindicator3.so.1)"
        echo "  • Legacy AppIndicator 3    (provides libappindicator3.so.1)  ← fallback"
        echo ""
        echo "Example package names by family:"
        echo "  Fedora/RHEL : gtk3  libayatana-appindicator"
        echo "  Debian/Ubuntu: libgtk-3-0  libayatana-appindicator3-1"
        echo "  Arch        : gtk3  libayatana-appindicator"
        echo "  openSUSE    : libgtk-3-0  libayatana-appindicator3-1"
        exit 1
    fi

    echo ""
    verify_binary "$binary"

    echo ""
    success "Done! You should now be able to run iptw."
    echo ""
}

# Pass the path to the iptw binary as an optional first argument so the script
# can verify all libraries resolved correctly after installation.
#   sudo ./linux-install-deps.sh ./iptw
main "${1:-}"
