#!/bin/sh
set -eu
mkdir -p /home/worker/workspace
chown -R worker:worker /home/worker/workspace
exec /usr/sbin/sshd -D -e
