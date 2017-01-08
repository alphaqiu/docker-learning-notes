#!/usr/bin/env bash

# step 1 create dir
mkdir -p /tmp/${SERVICE_DIR}

# step 2 link to the docker output
# mock /var/lib/docker/volumes/xxx -> /tmp/dzhyun/xxx
ln -s ${DOCKER_VOLUME_DIR} /tmp/${SERVICE_DIR}
