# Kaniko POC

## Local code

Build the container image of the `kaniko-app`
```bash
docker build -t kaniko-app -f Dockerfile_build .
```
Run it locally 
```bash
docker run --arg ext=app \
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