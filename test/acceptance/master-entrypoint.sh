#!/bin/sh
# Copy bind-mounted private keys to a mode OpenSSH accepts (bind mounts often look world-readable).
set -eu
mkdir -p /secrets/runtime
cp /secrets/id_ed25519 /secrets/runtime/id_ed25519
chmod 600 /secrets/runtime/id_ed25519
if [ -f /secrets/id_ed25519_b ]; then
	cp /secrets/id_ed25519_b /secrets/runtime/id_ed25519_b
	chmod 600 /secrets/runtime/id_ed25519_b
fi
exec "$@"
