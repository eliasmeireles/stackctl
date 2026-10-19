#!/bin/sh
# Disables SSH password authentication, keeping only public key access.
# Runs on the remote server; uses sudo when not executed as root.
set -e

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  SUDO="sudo"
fi

$SUDO sh -s <<'ROOT'
set -e
MAIN=/etc/ssh/sshd_config
DROPIN_DIR=/etc/ssh/sshd_config.d
DROPIN="$DROPIN_DIR/00-stackctl-disable-password.conf"

if [ -d "$DROPIN_DIR" ] && grep -qiE '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config\.d' "$MAIN"; then
  printf 'PasswordAuthentication no\nKbdInteractiveAuthentication no\nChallengeResponseAuthentication no\n' > "$DROPIN"
  chmod 644 "$DROPIN"
else
  cp "$MAIN" "$MAIN.stackctl.bak"
  for key in PasswordAuthentication KbdInteractiveAuthentication ChallengeResponseAuthentication; do
    sed -i -E "s/^#?[[:space:]]*$key[[:space:]].*/$key no/I" "$MAIN"
    grep -qiE "^$key no" "$MAIN" || echo "$key no" >> "$MAIN"
  done
fi

sshd -t
systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null || service ssh reload 2>/dev/null || service sshd reload

echo "sshd effective setting: $(sshd -T 2>/dev/null | grep -i '^passwordauthentication' || echo unknown)"
ROOT
