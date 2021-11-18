# Kaniko POC

## Dummy kaniko app

First download the dependencies using `go mod vendor` to avoid that for every `docker build`, docker reloads all the dependencies.
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
ls -la
total 5560
drwxr-xr-x   8 cmoullia  staff      256 Nov 18 13:54 .
drwxr-xr-x  10 cmoullia  staff      320 Nov 18 13:55 ..
-rw-r--r--@  1 cmoullia  staff     6148 Nov 18 13:54 .DS_Store
-rw-------   1 cmoullia  staff     1024 Nov 18 13:50 577703017
-rw-r--r--   1 cmoullia  staff      942 Nov 18 13:53 config.json
-rw-r--r--   1 cmoullia  staff       12 Nov 18 13:50 hello.txt
-rw-r--r--@  1 cmoullia  staff  2822981 Nov 18 13:50 sha256:97518928ae5f3d52d4164b314a7e73654eb686ecd8aafa0b79acd980773a740d.tgz
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