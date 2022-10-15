#!/usr/bin/env bash
set -eu -o pipefail

rm -rf pkgs repos docs
git checkout docs/CNAME
mkdir -p repos docs
target="$PWD/pkgs"

pushd repos

golist() {
    go list -f '{{.Module.Path}} {{.ImportPath}}' ./... >> "$target"
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
    (cd x/generics; golist)
    git checkout --quiet v2.2.0
    go mod init gotest.tools
    golist
)

popd
go run ./render.go ./docs/ < ./pkgs
