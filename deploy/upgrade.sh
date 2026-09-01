#!/usr/bin/env bash
# Upgrade the system master-agent to a new pinned image version.
#
# Usage:
#   sudo ./deploy/upgrade.sh <VERSION> [--ref GIT_REF]
#
# Builds master-agent:VERSION from the repo (optionally at GIT_REF), updates
# /opt/master-agent/.env, and reloads the systemd unit.

set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Run as root: sudo $0" >&2
	exit 1
fi

if [[ $# -lt 1 || -z "${1:-}" ]]; then
	echo "Usage: sudo $0 <VERSION> [--ref GIT_REF]" >&2
	exit 1
fi

NEW_VERSION="$1"
shift

GIT_REF=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	--ref)
		GIT_REF="${2:?--ref requires a value}"
		shift 2
		;;
	*)
		echo "Unknown argument: $1" >&2
		exit 1
		;;
	esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INSTALL_DIR="/opt/master-agent"
SERVICE_NAME="master-agent.service"
IMAGE="master-agent:${NEW_VERSION}"

if [[ ! -d "${INSTALL_DIR}" ]]; then
	echo "Not installed — run deploy/install.sh first" >&2
	exit 1
fi

BUILD_DIR="${REPO_ROOT}"
if [[ -n "${GIT_REF}" ]]; then
	BUILD_DIR="$(mktemp -d)"
	trap 'rm -rf "${BUILD_DIR}"' EXIT
	echo "==> Checking out ${GIT_REF} to ${BUILD_DIR}"
	git -C "${REPO_ROOT}" archive "${GIT_REF}" | tar -x -C "${BUILD_DIR}"
fi

echo "==> Building ${IMAGE}"
docker build -t "${IMAGE}" "${BUILD_DIR}"

OLD_VERSION=""
if [[ -f "${INSTALL_DIR}/VERSION" ]]; then
	OLD_VERSION="$(cat "${INSTALL_DIR}/VERSION")"
fi

ENV_FILE="${INSTALL_DIR}/.env"
if [[ -f "${ENV_FILE}" ]]; then
	# Preserve runtime credentials; update VERSION.
	TICK_INTERVAL="$(grep -E '^TICK_INTERVAL=' "${ENV_FILE}" | cut -d= -f2- || true)"
	ADMIN_USERNAME="$(grep -E '^ADMIN_USERNAME=' "${ENV_FILE}" | cut -d= -f2- || true)"
	ADMIN_PASSWORD="$(grep -E '^ADMIN_PASSWORD=' "${ENV_FILE}" | cut -d= -f2- || true)"
	SESSION_SECRET="$(grep -E '^SESSION_SECRET=' "${ENV_FILE}" | cut -d= -f2- || true)"
	TOKEN="$(grep -E '^MASTER_AGENT_TOKEN=' "${ENV_FILE}" | cut -d= -f2- || true)"
	[[ -z "${TICK_INTERVAL}" ]] && TICK_INTERVAL="30s"
	[[ -z "${ADMIN_USERNAME}" ]] && ADMIN_USERNAME="admin"
	[[ -z "${ADMIN_PASSWORD}" ]] && ADMIN_PASSWORD="$(openssl rand -hex 16 2>/dev/null || head -c 16 /dev/urandom | xxd -p -c 32)"
	[[ -z "${SESSION_SECRET}" ]] && SESSION_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 64)"
else
	TICK_INTERVAL="30s"
	ADMIN_USERNAME="admin"
	ADMIN_PASSWORD="$(openssl rand -hex 16 2>/dev/null || head -c 16 /dev/urandom | xxd -p -c 32)"
	SESSION_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p -c 64)"
	TOKEN=""
fi

cat >"${ENV_FILE}" <<EOF
VERSION=${NEW_VERSION}
TICK_INTERVAL=${TICK_INTERVAL}
ADMIN_USERNAME=${ADMIN_USERNAME}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
SESSION_SECRET=${SESSION_SECRET}
MASTER_AGENT_TOKEN=${TOKEN}
EOF
chmod 600 "${ENV_FILE}"

echo "${NEW_VERSION}" >"${INSTALL_DIR}/VERSION"

cp "${SCRIPT_DIR}/docker-compose.prod.yml" "${INSTALL_DIR}/docker-compose.yml"

echo "==> Reloading ${SERVICE_NAME} (${OLD_VERSION:-none} -> ${NEW_VERSION})"
systemctl daemon-reload
systemctl reload "${SERVICE_NAME}"

echo ""
echo "Upgraded to master-agent ${NEW_VERSION}"
echo "  Web UI: http://127.0.0.1:9080"
echo "Rollback: edit ${INSTALL_DIR}/.env VERSION, then systemctl reload ${SERVICE_NAME}"
