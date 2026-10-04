#!/usr/bin/env bash
# Adds (or removes, with --remove) the platform root CA to the system trust store.
set -euo pipefail

CA_FILE="${1:?usage: trust-ca.sh <ca.crt> [--remove]}"
ACTION="${2:-add}"
CA_NAME="paved-road root CA"

is_wsl() { grep -qi microsoft /proc/version 2>/dev/null; }

case "$(uname -s)" in
  Darwin)
    if [[ "$ACTION" == "--remove" ]]; then
      sudo security delete-certificate -c "$CA_NAME" /Library/Keychains/System.keychain
    else
      sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$CA_FILE"
    fi
    ;;
  Linux)
    if [[ "$ACTION" == "--remove" ]]; then
      sudo rm -f /usr/local/share/ca-certificates/paved-road.crt
      sudo update-ca-certificates --fresh
    else
      sudo cp "$CA_FILE" /usr/local/share/ca-certificates/paved-road.crt
      sudo update-ca-certificates
    fi
    # Browsers on WSL2 run on Windows: trust the CA in the Windows user store too
    # (no admin rights needed for the current-user store).
    if is_wsl; then
      if [[ "$ACTION" == "--remove" ]]; then
        certutil.exe -delstore -user Root "$CA_NAME" || true
      else
        certutil.exe -addstore -user Root "$(wslpath -w "$CA_FILE")"
      fi
    fi
    echo "Note: Firefox keeps its own trust store; import the CA there if you use it."
    ;;
  *)
    echo "Unsupported system: $(uname -s)" >&2
    exit 1
    ;;
esac
