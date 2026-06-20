#!/usr/bin/env bash
set -euo pipefail

TARGET_ROOT="${1:-/data/cloudiac-offline-deps}"
GIT_REMOTE_BASE="${GIT_REMOTE_BASE:-${2:-}}"
GIT_REMOTE_NAME="${GIT_REMOTE_NAME:-local}"
DRY_RUN="${DRY_RUN:-false}"

usage() {
  cat <<EOF
Usage:
  GIT_REMOTE_BASE=http://gitlab.local/iac-mirrors bash deploy/offline-mirror/push-git-mirrors.sh /data/cloudiac-offline-deps

Environment:
  GIT_REMOTE_BASE  Required. Base URL of the local GitLab group.
  GIT_REMOTE_NAME  Optional. Remote name used inside each bare repository. Default: local.
  DRY_RUN          Optional. Set to true to print planned pushes only.

Notes:
  GitLab projects/groups must already exist unless your GitLab allows push-to-create.
EOF
}

if [[ -z "${GIT_REMOTE_BASE}" ]]; then
  usage >&2
  exit 64
fi

if [[ ! -d "${TARGET_ROOT}/git" ]]; then
  echo "git mirror directory not found: ${TARGET_ROOT}/git" >&2
  exit 66
fi

find "${TARGET_ROOT}/git" -type d -name '*.git' | sort | while IFS= read -r repo_dir; do
  repo_rel="${repo_dir#"${TARGET_ROOT}/git/"}"
  remote_url="${GIT_REMOTE_BASE%/}/${repo_rel}"

  echo "sync git mirror: ${repo_rel} -> ${remote_url}"
  if [[ "${DRY_RUN}" == "true" ]]; then
    continue
  fi

  if git -C "${repo_dir}" remote get-url "${GIT_REMOTE_NAME}" >/dev/null 2>&1; then
    git -C "${repo_dir}" remote set-url "${GIT_REMOTE_NAME}" "${remote_url}"
  else
    git -C "${repo_dir}" remote add "${GIT_REMOTE_NAME}" "${remote_url}"
  fi
  git -C "${repo_dir}" push --mirror "${GIT_REMOTE_NAME}"
done

