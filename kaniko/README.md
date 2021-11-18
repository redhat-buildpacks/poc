# Kaniko POC

## Dummy kaniko app

The [kaniko app](./code/main.go) is a simple application able to build an image using kaniko and a [Dockerfile](./workspace/Dockerfile).
during the build, kaniko will parse the Dockerfile, execute the different docker commands (RUN, COPY, ...) and the resulting content will be pushed into an image

So, if we install a new application such as wget using the following dockerfile
```dockerfile
FROM alpine

RUN echo "Hello World" > hello.txt
RUN apk add wget
```
then the layer created will include it 
```bash
tar -vxf sha256:aa2ad9d70c8b9b0b0c885ba0a81d71f5414dcac97bee8f5753ec03f92425c540.tgz
tar: Removing leading '/' from member names
x .
x etc/
x etc/apk/
x etc/apk/world
x etc/wgetrc
x lib/
x lib/apk/
x lib/apk/db/
x lib/apk/db/installed
x lib/apk/db/scripts.tar
x lib/apk/db/triggers
x usr/
x usr/bin/
x usr/bin/idn2
x usr/bin/wget
x usr/lib/
x usr/lib/libidn2.so.0
x usr/lib/libidn2.so.0.3.7
x usr/lib/libunistring.so.2
x usr/lib/libunistring.so.2.1.0
x var/
x var/cache/
x var/cache/apk/
x var/cache/apk/APKINDEX.406b1341.tar.gz
x var/cache/apk/APKINDEX.a251b1f2.tar.gz
x var/cache/misc/
```

To play with the application, first download the dependencies using `go mod vendor` to avoid that for every `docker build`, docker reloads all the dependencies.
The commands reported hereafter should be executed in your terminal under: `$(pwd)/kaniko`
```bash
cd code
go mod vendor
cd ..
```
Build the container image of the `kaniko-app` using docker.
```bash
docker build -t kaniko-app -f Dockerfile_build .
```
Launch the `kaniko-app` container
```bash
docker run \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```  
The content of the `dockerfile` which has been processed by the `Kaniko` build is available under the `./cache` folder
```bash
drwxr-xr-x  10 cmoullia  staff      320 Nov 18 14:00 .
drwxr-xr-x  10 cmoullia  staff      320 Nov 18 13:56 ..
-rw-r--r--@  1 cmoullia  staff     6148 Nov 18 13:54 .DS_Store
-rw-------   1 cmoullia  staff  4383232 Nov 18 13:58 425529682
-rw-------   1 cmoullia  staff     1024 Nov 18 13:58 544414207
-rw-------   1 cmoullia  staff     1024 Nov 18 13:50 577703017
-rw-r--r--   1 cmoullia  staff      933 Nov 18 13:58 config.json
-rw-r--r--   1 cmoullia  staff       12 Nov 18 13:58 hello.txt
-rw-r--r--@  1 cmoullia  staff  2822981 Nov 18 13:58 sha256:97518928ae5f3d52d4164b314a7e73654eb686ecd8aafa0b79acd980773a740d.tgz
-rw-r--r--   1 cmoullia  staff  3175266 Nov 18 13:58 sha256:aa2ad9d70c8b9b0b0c885ba0a81d71f5414dcac97bee8f5753ec03f92425c540.tgz
```

To use the dlv remote debugger       
```bash
docker run \
       -e DEBUG=true \
       -p 4000:4000 \
       -v $(pwd)/workspace:/workspace \
       -it kaniko-app
```
or deploy it as a k8s pod
```bash

```

## Test case using kaiko image on k8s

- To test kaniko using a k8s cluster, create it using `kind`
- Before to create the cluster, change the `hostPath` within the `./k8s/cfg.yml` cfg file to point to your local folders
  ```yaml
  extraMounts:
    - hostPath: /Users/cmoullia/code/redhat-buildpacks/poc/kaniko/wks # PLEASE CHANGE ME
      containerPath: /workspace
    - hostPath: /Users/cmoullia/code/redhat-buildpacks/poc/kaniko/snapshot # PLEASE CHANGE ME
      containerPath: /cache
  ```
- Next, create the cluster
  ```bash
  kind create cluster --config ./k8s/cfg.yml 
  ```
- When the cluster is up and running, we can deploy the kaniko pod able to process the `./wks/Dockerfile`  
  ```bash
  kc delete -f k8s/manifest.yml --force && kc apply -f k8s/manifest.yml 
  ```
- Check the content of the pod initContainers logs using [k9s](https://k9scli.io/) or another tool :-)