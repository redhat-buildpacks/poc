Table of Contents
=================

* [Buildah App](#buildah-app)
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
- And now compile and build the `buildah-app` container image
```bash
docker build -t buildah-app -f Dockerfile_build .
```
- Launch the `buildah-app` container
```bash
docker run \
  -e WORKSPACE_DIR=/wks \
  -v $(pwd)/wks:/wks \
  -it buildah-app
```

### Kubernetes

To test the POC on a kubernetes cluster, build a container image from your local machine (containing the poc bud executable).

```bash
cd buildah
REPO=quay.io/snowdrop/buildah-poc
docker build -t $REPO -f Dockerfile-bud .
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