#!/usr/bin/env bash

set -e

pushd code
echo "### Remove vendor folder ..."
rm -rf vendor
echo "### Go mod vendor ..."
go mod vendor

#echo "### Compiling buildah app ..."
# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'containers_image_openpgp' -gcflags "all=-N -l" -o out/buildah-app main.go
popd

echo "### Build the image ..."
docker build -t buildah-app -f Dockerfile_build .