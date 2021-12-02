Table of Contents
=================

  * [Buildah App](#buildah-app)
  * [How to build and run](#how-to-build-and-run)
    * [Vagrant](#vagrant)
    * [Container](#container)
    * [How to verify what it happened](#how-to-verify-what-it-happened)
    * [Remote debugging](#remote-debugging)
    * [Kubernetes](#kubernetes)

## Buildah App

TODO

## How to build and run

### Vagrant

As it is needed to use a Linux environment to test the go executable, we will use Vagrant as tool
to launch a Linux VM locally which contains the needed tools (github, podman, buildah, ...), go framework, ...

- Open locally a terminal and move to the `vagrant` folder
- Launch the vm - `vagrant up` and ssh - `vagrant ssh`.
- Within the VM, you can build the project and launch it within the vm

```bash
cd poc/buildah/code
go build -tags exclude_graphdriver_devicemapper -o out/bud *.go
```

Copy the `dockerfile` to be parsed to the `/home/vagrant/wks` folder
```bash
cp $HOME/poc/buildah/wks/Dockerfile $HOME/wks
```

To parse the [Dockerfile](buildah/Dockerfile) pushed under the `WORKSPACE_DIR`, simply execute the
`bud` go application. It will process it and will generate an image
```bash
[vagrant@centos7 buildah]$ sudo WORKSPACE_DIR="/home/vagrant/wks" $HOME/poc/buildah/code/out/bud
WARN[0000] Failpwd
PACE DIR: /home/vagrant/wks             
INFO[0000] GRAPH_DRIVER: vfs                            
INFO[0000] STORAGE ROOT PATH: /var/lib/containers/storage 
INFO[0000] STORAGE RUN ROOT PATH: /var/run/containers/storage 
INFO[0000] Buildah tempdir: /home/vagrant/wks/buildah-layers 
INFO[0000] Dockerfile: /home/vagrant/wks/Dockerfile     
INFO[0027] Image id: bf4b432845dc71930dfcb9905d9a3de25c76f14763c0b69b97d87504ea228979 
INFO[0027] Image built successfully :-) 
```
The image created is available under the folder `/var/lib/containers/storage` using the appropriate grafh driver
```bash
sudo ls -la /var/lib/containers/storage/vfs-images/
total 44
drwx------.  8 root root  4096 Nov 17 17:01 .
drwx------. 14 root root   251 Oct 29 14:33 ..
drwx------.  2 root root  4096 Nov 17 17:01 9d69b1d0c28801834a8752b85de0a8d1b480ccc08e0696c241009e22db6729b9
drwx------.  2 root root  4096 Nov 17 16:50 bf4b432845dc71930dfcb9905d9a3de25c76f14763c0b69b97d87504ea228979
-rw-------.  1 root root 10079 Nov 17 17:01 images.json
-rw-r--r--.  1 root root    64 Nov 17 17:01 images.lock

```
**NOTE**: From your local machine, sync the files with the VM using the command `vagrant rsync`

### Container

Alternatively, build a container image using the following instructions

- First, download the vendor libs to avoid that docker download them for every build
```bash
cd code
go mod vendor
cd ..
```
- Build the image containing `golang` and the needed `containers` libs
```bash
docker build -t go-containers -f Dockerfile_go_containers .
```  
- And now build the `buildah-app` container image
```bash
./hack/build.sh
```
- Launch the `buildah-app` container
```bash
docker run \
  --privileged \
  -e GRAPH_DRIVER=vfs \
  -e LOGGING_LEVEL=debug \
  -e LOGGING_FORMAT=color \
  -e WORKSPACE_DIR=/wks \
  -v $(pwd)/wks:/wks \
  -v $(pwd)/cache:/cache \
  -it buildah-app
```

### How to verify what it happened

Review the log and check if an image has been built and layers copied under the folder `/cache`
using as key the id of the image. Examples of such reports are available within the [test_report](./test_report) folder (see: `test-d8ec29c.txt`):

```bash
INFO[0090] Image id: 85cac84a9ac17b782117490e4789525badc8de9a3bf71f7abd721b623a8b3521
INFO[0090] Image digest: localhost/buildpack-buildah:1638375495183671507-1@sha256:d2977cb5192d5045e2036855e20d2a2bc6da959a278366d91b5be0909ab03308
INFO[0090] Image manifest: {
...
INFO[0090] OCI Config: {
    "created": "2021-12-01T16:19:40.317081245Z",
    "architecture": "amd64",
    "os": "linux",
...    
INFO[0090] Layer sha: sha256:0d3f22d60daf4a2421b2239fb0e1c6ec02d3787274db8b098fb648941ea2d5dc
INFO[0090] Layer sha: sha256:0488bd866f642b2b1b5490f5c50d628815e4e8fa1f7cae57d52c67c1e9d3e2cc
INFO[0090] Layer sha: sha256:484159bb1f91a3a34382d43c2de5f8f95a8848947130179a0b2d44addfe3a03f
...
INFO[0090] Top layer: f747669093973254d1b3d1103397cc3b71e2c34da696b2d92b6081f6e431dd69
INFO[0090] Image repositry id: 85cac84a9ac
INFO[0090] Image built successfully :-)

IMAGE_ID="85cac84a9ac"
ls -la ./cache/$IMAGE_ID/blobs/sha256
total 264720
drwxr-xr-x  7 cmoullia  staff       224 Dec  1 18:03 .
drwxr-xr-x  3 cmoullia  staff        96 Dec  1 18:03 ..
-rw-r--r--  1 cmoullia  staff      1876 Dec  1 18:03 22f677655049d4c2e6cd9e49ca9ed20f34ac175ef0c82f5c5eabc79031c1c29a
-rw-r--r--  1 cmoullia  staff       664 Dec  1 18:03 4d614c43e697d0e2ed0383f06b3badd08e6edccf1643c2820a424e7c52c918e2
-rw-r--r--  1 cmoullia  staff  85633977 Dec  1 18:03 ac56bdc7f9934acede05653e9e01e73e961c31818b522c0732ad35350bb3a82b
-rw-r--r--  1 cmoullia  staff      2606 Dec  1 18:03 b1c9b294ef0424dccd2d082fb5e9002ae506b7d3f4132215d4f3f4296dbcfd45
-rw-r--r--  1 cmoullia  staff  33416720 Dec  1 18:03 f9a38a40c9dfafa1795d9655acefbbfcba44546a38382ab17abc892357fb0e95
```

### Remote debugging

To use the dlv remote debugger, simply pass as `ENV` var `DEBUG=true` and the port `2345` to access it using your favorite IDE (Visual studio, IntelliJ, ...)
```bash
docker run \
  -e DEBUG=true \
  -p 2345:2345 \
  -e GRAPH_DRIVER=vfs \
  -e LOGGING_LEVEL=debug \
  -e LOGGING_FORMAT=color \
  -e WORKSPACE_DIR=/wks \
  -v $(pwd)/vol:/var/lib/containers \
  -v $(pwd)/wks:/wks \
  -it buildah-app
```

### Kubernetes

To test the POC on a kubernetes cluster, build a container image from your local machine (containing the poc bud executable).

```bash
cd buildah
REPO=quay.io/snowdrop/buildah-poc
docker build -t $REPO -f Dockerfile_bud .
docker push $REPO
```

Next, deploy the poc on kubernetes to verify if buildah can build the image
```bash
kubectl apply -f k8s/manifest.yml
```
To clean up the project on kubernetes
```bash
kubectl delete -f k8s/manifest.yml
```