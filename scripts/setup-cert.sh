#!/usr/bin/env bash
# ============================================================================
# setup-cert.sh — provision PiCode's TLS certificate from a local mkcert CA
# and install trust everywhere the environment needs it. (Ported from the
# agentdeck-proven script; see ADR-0007.)
#
# Environment-aware; safe to re-run any time (renews the cert):
#   1. ensures mkcert exists (package manager or GitHub release binary)
#   2. installs the local CA into this system's trust store (sudo; skippable)
#   3. discovers every relevant name/IP: localhost + LAN + tailscale
#      (docker/link-local bridges excluded)
#   4. issues data/cert.pem + data/key.pem covering those SANs
#   5. WSL only: exports the CA to Windows and imports it into the Windows
#      machine trust store — an UAC prompt appears; approve it once
#   6. restarts the picode systemd user service when present
#   7. prints iOS trust instructions (or serve+QR with --ios)
#
# Usage:
#   scripts/setup-cert.sh [--check <days>] [--ios]
# ============================================================================

set -euo pipefail
cd "$(dirname "$0")/.."
ROOT="$PWD"

DATA_DIR="${PICODE_DATA:-$HOME/.picode}"

bold() { printf '\033[1m%s\033[0m\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*"; }
die()  { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

IOS_MODE=0
CHECK_DAYS=0
if [[ "${1:-}" == "--check" ]]; then
    [[ "${2:-}" =~ ^[0-9]+$ ]] || { echo "usage: setup-cert.sh [--check <days>] [--ios]" >&2; exit 1; }
    CHECK_DAYS=$2; shift 2
fi
[[ "${1:-}" == "--ios" ]] && IOS_MODE=1

# Early exit: certificate healthy and far from expiry? Nothing to do.
if [[ $CHECK_DAYS -gt 0 && -f "$DATA_DIR/cert.pem" ]]; then
    if openssl x509 -checkend $((CHECK_DAYS * 86400)) -noout -in "$DATA_DIR/cert.pem" 2>/dev/null; then
        ok "certificate valid for more than $CHECK_DAYS days — nothing to do"
        exit 0
    fi
fi

# Windows PowerShell via full path (WSL interop PATH may be disabled)
PWSH=""
if command -v powershell.exe &>/dev/null; then
    PWSH="powershell.exe"
elif [[ -x /mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe ]]; then
    PWSH=/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe
fi

# --------------------------------------------------------------------------
bold "[1/6] mkcert"
if ! command -v mkcert &>/dev/null; then
    arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac
    url="https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-linux-${arch}"
    mkdir -p "$HOME/.local/bin"
    curl -fsSL "$url" -o "$HOME/.local/bin/mkcert" || die "download failed: $url"
    chmod +x "$HOME/.local/bin/mkcert"
    ok "installed to ~/.local/bin/mkcert (v1.4.4, $arch)"
else
    ok "found: $(command -v mkcert) ($(mkcert -version 2>/dev/null || echo ?))"
fi

# --------------------------------------------------------------------------
bold "[2/6] local CA -> this system's trust store"
if mkcert -install 2>/dev/null; then
    ok "CA trusted system-wide (Linux side)"
else
    warn "mkcert -install needs sudo (interactive) — skipped."
    warn "Linux-side browsers will still warn; Windows/iOS trust is what matters."
fi

# --------------------------------------------------------------------------
bold "[3/6] discovering names/IPs for this machine"
SAN=("localhost" "picode.local")
add_ip() {
    [[ -n "$1" ]] || return 0
    for s in "${SAN[@]}"; do [[ "$s" == "$1" ]] && return 0; done
    SAN+=("$1")
}
# tailscale first (explicit, when available)
if command -v tailscale &>/dev/null; then
    add_ip "$(tailscale ip -4 2>/dev/null || true)"
fi
# every interface IPv4, excluding loopback / link-local / docker bridges
for ip in $(hostname -I 2>/dev/null); do
    [[ "$ip" == *:* ]] && continue # IPv6: browsers hit the v4 names
    case "$ip" in
        127.*|169.254.*) continue ;;
        172.1[6-9].*|172.2[0-9].*|172.3[01].*) continue ;; # docker bridge range
        *) add_ip "$ip" ;;
    esac
done
ok "SANs: ${SAN[*]}"

# --------------------------------------------------------------------------
bold "[4/6] issuing certificate ($DATA_DIR/cert.pem, $DATA_DIR/key.pem)"
mkdir -p "$DATA_DIR"
CAROOT_VAL="$(mkcert -CAROOT)"
mkcert -cert-file "$DATA_DIR/cert.pem" -key-file "$DATA_DIR/key.pem" \
    "${SAN[@]}" >/dev/null 2>&1 \
    || die "mkcert could not issue the certificate"
ok "issued by local CA: $CAROOT_VAL/rootCA.pem"

# --------------------------------------------------------------------------
bold "[5/6] Windows trust store (WSL detected?)"
IS_WSL=0
grep -qi microsoft /proc/version 2>/dev/null && IS_WSL=1
[[ -n "${WSL_DISTRO_NAME:-}" ]] && IS_WSL=1

