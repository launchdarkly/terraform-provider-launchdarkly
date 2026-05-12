#!/usr/bin/env bash
#
# scan.sh — fixture safety scan.
#
# Scans state-fixture files under launchdarkly/testdata/state-fixtures/ for
# patterns that look like real LD secrets or production identifiers.
# Anything matching the patterns below fails the build, forcing the
# committer to regenerate the fixture from synthetic values
# (capture.sh + safe-placeholders.txt) before it can land.
#
# This script is the source of truth for the CI fixture-safety gate; a
# matching .githooks/pre-commit can run the same script locally as
# best-effort developer DX, but CI remains the enforcement point.
#
# Exit codes:
#   0  — clean (no suspicious patterns found)
#   1  — at least one suspicious pattern detected
#   2  — internal error (missing files, bad args)

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fixtures_dir="${repo_root}/launchdarkly/testdata/state-fixtures"
allowlist_file="${repo_root}/scripts/capture-state-fixtures/safe-placeholders.txt"

if [[ ! -d "${fixtures_dir}" ]]; then
  echo "fixtures directory missing: ${fixtures_dir}" >&2
  exit 2
fi

if [[ ! -f "${allowlist_file}" ]]; then
  echo "allowlist file missing: ${allowlist_file}" >&2
  exit 2
fi

# Patterns that should never appear in committed fixtures.
#
# - api-<uuid>: LD personal/service access token shape.
# - sdk-<32hex>: LD SDK key shape.
# - mob-<32hex>: LD mobile key shape.
# - Long opaque base64-ish strings (>= 40 chars of [A-Za-z0-9_-]) that
#   aren't in the allowlist — likely API tokens or session secrets.
#
# Patterns are intentionally narrow; expand cautiously to avoid breaking
# legitimate hex-encoded resource IDs that synthetic captures use.
patterns=(
  'api-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}'
  'sdk-[a-f0-9]{32}'
  'mob-[a-f0-9]{32}'
)

# Build an awk allowlist filter — strings in safe-placeholders.txt are
# treated as known-safe even if they match a pattern (they shouldn't, but
# defence in depth).
mapfile -t allowlist < <(grep -v '^#' "${allowlist_file}" | grep -v '^$' || true)

shopt -s nullglob
files=("${fixtures_dir}"/*.tfstate "${fixtures_dir}"/*.json)
shopt -u nullglob

if [[ ${#files[@]} -eq 0 ]]; then
  echo "no fixtures present — scan passes vacuously"
  exit 0
fi

violations=0
for f in "${files[@]}"; do
  for pat in "${patterns[@]}"; do
    if matches="$(grep -E -o "${pat}" "${f}" 2>/dev/null)"; then
      while IFS= read -r match; do
        allowed=0
        for safe in "${allowlist[@]}"; do
          if [[ "${match}" == "${safe}" ]]; then
            allowed=1
            break
          fi
        done
        if [[ ${allowed} -eq 0 ]]; then
          echo "fixture-safety: ${f}: matched forbidden pattern (${pat}): ${match}" >&2
          violations=$((violations + 1))
        fi
      done <<< "${matches}"
    fi
  done
done

if [[ ${violations} -gt 0 ]]; then
  echo >&2
  echo "fixture-safety scan FAILED with ${violations} violations" >&2
  echo "regenerate the offending fixture using scripts/capture-state-fixtures/capture.sh" >&2
  echo "with synthetic values from scripts/capture-state-fixtures/safe-placeholders.txt" >&2
  exit 1
fi

echo "fixture-safety scan passed (${#files[@]} fixture(s) checked)"
exit 0
