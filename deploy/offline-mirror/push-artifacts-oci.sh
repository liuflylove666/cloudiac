#!/usr/bin/env bash
set -euo pipefail

TARGET_ROOT="${1:-/data/cloudiac-offline-deps}"
TERRASCAN_OCI_REF="${TERRASCAN_OCI_REF:-${2:-}}"
TERRASCAN_VERSION="${TERRASCAN_VERSION:-1.9.0}"
DRY_RUN="${DRY_RUN:-false}"

TERRASCAN_FILE="${TARGET_ROOT}/artifacts/terrascan/${TERRASCAN_VERSION}/terrascan_${TERRASCAN_VERSION}_Linux_x86_64.tar.gz"

usage() {
  cat <<EOF
Usage:
  TERRASCAN_OCI_REF=harbor.local/cloudiac-offline/terrascan:1.9.0 bash deploy/offline-mirror/push-artifacts-oci.sh /data/cloudiac-offline-deps

Environment:
  TERRASCAN_OCI_REF  Required. OCI artifact reference for the terrascan archive.
  TERRASCAN_VERSION  Optional. Default: 1.9.0.
  DRY_RUN            Optional. Set to true to print planned pushes only.

Prerequisites:
  Install oras and login to Harbor or another OCI registry before running this script.
EOF
}

if [[ -z "${TERRASCAN_OCI_REF}" ]]; then
  usage >&2
  exit 64
fi

if [[ ! -f "${TERRASCAN_FILE}" ]]; then
  echo "artifact not found: ${TERRASCAN_FILE}" >&2
  exit 66
fi

echo "sync artifact: ${TERRASCAN_FILE} -> ${TERRASCAN_OCI_REF}"
if [[ "${DRY_RUN}" == "true" ]]; then
  exit 0
fi

if ! command -v oras >/dev/null 2>&1; then
  echo "oras is required to push OCI artifacts" >&2
  exit 69
fi

oras push "${TERRASCAN_OCI_REF}" "${TERRASCAN_FILE}"

