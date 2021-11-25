# Poc development

This project has been designed in order to validate if we can build an image using
`buildah go lib` and `Dockerfiles` and next to access the content of the new layer(s)
created as part of the container FS.

## Kaniko 

See Kaniko [readme.md](./kaniko/README.md)

## Buildah

See Kaniko [readme.md](./buildah/README.md)

## How to get from an image, its index.json, manifest and digest files and content of a layer

The following instructions will help us to figure out how we:
- Apply a dockerfile to an image
- Get locally the image built
- Extract from the image its index.json, manifest file 
- Access the content of a layer
- Extract or check the content of the files added during the execution of the `dockerfiles`

**REMARK**: I commented the line using the tool umoci but if needed it could also be investigated

```bash
sudo rm -rf _temp && mkdir -p _temp
REPO="buildpack-poc"
sudo podman rmi localhost/$REPO
pushd _temp  

cat <<'EOF' > Dockerfile
FROM registry.access.redhat.com/ubi8:8.4-211

RUN yum install -y --setopt=tsflags=nodocs nodejs && \
	rpm -V nodejs && \
	yum -y clean all
EOF

sudo buildah bud -q -f Dockerfile -t $REPO . > /dev/null 2>&1

GRAPH_DRIVER="overlay"
TAG=$(sudo buildah --storage-driver $GRAPH_DRIVER images | awk -v r="$REPO" '$0 ~ r {print $2;}')
IMAGE_ID=$(sudo buildah --storage-driver $GRAPH_DRIVER images | awk -v r="$REPO" '$0 ~ r {print $3;}')
sudo skopeo copy -q containers-storage:$IMAGE_ID oci:$(pwd)/$IMAGE_ID:$TAG > /dev/null 2>&1

# TOOL able to unpack the FS from an image (https://github.com/opencontainers/umoci)
# sudo ../umoci unpack --image $IMAGE_ID:$TAG bundle

cat $IMAGE_ID/index.json
MANIFEST_SHA=$(cat $IMAGE_ID/index.json | jq .manifests[0].digest | cut -d: -f2 | sed 's/.$//')
echo "MANIFEST SHA: $MANIFEST_SHA"
cat $IMAGE_ID/blobs/sha256/$MANIFEST_SHA | python -m json.tool

DIGEST_SHA=$(cat $IMAGE_ID/blobs/sha256/$MANIFEST_SHA | jq .config.digest | cut -d: -f2 | sed 's/.$//')
echo "DIGEST SHA: $DIGEST_SHA"
cat $IMAGE_ID/blobs/sha256/$DIGEST_SHA | python -m json.tool

LAST_LAYER_ID=$(cat $IMAGE_ID/blobs/sha256/$MANIFEST_SHA | jq .layers[-1].digest | cut -d: -f2 | sed 's/.$//')
echo "LAST LAYER SHA: $LAST_LAYER_ID"
echo "## Display the content of the layer containing the package added ..."
tar -tvf $IMAGE_ID/blobs/sha256/$LAST_LAYER_ID

popd
```

## Python utility tool to list, unpack layer

See: https://blog.oddbit.com/post/2015-02-13-unpacking-docker-images/

To list the files of an image
```
docker save busybox | ./undocker.py -v --layers
```
End to end test using as FROM image `alpine` and the following Dockerfile
```bash
cat <<EOF > Dockerfile-alpine
FROM alpine

RUN apk add wget
EOF

docker build -f Dockerfile my-alpine .
IMAGE_ID=$(docker images --format="{{.Repository}} {{.ID}}" | awk '/none/ { print $2; }')
docker tag $IMAGE_ID my-alpine
LAST_LAYER_ID=$(docker save localhost/my-alpine | ./undocker.py --layers | head -n 1)
docker save my-alpine | ./undocker.py -i -o my-alpine-wget -l $LAST_LAYER_ID

Example: 
e2eb06d8af8218cfec8210147357a68b7e13f7c485b991c288c2d01dc228bb68 # Original image
b67c5a78b01d62b9eb65c0a8d46480c7b1882828b658ae8ddd5fc0601b2db3f9 # what I added with the RUN cmd

docker save my-alpine |
  ./undocker.py -vi -o my-alpine-wget -l f35d9c7ad180a77b0969ca4e87e6f9655098d577cc29f64cae5c300d9c33d753
  
Check the tree of the folder created locally
tree my-alpine-wget   
```

## Deprecated

### Mount root FS

See the following links where it is discussed `How to mount a root FS`:
- https://itnext.io/mount-a-kubernetes-workers-root-filesystem-as-a-container-volume-for-fun-and-fortune-53ae492698db
- https://github.com/kubernetes/kubernetes/issues/101749

The problem that we will have, if we want to mount the root FS, within the pod is that currently
we cannot mount it under `/` but using a different path as otherwise that will clash
```yaml
spec:
  volumes:
    - name: host
      hostPath:
        path: /
volumeMounts:
  - name: cache-dir
    mountPath: /workspace
  - name: host
    mountPath: /host
```

The consequence is that we need to find a way to copy the content of the subpath `/host` to the `/`
using a different `initContainer` which is only used to copy the files coming from the layers !

### MacOS

It is not possible for the moment to develop on a Mac as it is not a real Linux platform !

- Prerequisite

The following package is needed otherwise the compilation of the application will fail

```bash
# github.com/mtrmac/gpgme
../../golang/pkg/mod/github.com/mtrmac/gpgme@v0.1.2/data.go:4:11: fatal error: 'gpgme.h' file not found
 #include <gpgme.h>
          ^~~~~~~~~
```

It can be installed using brew
```bash
brew install gpgme
```