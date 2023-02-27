#!/bin/bash

set -ue

# Run goreleaser
# We can't run in the build step, as project-releaser only tags the commit after the build step finishes and goreleaser pulls the tag off the most recent commit
GPG_FINGERPRINT=$(gpg --with-colons --list-keys | awk -F: '/^pub/ { print $5 }') GITHUB_TOKEN="$(cat "${LD_RELEASE_SECRETS_DIR}/github_token")" goreleaser release --clean --release-notes ../entry.tmp

# Remove extra files that we don't want in our release
rm /tmp/project-releaser/artifacts/artifacts.json
rm /tmp/project-releaser/artifacts/metadata.json
rm /tmp/project-releaser/artifacts/config.yaml
# Remove the binaries themselves as goreleaser puts them in subfolders
# We only want to keep the .zip files to release
rm -rf /tmp/project-releaser/artifacts/*/
