#!/usr/bin/env bash

set -e

pushd code
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'containers_image_openpgp' -gcflags "all=-N -l" -o out/buildah-app main.go
popd

docker build -t build-app -f Dockerfile_build .