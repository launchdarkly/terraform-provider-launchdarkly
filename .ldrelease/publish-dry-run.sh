#!/bin/bash

set -ue

# Run goreleaser
# We can't run in the build step, as project-releaser only tags the commit after the build step finishes and goreleaser pulls the tag off the most recent commit
GPG_FINGERPRINT=$(gpg --with-colons --list-keys | awk -F: '/^pub/ { print $5 }') GITHUB_TOKEN="$(cat "${LD_RELEASE_SECRETS_DIR}/github_token")" LD_RELEASE_ARTIFACTS_DIR="${LD_RELEASE_ARTIFACTS_DIR}" goreleaser release --clean
