#!/bin/bash
#

gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" "$@"

