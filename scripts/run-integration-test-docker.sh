#!/bin/sh
#

# set -euo pipefail

export COMPOSE_PROJECT_NAME=event_router_integration
NAME=event_router_integration_sut 

run() {
    docker rm -f $NAME || true 
    docker compose -f docker-compose.test.yml run --build --name $NAME sut
    rc=$?
    docker cp $NAME:/app/src/coverage.xml .
    docker cp $NAME:/app/src/coverage.svg .
    docker cp $NAME:/app/src/report.xml .
    docker rm -f $NAME || true 
    return $rc
}

case ${1:-} in
    reset)
        docker compose -f docker-compose.test.yml pull
        docker compose -f docker-compose.test.yml down --remove-orphans -v
        run
        rc=$?
        docker compose -f docker-compose.test.yml down --remove-orphans -v
        exit $rc
    ;;
    *)
        run
        exit $?
    ;;
esac
