#!/usr/bin/env bash
# Run on LOCAL machine: generate SSH key (if missing), connect to server with password,
# create a new restricted user on the server, install your public key for passwordless login,
# then test SSH as the new user.
#
# Usage:
#   ./scripts/ssh-create-remote-user.sh ADMIN_USER HOST NEW_USER [KEY_PATH]
#
# Example:
#   ./scripts/ssh-create-remote-user.sh root nas.local pa
#   ./scripts/ssh-create-remote-user.sh admin myserver.local myuser ~/.ssh/myuser_server
#
# You will be prompted for the ADMIN_USER password when connecting to the server.
# The new user has no sudo, login only by key (password disabled).
# Server: auto-detects Synology DSM (synouser + /var/services/homes) or Debian/Ubuntu (adduser + /home).

set -e

if [ $# -lt 3 ]; then
  echo "Usage: $0 ADMIN_USER HOST NEW_USER [KEY_PATH]" >&2
  echo "Example: $0 root nas.local pa" >&2
  exit 1
fi

ADMIN_USER="$1"
HOST="$2"
NEW_USER="$3"
KEY_PATH="${4:-${HOME}/.ssh/${NEW_USER}_${HOST}}"
KEY_PUB="${KEY_PATH}.pub"
REMOTE_ADMIN="${ADMIN_USER}@${HOST}"

# --- 1–2: Ensure key pair exists on local machine ---
if [ ! -f "$KEY_PATH" ]; then
  echo "Generating SSH key: $KEY_PATH"
  mkdir -p "$(dirname "$KEY_PATH")"
  chmod 700 "$(dirname "$KEY_PATH")" 2>/dev/null || true
  ssh-keygen -t ed25519 -f "$KEY_PATH" -N "" -C "${NEW_USER}@${HOST}"
else
  echo "Using existing key: $KEY_PATH"
fi

if [ ! -f "$KEY_PUB" ]; then
  echo "Error: public key not found: $KEY_PUB" >&2
  exit 1
fi

# --- 3–7: Connect as admin and create user + install key (will prompt for admin password) ---
KEY_FP=$(ssh-keygen -lf "$KEY_PUB" 2>/dev/null | awk '{print $2}')
echo "Installing key: $KEY_PUB (fingerprint: ${KEY_FP:-unknown})"
echo "  → Should match 'Offering public key: ...' in ssh -v output."
echo "Connecting to $REMOTE_ADMIN (enter password when prompted)..."
PUBKEY_B64=$(base64 < "$KEY_PUB" | tr -d '\n')

# -t allocates a TTY so sudo can prompt for password when required.
# If admin uses key-based SSH and/or NOPASSWD sudo, no password is asked; -t is harmless then.
ssh -t -o BatchMode=no -o StrictHostKeyChecking=accept-new -- "$REMOTE_ADMIN" "NEW_USER='$NEW_USER'; PUBKEY_B64='$PUBKEY_B64';
  set -e
  if [ -f /usr/syno/sbin/synouser ]; then
    echo \"Synology DSM detected.\"
    if ! id -u \"\$NEW_USER\" &>/dev/null; then
      echo \"Creating user: \$NEW_USER (synouser, key-only)\"
      sudo /usr/syno/sbin/synouser --add \"\$NEW_USER\" '' \"\" 0 '' 0 || true
    else
      echo \"User \$NEW_USER already exists.\"
    fi
    if ! id -u \"\$NEW_USER\" &>/dev/null; then
      echo \"Error: user \$NEW_USER could not be created or found.\" >&2
      exit 1
    fi
    # Synology home is /var/services/homes/<user>. getent not available on DSM; fallback to grep /etc/passwd or default.
    HOME_DIR=\$(getent passwd \"\$NEW_USER\" 2>/dev/null | cut -d: -f6) || true
    [ -z \"\$HOME_DIR\" ] && HOME_DIR=\$(grep \"^\${NEW_USER}:\" /etc/passwd 2>/dev/null | cut -d: -f6) || true
    [ -z \"\$HOME_DIR\" ] && HOME_DIR=\"/var/services/homes/\${NEW_USER}\"
    sudo mkdir -p \"\${HOME_DIR}\"
    sudo chown \"\${NEW_USER}:\" \"\${HOME_DIR}\"
  else
    if ! id -u \"\$NEW_USER\" &>/dev/null; then
      echo \"Creating user: \$NEW_USER (no password, key-only login)\"
      sudo adduser --disabled-password --gecos '' \"\$NEW_USER\" || true
    else
      echo \"User \$NEW_USER already exists.\"
    fi
    HOME_DIR=\"/home/\${NEW_USER}\"
  fi
  sudo mkdir -p \"\${HOME_DIR}/.ssh\"
  echo \"\$PUBKEY_B64\" | base64 -d | sudo tee \"\${HOME_DIR}/.ssh/authorized_keys\" > /dev/null
  sudo chmod 700 \"\${HOME_DIR}/.ssh\"
  sudo chmod 600 \"\${HOME_DIR}/.ssh/authorized_keys\"
  # Use user: (primary group) so it works on Synology where group may be 'users' not same as username
  sudo chown -R \"\${NEW_USER}:\" \"\${HOME_DIR}/.ssh\"
  echo \"Key installed for \$NEW_USER.\"
  echo \"Verify: \${HOME_DIR}/.ssh/authorized_keys\"
  sudo ls -la \"\${HOME_DIR}/.ssh\"
  echo -n \"Fingerprint on server: \"
  sudo cat \"\${HOME_DIR}/.ssh/authorized_keys\" | ssh-keygen -lf - 2>/dev/null || echo \"(ssh-keygen not in PATH)\"
"

echo "Disconnected from $REMOTE_ADMIN."

# --- 8: Test SSH as new user ---
echo "Testing SSH as ${NEW_USER}@${HOST}..."
ssh -i "$KEY_PATH" -o BatchMode=yes -o ConnectTimeout=10 -o StrictHostKeyChecking=accept-new -- "${NEW_USER}@${HOST}" "echo 'Logged in as'; whoami; id"

# --- 9: Success ---
echo ""
echo "Success. User ${NEW_USER} on ${HOST} is ready."
echo "Connect with: ssh -i ${KEY_PATH} ${NEW_USER}@${HOST}"
