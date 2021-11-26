#!/usr/bin/env bash

set -e

pushd code
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags "all=-N -l" -o out/kaniko-app main.go
popd

docker build -t kaniko-app -f Dockerfile_build .