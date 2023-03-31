#!/bin/bash
#



set -euo pipefail

I=$(realpath $(dirname "$0")/..)
cd $I

docker build ./agent-rpm -t fm-agent-tests
docker run --user 10000:10000 --env USER=zouzou --rm fm-agent-tests /config/tests/test.sh

