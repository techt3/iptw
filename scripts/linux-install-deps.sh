#!/usr/bin/env bash
# linux-install-deps.sh - Runtime environment check for iptw on Linux.
#
# Since iptw 1.1+ the binary is built with CGO_ENABLED=0. It is a statically
# linked pure-Go binary with NO native library dependencies. The system tray
# is implemented via the StatusNotifierItem D-Bus protocol (fyne.io/systray),
# which communicates over the session D-Bus -- no GTK or AppIndicator .so
# files are needed.
#
# Usage (no root required):
#   ./linux-install-deps.sh [path/to/iptw]

set -euo pipefail

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()    { echo -e "${BOLD}[iptw]${NC} $*"; }
success() { echo -e "${GREEN}[iptw]${NC} $*"; }
warn()    { echo -e "${YELLOW}[iptw]${NC} $*"; }
error()   { echo -e "${RED}[iptw]${NC} $*" >&2; }

# --- checks -------------------------------------------------------------------

check_dbus() {
    if [[ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ]]; then
        warn "DBUS_SESSION_BUS_ADDRESS is not set."
        warn "iptw must be launched inside a desktop session, not over bare SSH."
        warn "If using SSH: export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus"
        return 1
    fi
    if command -v dbus-send >/dev/null 2>&1; then
        if dbus-send --session --print-reply --dest=org.freedesktop.DBus / \
                org.freedesktop.DBus.Ping >/dev/null 2>&1; then
            success "Session D-Bus is reachable."
        else
            warn "Session D-Bus ping failed -- tray icon may not appear."
        fi
    else
        info "dbus-send not found; skipping D-Bus ping (usually fine)."
    fi
}

check_sni() {
    # StatusNotifierWatcher is registered by the DE/panel when it supports SNI.
    if command -v dbus-send >/dev/null 2>&1; then
        if dbus-send --session --print-reply \
                --dest=org.kde.StatusNotifierWatcher \
                /StatusNotifierWatcher \
                org.freedesktop.DBus.Introspectable.Introspect >/dev/null 2>&1; then
            success "StatusNotifierItem-compatible taskbar detected."
            return 0
        fi
    fi

    warn "No StatusNotifierItem-compatible taskbar detected on the session bus."
    echo ""
    echo "  iptw uses the StatusNotifierItem (SNI) D-Bus protocol for the tray icon."
    echo "  Most desktop environments support it natively:"
    echo ""
    echo "    KDE Plasma, XFCE, LXDE, Cinnamon, MATE  -- works out of the box"
    echo "    GNOME -- install the AppIndicator extension then log out/in:"
    echo "             https://extensions.gnome.org/extension/615/"
    echo ""
    warn "Map and wallpaper features still work without a tray icon."
}

verify_binary() {
    local binary="${1:-}"
    [[ -z "$binary" ]] && return 0
    if [[ ! -x "$binary" ]]; then
        warn "Binary not found or not executable: $binary"
        return 0
    fi
    info "Checking binary: $binary"
    # A CGO_ENABLED=0 Go binary should reference only the kernel vDSO.
    local unexpected
    unexpected=$(ldd "$binary" 2>&1 | grep -v -E "linux-vdso|ld-linux|ldd|statically" || true)
    if [[ -n "$unexpected" ]]; then
        warn "Unexpected native library references:"
        echo "$unexpected" | sed 's/^/    /'
    else
        success "Binary has no unexpected native library dependencies."
    fi
}

# --- main ---------------------------------------------------------------------

main() {
    local binary="${1:-}"

    echo ""
    echo -e "${BOLD}iptw -- Linux Environment Check${NC}"
    echo "================================="
    echo ""
    echo "  iptw is a pure-Go binary -- no GTK or AppIndicator libraries required."
    echo ""

    check_dbus || true
    echo ""
    check_sni  || true
    echo ""
    verify_binary "$binary"

    echo ""
    success "Done."
    echo ""
}

# Optionally pass the path to the iptw binary so the script can verify it:
#   ./linux-install-deps.sh ./iptw
main "${1:-}"
