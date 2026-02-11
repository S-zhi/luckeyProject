#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DOC_PATH="${PROJECT_ROOT}/docs/SSH_KEY_SETUP.md"

log_info() {
  printf '[%s] [INFO] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

log_warn() {
  printf '[%s] [WARN] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
}

log_error() {
  printf '[%s] [ERROR] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
}

usage() {
  cat <<'EOF'
Usage:
  scripts/upload_ssh_key.sh --host <ip_or_domain> [--user <user>] [--port <port>] [--pubkey <path>] [--doc-only]

Options:
  --host      Remote server IP or domain (required unless --doc-only)
  --user      SSH login username (default: root)
  --port      SSH port (default: 22)
  --pubkey    Local public key path (default: ~/.ssh/id_rsa.pub)
  --doc-only  Only generate docs file, skip uploading key
  --help      Show this help

Examples:
  scripts/upload_ssh_key.sh --host 192.168.1.100 --user root
  scripts/upload_ssh_key.sh --host 10.0.0.7 --user ubuntu --port 22 --pubkey ~/.ssh/id_ed25519.pub
  scripts/upload_ssh_key.sh --doc-only
EOF
}

generate_doc() {
  mkdir -p "$(dirname "${DOC_PATH}")"
  cat > "${DOC_PATH}" <<'EOF'
# SSH Key Upload Usage

## 1. Generate key pair (if needed)
```bash
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
```

## 2. Upload local public key to remote server
```bash
scripts/upload_ssh_key.sh --host 192.168.1.100 --user root
```

Optional parameters:
- `--port`: SSH port, default `22`
- `--pubkey`: local public key path, default `~/.ssh/id_rsa.pub`
- `--doc-only`: only generate this doc file, skip upload step

## 3. Verify login with key auth
```bash
ssh -i ~/.ssh/id_rsa root@192.168.1.100
```

If it enters shell without password prompt, key auth is ready.

## 4. Common failures
- Public key file missing:
  - Run `ssh-keygen` first, or pass correct `--pubkey`
- Permission denied (publickey):
  - Confirm remote user is correct
  - Check `~/.ssh` and `~/.ssh/authorized_keys` permissions on remote host
- Network timeout:
  - Check IP/port and firewall
EOF
  log_info "usage doc generated: ${DOC_PATH}"
}

HOST=""
USER_NAME="root"
PORT="22"
PUBKEY_PATH="${HOME}/.ssh/id_rsa.pub"
DOC_ONLY="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      HOST="${2:-}"
      shift 2
      ;;
    --user)
      USER_NAME="${2:-}"
      shift 2
      ;;
    --port)
      PORT="${2:-}"
      shift 2
      ;;
    --pubkey)
      PUBKEY_PATH="${2:-}"
      shift 2
      ;;
    --doc-only)
      DOC_ONLY="1"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      log_error "unknown argument: $1"
      usage
      exit 1
      ;;
  esac
done

generate_doc

if [[ "${DOC_ONLY}" == "1" ]]; then
  log_info "doc-only mode enabled, skip key upload"
  exit 0
fi

if [[ -z "${HOST}" ]]; then
  log_error "--host is required"
  usage
  exit 1
fi

if [[ ! -f "${PUBKEY_PATH}" ]]; then
  log_error "public key file not found: ${PUBKEY_PATH}"
  exit 1
fi

TARGET="${USER_NAME}@${HOST}"
log_info "upload public key begin: target=${TARGET}, port=${PORT}, pubkey=${PUBKEY_PATH}"

if command -v ssh-copy-id >/dev/null 2>&1; then
  log_info "ssh-copy-id found, use ssh-copy-id"
  ssh-copy-id -i "${PUBKEY_PATH}" -p "${PORT}" "${TARGET}"
else
  log_warn "ssh-copy-id not found, fallback to manual append"
  KEY_CONTENT="$(cat "${PUBKEY_PATH}")"
  ssh -p "${PORT}" "${TARGET}" "umask 077; mkdir -p ~/.ssh; touch ~/.ssh/authorized_keys; grep -qxF '${KEY_CONTENT}' ~/.ssh/authorized_keys || echo '${KEY_CONTENT}' >> ~/.ssh/authorized_keys"
fi

log_info "upload public key success: target=${TARGET}"
