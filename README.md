# Poc development

This project has been designed in order to validate if we can build an image using
`buildah go lib` and `Dockerfiles`.

As it is needed to use a Linux environment to test the go executable, we will use Vagrant as tool
to launch a Linux VM locally which contains the needed tools (github, podman, buildah, ...) , go framework, ...

Open a terminal where you will be able to bump the VM using the command `vagrant up` and `vagrant ssh`.
Next, you can build the project and launch it there

```bash
cd poc
go build -o out/bud ./bud

sudo WORKSPACE_DIR="/home/vagrant/wks" ./out/bud
INFO[0000] WORKSPACE DIR: %!(EXTRA string=/home/vagrant) 
INFO[0000] Buildah tempdir : %!(EXTRA string=/home/vagrant/buildah-poc-3853289916) 
INFO[0000] Dockerfile name: %!(EXTRA string=/home/vagrant/poc/Dockerfile, string=/home/vagrant/poc/Dockerfile) 
INFO[0003] Image id: %!(EXTRA string=f8cce29a1b02c26b62dee5d7fad3dfd3e9474ca4c0205de626bc681b04b3f014)  
```
The image created is available under the temp buildah folder creates:
```bash
sudo ls -la /root/buildah-poc-3011355348/root/overlay-images/
total 16
drwx------. 4 root root  188 Oct 27 11:52 .
drwx------. 8 root root  155 Oct 27 11:52 ..
drwx------. 2 root root 4096 Oct 27 11:52 cf2a2d19642401ea6af3a51cfc5f5190fca39734409fb2f7f4f4c5173da9d70e
drwx------. 2 root root 4096 Oct 27 11:52 f8cce29a1b02c26b62dee5d7fad3dfd3e9474ca4c0205de626bc681b04b3f014
-rw-------. 1 root root 3558 Oct 27 11:52 images.json
-rw-r--r--. 1 root root   64 Oct 27 11:52 images.lock
```
**NOTE**: From your local machine, sync the files with the VM using the command `vagrant rsync`

## Kubernetes

To test the POC on kubernetes, build a container image from your local machine (containing the poc bud executable).

```bash
docker build -t quay.io/snowdrop/buildah-poc -f Dockerfile-bud .
docker push quay.io/snowdrop/buildah-poc
```

Next, deploy the poc on kubernetes to verify if buildah can buld the image
```bash
kubectl apply -f k8s/manifest.yml
```
To clean up the project on kubernetes
```bash
kubectl delete -f k8s/manifest.yml
```

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

## MacOS

It is not possible for the moment to develop on a Mac as it is not a real Linux platform !

### Prerequisite

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