#!/bin/bash

set -ue
# Prep for getting goreleaser
echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | tee /etc/apt/sources.list.d/goreleaser.list
apt-get update
# Get goreleaser and gnupg
apt-get install -y --no-install-recommends \
    goreleaser \
    gnupg \
; \

# Get GPG Key
echo -e "$(cat "${LD_RELEASE_SECRETS_DIR}/gpg_private_key")" | gpg --import --batch --no-tty
echo "hello world" > temp.txt
gpg --detach-sig --yes -v --output=/dev/null --pinentry-mode loopback --passphrase "$(cat "${LD_RELEASE_SECRETS_DIR}/gpg_passphrase")" temp.txt
rm temp.txt
# Set it to env
export GPG_FINGERPRINT=$(gpg --with-colons --list-keys | awk -F: '/^pub/ { print $5 }')
