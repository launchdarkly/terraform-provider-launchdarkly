#!/usr/bin/env bash

# Updates the api-client-go dependency, including a major version bump.
#
# Usage: scripts/update_api_client_go.sh VERSION      # e.g. 24.0.0
#
# A major release of the LaunchDarkly API client moves the Go module path
# (api-client-go/vN), so this is not a one-line go.mod edit — the import path
# appears in ~220 files here and all of them have to move together.
#
# Called by the release automation in launchdarkly/ld-openapi-private, which opens
# the bump PR here after publishing a new client version. Safe to run by hand.

set -e

version=$1

if [ -z "$version" ]; then
  echo "usage: scripts/update_api_client_go.sh VERSION (e.g. 24.0.0)" >&2
  exit 1
fi

if ! echo "$version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "VERSION must be bare semver MAJOR.MINOR.PATCH with no leading 'v' (got '$version')" >&2
  exit 1
fi

root="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." >/dev/null 2>&1 && pwd )"
cd "$root"

module_base=github.com/launchdarkly/api-client-go
new_major=${version%%.*}

current_path=$(grep -oE "${module_base}/v[0-9]+" go.mod | head -1 || true)
if [ -z "$current_path" ]; then
  echo "could not find ${module_base}/vN in go.mod" >&2
  exit 1
fi
old_major=${current_path##*/v}

echo "Updating api-client-go: v${old_major} -> v${new_major} (pinning ${version})"

if [ "$old_major" != "$new_major" ]; then
  old_module="${module_base}/v${old_major}"
  new_module="${module_base}/v${new_major}"

  # Read loop rather than `mapfile`, and `perl -pi` rather than `sed -i`: both
  # mapfile and bare sed -i are GNU/bash-4-only, and macOS ships bash 3.2 with a
  # BSD sed that consumes the next argument as a backup suffix.
  echo "Rewriting import paths in Go sources"
  go_files=()
  while IFS= read -r file; do
    go_files+=("$file")
  done < <(git grep -l --fixed-strings "$old_module" -- '*.go' || true)
  if [ ${#go_files[@]} -eq 0 ]; then
    echo "expected Go files importing ${old_module}, found none - has the dependency moved?" >&2
    exit 1
  fi
  echo "  ${#go_files[@]} files"
  perl -pi -e "s|\Q${old_module}\E|${new_module}|g" "${go_files[@]}"

  # Drop the stale requirement first, so go.mod never holds a vN module path
  # pinned to a v(N-1) version.
  go mod edit -droprequire "$old_module"
fi

go get "${module_base}/v${new_major}@v${version}"
go mod tidy
