#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

TARGET_ROOT="${1:-/data/cloudiac-offline-deps}"
DEPS_FILE="${DEPS_FILE:-${SCRIPT_DIR}/github-deps.txt}"
REPOS_LIST="${REPOS_LIST:-${REPO_ROOT}/backend/repos.list}"
REPO_BASE="${REPO_BASE:-}"
CONTINUE_ON_ERROR="${CONTINUE_ON_ERROR:-false}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

FAILURES=()

export GIT_TERMINAL_PROMPT="${GIT_TERMINAL_PROMPT:-0}"

mkdir -p "${TARGET_ROOT}"

is_github_url() {
  local source_url="$1"
  [[ "${source_url}" == https://github.com/* ]]
}

record_failure() {
  local item="$1"
  FAILURES+=("${item}")
  if [[ "${CONTINUE_ON_ERROR}" != "true" ]]; then
    exit 1
  fi
}

mirror_git() {
  local source_url="$1"
  local target_dir="$2"

  mkdir -p "$(dirname "${target_dir}")"
  if [[ -d "${target_dir}" ]]; then
    git -C "${target_dir}" remote set-url origin "${source_url}"
    if [[ -n "${GITHUB_TOKEN}" ]] && is_github_url "${source_url}"; then
      git -C "${target_dir}" \
        -c "http.https://github.com/.extraheader=Authorization: Bearer ${GITHUB_TOKEN}" \
        remote update --prune
    else
      git -C "${target_dir}" remote update --prune
    fi
  else
    if [[ -n "${GITHUB_TOKEN}" ]] && is_github_url "${source_url}"; then
      git -c "http.https://github.com/.extraheader=Authorization: Bearer ${GITHUB_TOKEN}" \
        clone --mirror "${source_url}" "${target_dir}"
    else
      git clone --mirror "${source_url}" "${target_dir}"
    fi
  fi
}

download_artifact() {
  local source_url="$1"
  local target_file="$2"

  mkdir -p "$(dirname "${target_file}")"
  if [[ -n "${GITHUB_TOKEN}" ]] && is_github_url "${source_url}"; then
    curl -fL --retry 3 --connect-timeout 15 --max-time 600 \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      -o "${target_file}.tmp" "${source_url}"
  else
    curl -fL --retry 3 --connect-timeout 15 --max-time 600 \
      -o "${target_file}.tmp" "${source_url}"
  fi
  mv "${target_file}.tmp" "${target_file}"
}

while read -r dep_type dep_url dep_target; do
  [[ -z "${dep_type:-}" ]] && continue
  [[ "${dep_type}" =~ ^# ]] && continue

  case "${dep_type}" in
    git)
      mirror_git "${dep_url}" "${TARGET_ROOT}/${dep_target}" || record_failure "git ${dep_url}"
      ;;
    artifact)
      download_artifact "${dep_url}" "${TARGET_ROOT}/${dep_target}" || record_failure "artifact ${dep_url}"
      ;;
    *)
      echo "unknown dependency type: ${dep_type}" >&2
      exit 1
      ;;
  esac
done < "${DEPS_FILE}"

if [[ -f "${REPOS_LIST}" ]]; then
  while read -r repo_path; do
    [[ -z "${repo_path}" ]] && continue
    [[ "${repo_path}" =~ ^# ]] && continue

    repo_name="$(basename "${repo_path}")"
    if [[ "${repo_path}" == *"://"* ]]; then
      repo_url="${repo_path}"
    else
      if [[ -z "${REPO_BASE}" ]]; then
        record_failure "relative repo ${repo_path}: REPO_BASE is required"
        continue
      fi
      repo_url="${REPO_BASE%/}/${repo_path}"
    fi
    mirror_git "${repo_url}" "${TARGET_ROOT}/git/cloudiac/${repo_name}" || record_failure "git ${repo_url}"
  done < "${REPOS_LIST}"
fi

cat <<EOF
Offline dependencies are mirrored under:
  ${TARGET_ROOT}

Next steps:
  1. Push ${TARGET_ROOT}/git/**/*.git to local GitLab with 'git push --mirror'.
  2. Push container images and binary artifacts to Harbor, GitLab Package Registry, Nexus, or an internal static server.
  3. Generate Terraform providers into ${TARGET_ROOT}/terraform/providers.
  4. If backend/repos.list contains relative paths, rerun with REPO_BASE pointing to local GitLab.
EOF

if [[ "${#FAILURES[@]}" -gt 0 ]]; then
  echo
  echo "Some dependencies were not mirrored:" >&2
  printf '  - %s\n' "${FAILURES[@]}" >&2
  echo "For private GitHub repositories, rerun with GITHUB_TOKEN." >&2
  exit 2
fi
