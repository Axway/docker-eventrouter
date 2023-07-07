#!/bin/sh
#

set -euo pipefail

export COMPOSE_PROJECT_NAME=qlt_router_integration
NAME=qlt_router_integration_sut 
docker rm -f $NAME || true 
docker compose -f docker-compose.test.yml run --build --name $NAME sut 
docker cp $NAME:/app/src/coverage.xml .
docker cp $NAME:/app/src/coverage.svg .
docker cp $NAME:/app/src/report.xml .
docker rm -f $NAME || true 
