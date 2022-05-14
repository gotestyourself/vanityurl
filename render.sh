#!/usr/bin/env bash
set -eu -o pipefail

rm -rf pkgs repos docs
mkdir -p repos docs
pushd repos

golist() {
    go list -f '{{.Module.Path}} {{.ImportPath}}' ./... >> ../../pkgs
}

git clone --quiet git@github.com:gotestyourself/gotestsum
(
    cd gotestsum
    golist
)

git clone --quiet git@github.com:gotestyourself/gotest.tools
(
    cd gotest.tools
    golist
    git checkout --quiet v2.2.0
    go mod init gotest.tools
    golist
)

popd
go run ./render.go ./docs/ < ./pkgs
