#!/usr/bin/env bash
#
# capture.sh — generate state fixtures from synthetic local-testing configs.
#
# Per the locked fixture-safety policy in MIGRATION_PLAN_NON_BREAKING.md
# (Phase 0.5), fixtures committed to launchdarkly/testdata/state-fixtures/
# MUST be generated from synthetic configs against a dedicated test LD
# account, never from production state. The `access_token` resource stores
# token secrets in plaintext state, so real captures are forbidden in the
# repo regardless of sanitisation claims.
#
# This script:
#   1. Picks a synthetic .tf config from configs/<name>.tf
#   2. Runs `terraform init && terraform apply -auto-approve` against the
#      configured test LD account (LAUNCHDARKLY_ACCESS_TOKEN required)
#   3. Reads the resulting terraform.tfstate
#   4. Pipes it through sanitize.jq to deterministically replace
#      LD-side identifiers with placeholders from safe-placeholders.txt
#   5. Writes the result to launchdarkly/testdata/state-fixtures/<name>.tfstate
#   6. Re-runs scan.sh on the new fixture to confirm zero violations
#
# Usage:
#   ./scripts/capture-state-fixtures/capture.sh <fixture-name>
#
# Where <fixture-name> matches a file under
# scripts/capture-state-fixtures/configs/<fixture-name>.tf.
#
# Per-PR fixture work (the actual configs + their expected sanitised
# output) lands incrementally in Phases 2-4 alongside each resource
# migration, per the rolling-gate model in §Phase 0.9b.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
script_dir="${repo_root}/scripts/capture-state-fixtures"
configs_dir="${script_dir}/configs"
fixtures_dir="${repo_root}/launchdarkly/testdata/state-fixtures"

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <fixture-name>" >&2
  exit 2
fi

name="$1"
config="${configs_dir}/${name}.tf"

if [[ ! -f "${config}" ]]; then
  echo "missing config: ${config}" >&2
  echo "create scripts/capture-state-fixtures/configs/${name}.tf first" >&2
  exit 2
fi

if [[ -z "${LAUNCHDARKLY_ACCESS_TOKEN:-}" ]]; then
  echo "LAUNCHDARKLY_ACCESS_TOKEN env var is required" >&2
  exit 2
fi

workdir="$(mktemp -d)"
trap 'rm -rf "${workdir}"' EXIT

cp "${config}" "${workdir}/main.tf"

pushd "${workdir}" >/dev/null
terraform init -input=false
terraform apply -auto-approve -input=false
popd >/dev/null

raw_state="${workdir}/terraform.tfstate"
if [[ ! -f "${raw_state}" ]]; then
  echo "expected ${raw_state} after apply; aborting" >&2
  exit 2
fi

mkdir -p "${fixtures_dir}"
out="${fixtures_dir}/${name}.tfstate"

if [[ -f "${script_dir}/sanitize.jq" ]]; then
  jq -f "${script_dir}/sanitize.jq" "${raw_state}" > "${out}"
else
  # No sanitiser yet (early Phase 0); copy verbatim and rely on scan.sh
  # to catch any sensitive content.
  cp "${raw_state}" "${out}"
fi

# Tear down so we don't leak resources in the test LD account.
pushd "${workdir}" >/dev/null
terraform destroy -auto-approve -input=false
popd >/dev/null

"${script_dir}/scan.sh"
echo "captured fixture: ${out}"
