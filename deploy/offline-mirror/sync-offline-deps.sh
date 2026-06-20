#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_ROOT="${1:-/data/cloudiac-offline-deps}"
ALLOW_PARTIAL_MIRROR="${ALLOW_PARTIAL_MIRROR:-false}"

set +e
bash "${SCRIPT_DIR}/mirror-github-deps.sh" "${TARGET_ROOT}"
mirror_rc=$?
set -e

if [[ "${mirror_rc}" -ne 0 ]]; then
  if [[ "${mirror_rc}" -eq 2 && "${ALLOW_PARTIAL_MIRROR}" == "true" ]]; then
    echo "mirror step completed with partial failures; continuing because ALLOW_PARTIAL_MIRROR=true" >&2
  else
    exit "${mirror_rc}"
  fi
fi

if [[ -n "${GIT_REMOTE_BASE:-}" ]]; then
  bash "${SCRIPT_DIR}/push-git-mirrors.sh" "${TARGET_ROOT}"
fi

if [[ -n "${TERRASCAN_OCI_REF:-}" ]]; then
  bash "${SCRIPT_DIR}/push-artifacts-oci.sh" "${TARGET_ROOT}"
fi

cat <<EOF
Offline dependency sync finished.
  target: ${TARGET_ROOT}
  git push: ${GIT_REMOTE_BASE:-skipped}
  artifact push: ${TERRASCAN_OCI_REF:-skipped}
EOF