if [[ $IS_WSL -eq 1 && -n "$PWSH" ]]; then
    win_user="$($PWSH -NoProfile -Command '$env:USERNAME' 2>/dev/null | tr -d '\r\n')"
    [[ -n "$win_user" ]] || win_user="$(ls /mnt/c/Users 2>/dev/null | grep -viE 'public|default|all users|desktop.ini' | head -1)"
    if [[ -n "${win_user:-}" ]]; then
        ca_win="/mnt/c/Users/$win_user/picode-mkcert-rootCA.cer"
        cp "$CAROOT_VAL/rootCA.pem" "$ca_win"
        ok "CA exported to C:\\Users\\$win_user\\picode-mkcert-rootCA.cer"
        # already trusted? (reading LocalMachine\Root needs no admin)
        already="$($PWSH -NoProfile -Command \
            "(Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like '*mkcert*').Count" 2>/dev/null | tr -d '\r\n ' || echo 0)"
        if [[ "${already:-0}" -ge 1 ]]; then
            ok "a mkcert CA is already trusted on Windows"
        else
            bold "  → UAC prompt incoming on Windows: approve to import the CA"
            "$PWSH" -NoProfile -Command \
                "Start-Process powershell -Verb RunAs -Wait -ArgumentList '-NoProfile','-Command','Import-Certificate -FilePath C:\\Users\\$win_user\\picode-mkcert-rootCA.cer -CertStoreLocation Cert:\LocalMachine\Root'" \
                2>/dev/null || true
            verify="$($PWSH -NoProfile -Command \
                "(Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -like '*mkcert*').Count" 2>/dev/null | tr -d '\r\n ' || echo 0)"
            [[ "${verify:-0}" -ge 1 ]] && ok "CA imported into Windows trust store" \
                                     || warn "import not confirmed — run manually as Admin: Import-Certificate -FilePath C:\\Users\\$win_user\\picode-mkcert-rootCA.cer -CertStoreLocation Cert:\LocalMachine\Root"
        fi
        warn "close ALL Chrome/Edge windows and reopen: CAs are read at browser boot."
    else
        warn "could not detect the Windows user — import manually (Admin):"
        warn "  Import-Certificate -FilePath <CA path> -CertStoreLocation Cert:\LocalMachine\Root"
    fi
elif [[ $IS_WSL -eq 1 ]]; then
    warn "Windows PowerShell not reachable from WSL — import manually (Admin):"
    warn "  Import-Certificate -FilePath <CA path> -CertStoreLocation Cert:\LocalMachine\Root"
else
    ok "not WSL — nothing to do on the Windows side"
fi

# --------------------------------------------------------------------------
bold "[6/6] restarting PiCode"
if systemctl --user is-active --quiet picode 2>/dev/null; then
    systemctl --user restart picode
    ok "systemd user service restarted"
elif [[ -f "$DATA_DIR/server.json" ]]; then
    port=$(python3 -c "import json;print(json.load(open('$DATA_DIR/server.json'))['port'])" 2>/dev/null || echo "?")
    if [[ -n "${PICODE_PID:-}" ]] && kill -0 "$PICODE_PID" 2>/dev/null; then
        kill -HUP "$PICODE_PID" 2>/dev/null && ok "signaled picode (pid $PICODE_PID)" \
                                              || warn "could not signal — restart picode manually to pick up the new cert"
    else
        warn "picode should pick the cert up on restart — currently on port $port"
    fi
else
    ok "picode not running — cert will be used on next start"
fi

# --------------------------------------------------------------------------
bold "iOS (optional, manual on the phone)"
if [[ $IOS_MODE -eq 1 ]]; then
    ts_ip="$(tailscale ip -4 2>/dev/null || true)"
    [[ -n "$ts_ip" ]] || ts_ip="$(hostname -I | awk '{print $1}')"
    port=8443
    (cd "$CAROOT_VAL" && python3 -m http.server "$port" --bind 0.0.0.0 >/dev/null 2>&1 &)
    sleep 1
    ok "serving rootCA.pem on http://$ts_ip:$port for 10 minutes (iOS: download → install profile → enable trust)"
    if python3 -c "import qrcode" 2>/dev/null; then
        python3 - "$ts_ip" "$port" << 'PYEOF'
import qrcode, sys
qr = qrcode.QRCode(border=2)
qr.add_data(f"http://{sys.argv[1]}:{sys.argv[2]}/rootCA.pem")
qr.make(fit=True)
qr.print_ascii(invert=True)
PYEOF
    else
        warn "install 'qrcode' (pip) for a scannable QR"
    fi
    (sleep 600; pkill -f "http.server $port" 2>/dev/null) &
else
    cat <<EOF
  1. Get rootCA.pem onto the phone ($CAROOT_VAL/rootCA.pem)
  2. Settings → Profile Downloaded → Install
  3. Settings → General → About → Certificate Trust Settings → enable
  (re-run with --ios to serve it on your tailnet with a QR code)
EOF
fi

port_now=$(python3 -c "import json;print(json.load(open('$DATA_DIR/server.json'))['url'])" 2>/dev/null || echo "https://localhost:<port>")
bold "done — $port_now (SANs: ${SAN[*]})"
