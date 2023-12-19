#!/bin/sh
#

set -euo pipefail

export COMPOSE_PROJECT_NAME=event_router_build
NAME=event-router
CONTAINER=$COMPOSE_PROJECT_NAME-$NAME

run() {
    docker rm -f $CONTAINER || true
    docker build -f docker/Dockerfile.glibc -t event_router_build_glibc .
    docker run --name $CONTAINER event_router_build_glibc $NAME version
    docker cp $CONTAINER:/usr/bin/$NAME .
    docker rm -f $CONTAINER || true 
}

case ${1:-} in
    *)
        run
    ;;
esac
