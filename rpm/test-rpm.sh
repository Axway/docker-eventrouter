#!/bin/bash
#

NAME="qlt-router"

set -euo pipefail

I=$(realpath "$(dirname "$0")/..")
cd "$I"

docker build -f ./rpm/Dockerfile.redhat8 -t $NAME-tests .
docker run --user 10000:10000 --env USER=zouzou --rm $NAME-tests /config/tests/test.sh

docker build -f ./rpm/Dockerfile.redhat8.non-root -t $NAME-tests-nonroot .
docker run --user 10000:10000 --env USER=zouzou --rm $NAME-tests-nonroot /config/tests/test-non-root.sh

docker build -f ./rpm/Dockerfile.suze15 -t $NAME-tests .
docker run --user 10000:10000 --env USER=zouzou --rm $NAME-tests /config/tests/test.sh

docker build -f ./rpm/Dockerfile.suze15.non-root -t $NAME-tests-nonroot .
docker run --user 10000:10000 --env USER=zouzou --rm $NAME-tests-nonroot /config/tests/test-non-root.sh
