#!/usr/bin/env bash
# Install master-agent as a system service under /opt/master-agent (Docker + systemd).
#
# Usage:
#   sudo ./deploy/install.sh [VERSION]
#
# VERSION defaults to git describe --tags --always from the repo root.

set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Run as root: sudo $0" >&2
	exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INSTALL_DIR="/opt/master-agent"
SERVICE_NAME="master-agent.service"

resolve_version() {
	if [[ $# -ge 1 && -n "${1:-}" ]]; then
		echo "$1"
		return
	fi
	git -C "${REPO_ROOT}" describe --tags --always 2>/dev/null || date +%Y%m%d
}

VERSION="$(resolve_version "${1:-}")"
IMAGE="master-agent:${VERSION}"

echo "==> Building ${IMAGE} from ${REPO_ROOT}"
docker build -t "${IMAGE}" "${REPO_ROOT}"

echo "==> Preparing ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/secrets/projects" "${INSTALL_DIR}/ssh"

if [[ ! -f "${INSTALL_DIR}/ssh/known_hosts" ]]; then
	touch "${INSTALL_DIR}/ssh/known_hosts"
	chmod 644 "${INSTALL_DIR}/ssh/known_hosts"
fi

cp "${SCRIPT_DIR}/docker-compose.prod.yml" "${INSTALL_DIR}/docker-compose.yml"

ADMIN_PASSWORD="$(openssl rand -hex 16 2>/dev/null || head -c 16 /dev/urandom | xxd -p -c 32)"
SESSION_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 64)"
cat >"${INSTALL_DIR}/.env" <<EOF
VERSION=${VERSION}
TICK_INTERVAL=30s
ADMIN_USERNAME=admin
ADMIN_PASSWORD=${ADMIN_PASSWORD}
SESSION_SECRET=${SESSION_SECRET}
EOF
chmod 600 "${INSTALL_DIR}/.env"

echo "${VERSION}" >"${INSTALL_DIR}/VERSION"

cp "${SCRIPT_DIR}/master-agent.service" "/etc/systemd/system/${SERVICE_NAME}"
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"

echo ""
echo "Installed master-agent ${VERSION}"
echo "  Web UI:  http://$(hostname -I 2>/dev/null | awk '{print $1}'):9080"
echo "  Local:   http://127.0.0.1:9080"
echo "  Login:   admin / ${ADMIN_PASSWORD}"
echo "  Config:  ${INSTALL_DIR}/.env"
echo ""
echo "Manage: systemctl status ${SERVICE_NAME}"
echo "Logs:   docker logs master-agent-prod-master-agent-1"
echo "Upgrade: sudo ${REPO_ROOT}/deploy/upgrade.sh <new-version>"
