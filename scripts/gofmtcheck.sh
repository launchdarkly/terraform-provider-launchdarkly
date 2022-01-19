#!/usr/bin/env bash

echo "==> Checking that code complies with gofmt and gmts requirements..."
# Check gofmt
gofmt_files=$(gofmt -l `find . -name '*.go' | grep -v vendor`)
if [[ -n ${gofmt_files} ]]; then
    echo 'gofmt needs running on the following files:'
    echo "${gofmt_files}"
    echo "You can use the command: \`make fmt\` to reformat code."
    exit 1
fi

# Check gofmts
if ! which gofmts > /dev/null; then
    echo "==> Installing gofmts..."
    go install github.com/ashanbrown/gofmts/cmd/gofmts@v0.1.4
fi
gofmts_files=$(gofmts -l `find . -name '*.go' | grep -v vendor`)
if [[ -n ${gofmt_files} ]]; then
    echo 'gofmts needs running on the following files:'
    echo "${gofmts_files}"
    echo "You can use the command: \`make fmt\` to reformat code."
    exit 1
fi


exit 0
