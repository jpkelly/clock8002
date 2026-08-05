#!/bin/sh
# Install SSH public keys from the FAT boot partition into the login user's
# ~/.ssh/authorized_keys. The Trixie installer had no equivalent, so the
# documented boot-partition key drop never actually worked on this platform
# (see README "authorized_keys").
set -eu

USER_NAME="${1:-pi}"
SRC="/boot/firmware/piclock/authorized_keys"

[ -f "$SRC" ] || exit 0

USER_HOME="$(getent passwd "$USER_NAME" | cut -d: -f6)"
[ -n "$USER_HOME" ] || exit 0
[ -d "$USER_HOME" ] || exit 0

DEST="${USER_HOME}/.ssh/authorized_keys"

install -d -m 700 -o "$USER_NAME" -g "$USER_NAME" "${USER_HOME}/.ssh"

# Merge rather than overwrite. Replacing the file outright with a boot-partition
# copy that omits the operator's current key locks them out of the unit, and on
# 2026-08-02 that is exactly what happened. Union means a key added on the FAT
# partition always works, at the cost of not being able to revoke one by
# deleting it there.
TMP="$(mktemp)"
cat "$DEST" "$SRC" 2>/dev/null | grep -v '^[[:space:]]*$' | sort -u >"$TMP"
install -m 600 -o "$USER_NAME" -g "$USER_NAME" "$TMP" "$DEST"
rm -f "$TMP"

echo "piclock: installed keys from ${SRC} into ${DEST}"
