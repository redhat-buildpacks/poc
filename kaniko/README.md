Table of Contents
=================

* [kaniko go app](#kaniko-go-app)
* [How to build and run the application](#how-to-build-and-run-the-application)
* [Remote debugging](#remote-debugging)
* [CNB Build args](#cnb-build-args)
* [Ignore Paths](#ignore-paths)
* [Extract layer files](#extract-layer-files)
* [Verify if files exist](#verify-if-files-exist)
* [Cache content](#cache-content)
* [Using Kubernetes](#using-kubernetes)

## kaniko go app

The [kaniko app](./code/main.go) is a simple application able to build an image using kaniko and a [Dockerfile](./workspace/alpine).

Example of a dockerfile to be parsed by Kaniko
```dockerfile
FROM alpine

RUN apk add wget curl
```

During the execution of this kaniko app:
- We will call the [kaniko build function](https://github.com/GoogleContainerTools/kaniko/blob/master/pkg/executor/build.go#L278),
- Kaniko will parse the Dockerfile, execute each docker commands (RUN, COPY, ...) that it [supports](https://github.com/GoogleContainerTools/kaniko/tree/master/pkg/commands),
- A snapshot of each layer (= command executed) is then created,
- Finally, the layers will be pushed into an image,
- Our app will copy the layers created from the `/kaniko` dir to the `/cache` dir
- For each layer (except the base image), the content will be extracted under the root FS `/`

When the `kaniko-app` is launched, then the following [Dockerfile](./workspace/alpine) is parsed. This dockerfile will install some missing packages: `wget, curl`

**NOTE**: a layer is saved as a `sha256:xxxxx.tgz` file under the `/kaniko` dir. The `xxxxxx` corresponds the [layer.digest](https://pkg.go.dev/github.com/google/go-containerregistry@v0.7.0/pkg/name#Digest)
which is the hash of the compressed layer.

## How to build and run the application

To play with the application, first download the dependencies using `go mod vendor` to avoid that for every `docker build`, docker reloads all the dependencies.
```bash
cd code
go mod vendor
cd ..
```

**NOTE**: The commands reported hereafter should be executed in your terminal under: `$(pwd)/kaniko`

Build next the container image of the `kaniko-app`.
```bash
./hack/build.sh
```
Launch the `kaniko-app` container
```bash
docker run \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```
Different `ENV` variables can be defined and passed as parameters to the containerized engine:
`LOGGING_LEVEL`    Log level: trace, debug, **info**, warn, error, fatal, panic
`LOGGING_FORMAT`   Logging format: **text**, color, json
`DOCKER_FILE_NAME` Dockerfile to be parsed: **Dockerfile** is the default name
`DEBUG`            To launch the `dlv` remote debugger. See [remote debugger](#remote-debugging) 
`EXTRACT_LAYERS`   To extract from the layers (= tgz files) the files. See [extract layers](#extract-layer-files)
`CNB_*`            Pass Arg to the Dockerfile. See [CNB Args](#cnb-build-args)
`IGNORE_PATHS`     Files to be ignored by Kaniko. See [Ignore Paths](#ignore-paths). TODO: Should be also used to ignore paths during `untar` process or file search
`FILES_TO_SEARCH`  Files to be searched post layers content extraction. See [files to search](#verify-if-files-exist)                

Example using `DOCKER_FILE_NAME` env var
```bash
docker run \
       -e DOCKER_FILE_NAME="alpine" \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```

To verify that the `kaniko` application is working fine, execute the following command
```dockerfile
dockerfile="ubi8-nodejs"        
filesToSearch="node,hello.txt"
docker run \
  -e EXTRACT_LAYERS=true \
  -e IGNORE_PATHS="/proc" \
  -e FILES_TO_SEARCH=${filesToSearch} \
  -e LOGGING_LEVEL=info \
  -e LOGGING_FORMAT=color \
  -e DOCKER_FILE_NAME=${dockerfile} \
  -v $(pwd)/workspace:/workspace \
  -v $(pwd)/cache:/cache \
  -it kaniko-app:latest
-->
...
DEBU[0349] File found: /usr/bin/node                    
DEBU[0349] File found: /workspace/hello.txt 
```
and check the information logged.

**NOTE**: You can also use the search_files bash [script](./scripts/search_files.sh) which will scan the content of the `tgz` files
and search about the following keywords passed to grep - "wget\|curl\|hello.txt". TODO: Pass the keywords as bash script arg

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

## Remote debugging

To use the dlv remote debugger, simply pass as `ENV` var `DEBUG=true` and the port `4000` to access it using your favorite IDE (Visual studio, IntelliJ, ...)
```bash
docker run \
  -e DEBUG=true \
  -p 2345:2345 \
  -v $(pwd)/workspace:/workspace \
  -v $(pwd)/cache:/cache \
  -it kaniko-app
```

## CNB Build args

When the Dockerfile contains some `ARG arg` commands

```dockerfile
ARG CNB_BaseImage
FROM ${CNB_BaseImage}
...
```
then, we must pass them as `ENV vars` to the container. Our application will then convert the ENV var into a Kaniko `BuildArgs` array of `[]string`

```bash
docker run \
       -e LOGGING_LEVEL=debug \
       -e LOGGING_FORMAT=color \
       -e CNB_BaseImage="ubuntu:bionic" \
       -e DOCKER_FILE_NAME="base-image-arg" \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```

## Ignore Paths

To ignore some paths during the process to create the new image, then use the following `IGNORE_PATHS` env var which is used by [kaniko](https://github.com/GoogleContainerTools/kaniko#--ignore-path).
Multiple paths can be defined using as separator `,`.

```bash
docker run \
       -e EXTRACT_LAYERS=false \
       -e IGNORE_PATHS="/var/spool/mail" \
       -e FILES_TO_SEARCH="hello.txt,curl" \
       -e LOGGING_LEVEL=debug \
       -e LOGGING_FORMAT=color \
       -e DOCKER_FILE_NAME="alpine" \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```

**NOTE**: If the ENV var is not set, then an empty array of string is passed to Kaniko Opts


## Extract layer files

By default, the layer tgz files are not extracted to the home dir of the container's filesystem. Nevertheless, the files part
of the compressed tgz files will be logged.

To extract the layers files, enable the following ENV var `EXTRACT_LAYERS=true`

```bash
docker run \
       -e EXTRACT_LAYERS=true \
       -e LOGGING_FORMAT=color \
       -e DOCKER_FILE_NAME="alpine" \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
```

## Verify if files exist

To check/control if files added from the layers exist under the root filesystem, please use the following `ENV` var `FILES_TO_SEARCH`

```bash
docker run \
       -e EXTRACT_LAYERS=true \
       -e FILES_TO_SEARCH="hello.txt,curl" \
       -e LOGGING_LEVEL=debug \
       -e LOGGING_FORMAT=color \
       -e DOCKER_FILE_NAME="alpine" \
       -v $(pwd)/workspace:/workspace \
       -v $(pwd)/cache:/cache \
       -it kaniko-app
...
DEBU[0009] File found: /usr/bin/curl                    
DEBU[0009] File found: /workspace/hello.txt          
```


## Cache content

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

## Using Kubernetes

To run the `kaniko-app` as a kubernetes pod, some additional steps are required and described hereafter.

Create a k8s cluster having access to your local workspace and cache folders. This step can be achieved easily using kind
and the following [bash script](scripts/kind-reg.sh) where the following config can be defined to access your local folders
```yaml
  extraMounts:
    - hostPath: $(pwd)/workspace  # PLEASE CHANGE ME
      containerPath: /workspace
    - hostPath: $(pwd)/cache      # PLEASE CHANGE ME
      containerPath: /cache
```
Next, create the cluster using the command `./k8s/kind-reg.sh`

When the cluster is up and running like the registry, we can push the image:
```bash
REGISTRY="localhost:5000"
docker tag kaniko-app $REGISTRY/kaniko-app
docker push $REGISTRY/kaniko-app
```

and then deploy the kaniko pod
```bash
kubectl apply -f k8s/manifest.yml 
```
**NOTE**: Check the content of the pod initContainers logs using [k9s](https://k9scli.io/) or another tool :-)

To delete the pod, do
```bash
kubectl delete -f k8s/manifest.yml
```