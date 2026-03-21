#!/usr/bin/env bash
# Deploy pa-runner to a node: create restricted user, install binary, allowlist, and SSH config (ForceCommand).
# Run on the machine where PA runs (or where you build pa-runner). Connects to the node as admin (password prompt).
#
# Usage:
#   ./scripts/deploy-pa-runner.sh ADMIN_USER HOST DEDICATED_USER PA_PUBKEY_PATH PA_RUNNER_BINARY [ALLOWLIST_PATH]
#
# Example:
#   ./scripts/deploy-pa-runner.sh root nas.local pa-runner ~/.ssh/pa_nas.pub ./pa-runner
#   ./scripts/deploy-pa-runner.sh admin myserver.local pa ~/.ssh/pa.pub ./pa-runner ./config/nas_allowlist
#
# You will be prompted for ADMIN_USER password when connecting.
# Requires: admin can sudo on the node; node is Linux (Debian/Ubuntu or Synology DSM).
# After deploy: PA config must use this node with dedicated_user=DEDICATED_USER and private key matching PA_PUBKEY_PATH.

set -e

if [ $# -lt 5 ]; then
  echo "Usage: $0 ADMIN_USER HOST DEDICATED_USER PA_PUBKEY_PATH PA_RUNNER_BINARY [ALLOWLIST_PATH]" >&2
  echo "  ADMIN_USER       — user to connect as (e.g. root); will be prompted for password." >&2
  echo "  HOST             — node hostname or IP." >&2
  echo "  DEDICATED_USER   — restricted user created on node (e.g. pa-runner); PA will connect as this user." >&2
  echo "  PA_PUBKEY_PATH   — path to PA server's public key (installed to dedicated user's authorized_keys)." >&2
  echo "  PA_RUNNER_BINARY — path to pa-runner binary (built e.g. with go build -o pa-runner ./cmd/pa-runner)." >&2
  echo "  ALLOWLIST_PATH   — optional: local file with allowed commands (one per line); else a sample allowlist is created." >&2
  exit 1
fi

ADMIN_USER="$1"
HOST="$2"
DEDICATED_USER="$3"
PA_PUBKEY_PATH="$4"
PA_RUNNER_BINARY="$5"
ALLOWLIST_PATH="${6:-}"

REMOTE_ADMIN="${ADMIN_USER}@${HOST}"
PA_RUNNER_REMOTE="/usr/local/bin/pa-runner"
PA_RUNNER_CONF_DIR="/etc/pa-runner"
ALLOWLIST_REMOTE="${PA_RUNNER_CONF_DIR}/allowlist.txt"

if [ ! -f "$PA_PUBKEY_PATH" ]; then
  echo "Error: PA public key not found: $PA_PUBKEY_PATH" >&2
  exit 1
fi

if [ ! -f "$PA_RUNNER_BINARY" ]; then
  echo "Error: pa-runner binary not found: $PA_RUNNER_BINARY" >&2
  echo "Build with: go build -o pa-runner ./cmd/pa-runner  (when cmd/pa-runner exists)" >&2
  exit 1
fi

if [ -n "$ALLOWLIST_PATH" ] && [ ! -f "$ALLOWLIST_PATH" ]; then
  echo "Error: allowlist file not found: $ALLOWLIST_PATH" >&2
  exit 1
fi

PUBKEY_B64=$(base64 < "$PA_PUBKEY_PATH" | tr -d '\n')

echo "Deploying pa-runner to $REMOTE_ADMIN"
echo "  Dedicated user: $DEDICATED_USER"
echo "  Binary: $PA_RUNNER_BINARY -> $PA_RUNNER_REMOTE"
echo "  Allowlist: ${ALLOWLIST_PATH:-'(sample)'} -> $ALLOWLIST_REMOTE"
echo "Connecting (enter $ADMIN_USER password when prompted)..."
echo ""

# --- 1. Create dedicated user and .ssh (reuse logic from ssh-create-remote-user.sh) ---
ssh -t -o BatchMode=no -o StrictHostKeyChecking=accept-new -- "$REMOTE_ADMIN" "DEDICATED_USER='$DEDICATED_USER'; PUBKEY_B64='$PUBKEY_B64';
  set -e
  if id -u \"\$DEDICATED_USER\" &>/dev/null; then
    echo \"User \$DEDICATED_USER already exists.\"
  else
    if [ -f /usr/syno/sbin/synouser ]; then
      echo \"Synology DSM: creating user \$DEDICATED_USER\"
      sudo /usr/syno/sbin/synouser --add \"\$DEDICATED_USER\" '' \"\" 0 '' 0 || true
      HOME_DIR=\$(getent passwd \"\$DEDICATED_USER\" 2>/dev/null | cut -d: -f6) || HOME_DIR=\"/var/services/homes/\${DEDICATED_USER}\"
    else
      echo \"Creating user \$DEDICATED_USER (no password, key-only)\"
      sudo adduser --disabled-password --gecos '' \"\$DEDICATED_USER\" || true
      HOME_DIR=\"/home/\${DEDICATED_USER}\"
    fi
  fi
  HOME_DIR=\${HOME_DIR:-\$(getent passwd \"\$DEDICATED_USER\" 2>/dev/null | cut -d: -f6)}
  [ -z \"\$HOME_DIR\" ] && HOME_DIR=\"/home/\${DEDICATED_USER}\"
  sudo mkdir -p \"\${HOME_DIR}/.ssh\"
  echo \"\$PUBKEY_B64\" | base64 -d | sudo tee \"\${HOME_DIR}/.ssh/authorized_keys\" > /dev/null
  sudo chmod 700 \"\${HOME_DIR}/.ssh\"
  sudo chmod 600 \"\${HOME_DIR}/.ssh/authorized_keys\"
  sudo chown -R \"\${DEDICATED_USER}:\" \"\${HOME_DIR}/.ssh\"
  echo \"Key installed for \$DEDICATED_USER.\"
"

# --- 2. Create /etc/pa-runner and allowlist ---
if [ -n "$ALLOWLIST_PATH" ]; then
  ALLOWLIST_B64=$(base64 < "$ALLOWLIST_PATH" | tr -d '\n')
  ssh -o BatchMode=no -o StrictHostKeyChecking=accept-new -- "$REMOTE_ADMIN" "ALLOWLIST_B64='$ALLOWLIST_B64'; ALLOWLIST_REMOTE='$ALLOWLIST_REMOTE';
    set -e
    sudo mkdir -p $(dirname '$ALLOWLIST_REMOTE')
    echo \"\$ALLOWLIST_B64\" | base64 -d | sudo tee '$ALLOWLIST_REMOTE' > /dev/null
    sudo chmod 644 '$ALLOWLIST_REMOTE'
    echo \"Allowlist installed: $ALLOWLIST_REMOTE\"
  "
else
  ssh -o BatchMode=no -o StrictHostKeyChecking=accept-new -- "$REMOTE_ADMIN" "ALLOWLIST_REMOTE='$ALLOWLIST_REMOTE';
    set -e
    sudo mkdir -p $(dirname '$ALLOWLIST_REMOTE')
    echo 'date -Iseconds' | sudo tee '$ALLOWLIST_REMOTE' > /dev/null
    sudo chmod 644 '$ALLOWLIST_REMOTE'
    echo \"Sample allowlist created: $ALLOWLIST_REMOTE (one line: date -Iseconds)\"
  "
fi

# --- 3. Copy pa-runner binary (single ssh, no second password for scp) ---
echo "Uploading pa-runner binary..."
base64 < "$PA_RUNNER_BINARY" | ssh -o BatchMode=no -o StrictHostKeyChecking=accept-new -- "$REMOTE_ADMIN" "
  set -e
  sudo base64 -d > '$PA_RUNNER_REMOTE'
  sudo chmod 755 '$PA_RUNNER_REMOTE'
  echo \"Binary installed: $PA_RUNNER_REMOTE\"
"

# --- 4. Configure sshd: ForceCommand for dedicated user (idempotent: skip if already present) ---
ssh -o BatchMode=no -o StrictHostKeyChecking=accept-new -- "$REMOTE_ADMIN" "DEDICATED_USER='$DEDICATED_USER'; PA_RUNNER_REMOTE='$PA_RUNNER_REMOTE';
  set -e
  if sudo grep -q \"Match User \$DEDICATED_USER\" /etc/ssh/sshd_config 2>/dev/null; then
    echo \"sshd_config: Match User \$DEDICATED_USER already present, skipping.\"
  else
    sudo tee -a /etc/ssh/sshd_config << SSHD_EOF

# PA runner (restricted user, exec-only)
Match User \$DEDICATED_USER
    ForceCommand $PA_RUNNER_REMOTE
    AllowTcpForwarding no
SSHD_EOF
    echo \"sshd_config: appended Match User \$DEDICATED_USER with ForceCommand.\"
    if command -v systemctl &>/dev/null; then
      sudo systemctl restart sshd 2>/dev/null || sudo systemctl restart ssh 2>/dev/null || true
    fi
    echo \"Restart sshd manually if needed (e.g. systemctl restart sshd).\"
  fi
"

echo ""
echo "Done. Next steps:"
echo "  1. Ensure sshd was restarted (script may have run systemctl restart sshd)."
echo "  2. In PA config (config.json): set nodes.<node_id>.host = $HOST, dedicated_user = $DEDICATED_USER, auth.private_key_path = path to key matching $PA_PUBKEY_PATH."
echo "  3. Add the same node_id to config and allowlist (nas_allowlist or node-specific) with the exact commands that appear in $ALLOWLIST_REMOTE."
echo "  4. Test: ssh -i <private_key> $DEDICATED_USER@$HOST 'date -Iseconds'  (should run via pa-runner and return output)."
