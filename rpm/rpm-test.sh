#!/bin/bash
#



set -euo pipefail

NAME="$1"
I=$(realpath $(dirname "$0")/..)
cd $I

(
    image="rpm-test-rh8-root-$NAME"
    docker build -f ./rpm/Dockerfile.redhat8 -t "$image" --build-arg="NAME=$NAME" .
    docker run --user 10000:10000 --env USER=zouzou --rm "$image" /config/tests/test.sh "$NAME" "root"
)&
pid1=$!

(
image="rpm-test-rh8-non-root-$NAME"
docker build -f ./rpm/Dockerfile.redhat8.non-root -t "$image" --build-arg="NAME=$NAME" .
docker run --user 10000:10000 --env USER=zouzou --rm "$image" /config/tests/test.sh "$NAME" "non-root"
)&
pid2=$!

(
image="rpm-test-suze15-root-$NAME"
docker build -f ./rpm/Dockerfile.suze15 -t "$image" --build-arg="NAME=$NAME" .
docker run --user 10000:10000 --env USER=zouzou --rm "$image" /config/tests/test.sh "$NAME" "root"
)&
pid3=$!

(
image="rpm-test-suze15-non-root-$NAME"
docker build -f ./rpm/Dockerfile.suze15.non-root -t "$image" --build-arg="NAME=$NAME" .
docker run --user 10000:10000 --env USER=zouzou --rm "$image" /config/tests/test.sh "$NAME" "non-root"
)&
pid4=$!

wait $pid1
wait $pid2
wait $pid3
wait $pid4
