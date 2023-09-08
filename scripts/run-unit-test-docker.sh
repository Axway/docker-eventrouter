#!/bin/sh
#

set -euo pipefail

export COMPOSE_PROJECT_NAME=qlt_router_unit
NAME=qlt_router_unit_sut 

run() {
    docker rm -f $NAME || true 
    docker compose -f docker-compose.test.yml run --build --name $NAME sut-unit 
    docker cp $NAME:/app/src/coverage.xml .
    docker cp $NAME:/app/src/coverage.svg .
    docker cp $NAME:/app/src/report.xml .
    docker rm -f $NAME || true 
}

case ${1:-} in
    reset)
        docker compose -f docker-compose.test.yml pull
        docker compose -f docker-compose.test.yml down --remove-orphans -v
        run
    ;;
    *)
        run
    ;;
esac
