#!/usr/bin/env bash

set -e

pushd code
echo "### Remove vendor folder ..."
rm -rf vendor
echo "### Go mod vendor ..."
go mod vendor

echo "### Compiling POC app ..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags "all=-N -l" -o out/kaniko-app main.go
popd

echo "### Creating the POC image ..."
docker build -t kaniko-app -f Dockerfile_build .