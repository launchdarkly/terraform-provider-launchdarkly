#!/bin/bash

set -ue

# This adds a new line at the top of of CHANGELOG.md to the release version set in the releaser ui, and sets the date to todays date
# project-releaser reads CHANGELOG.md and puts all changelog entries from the UI under this heading
new_version_header="## [$LD_RELEASE_VERSION] - $(date +%F)"
sed -i'' -e "1i $new_version_header" ./CHANGELOG.md