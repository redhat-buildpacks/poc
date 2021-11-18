# Kaniko POC

## Dummy kaniko app

The [kaniko app](./code/main.go) is a simple application able to build an image using kaniko and a [Dockerfile](./workspace/Dockerfile).
During the build, kaniko will parse the Dockerfile, execute the different docker commands (RUN, COPY, ...) and the resulting content will be pushed into an image.
Kaniko will create different layers under the folder `/kaniko` as `sha256:xxxxx.tgz` files. The layer files will be then copied to the mounted volume `/cache`.

When the `kaniko-app` is launched, then the following [Dockerfile](./workspace/Dockerfile) is parsed. This dockerfile will install some missing packages: `wget, curl`
```dockerfile
FROM alpine

RUN echo "Hello World" > hello.txt
RUN apk add wget curl
```
then we can read the content of the layer tar file crated to verify if they have been added:
```bash
tar -tvf sha256:aa2ad9d70c8b9b0b0c885ba0a81d71f5414dcac97bee8f5753ec03f92425c540.tgz
...
drwxr-xr-x  0 0      0           0 Nov 18 14:22 lib/
drwxr-xr-x  0 0      0           0 Nov 12 10:18 lib/apk/
drwxr-xr-x  0 0      0           0 Nov 18 14:22 lib/apk/db/
-rw-r--r--  0 0      0       28213 Nov 18 14:22 lib/apk/db/installed
-rw-r--r--  0 0      0       13312 Nov 18 14:22 lib/apk/db/scripts.tar
-rw-r--r--  0 0      0         212 Nov 18 14:22 lib/apk/db/triggers
drwxr-xr-x  0 0      0           0 Nov 12 10:18 usr/
drwxr-xr-x  0 0      0           0 Nov 18 14:22 usr/bin/
-rwxr-xr-x  0 0      0       14232 Oct 25  2020 usr/bin/c_rehash
-rwxr-xr-x  0 0      0      239568 Sep 22 20:50 usr/bin/curl ## <-- curl app
-rwxr-xr-x  0 0      0       59864 May 17  2021 usr/bin/idn2
-rwxr-xr-x  0 0      0      465912 Jan 12  2021 usr/bin/wget ## <-- wget app
...
```
### How to build and run the application

To play with the application, first download the dependencies using `go mod vendor` to avoid that for every `docker build`, docker reloads all the dependencies.
```bash
cd code
go mod vendor
cd ..
```

**NOTE**: The commands reported hereafter should be executed in your terminal under: `$(pwd)/kaniko`

Build next the container image of the `kaniko-app`.
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
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```
or deploy it as a k8s pod
```bash

```

## Test case using kaniko image on k8s

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