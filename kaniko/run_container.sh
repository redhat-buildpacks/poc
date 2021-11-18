#!/bin/bash

container_image=kaniko-app # gcr.io/kaniko-project/executor:latest

docker run -it \
    -v $(pwd)/workspace:/workspace \
    ${container_image}