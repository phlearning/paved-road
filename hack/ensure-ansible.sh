#!/usr/bin/env bash
# Installs Ansible if it is missing, so `make bootstrap` works on a fresh machine.
set -euo pipefail

command -v ansible-playbook >/dev/null && exit 0

case "$(uname -s)" in
  Darwin)
    brew install ansible
    ;;
  Linux)
    sudo apt-get update
    sudo apt-get install -y pipx
    pipx install --include-deps ansible-core
    ;;
  *)
    echo "Unsupported system: $(uname -s)" >&2
    exit 1
    ;;
esac
