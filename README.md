# qlt-router

## Contribute

### Development Prerequisites

- go
- make
- docker / docker-compose
- gitlab-runner - for local testing

### build

```sh
make 
```

### test

unit-test (short)
```sh
go test --short --timeout 5s ./src/...
# or
gotestsum --junitfile report.xml --format testname --raw-command go test --short --timeout 5s --json ./src/...
```

all test (short + !short)
```sh
./scripts/run-integration-test-local.sh
# or
docker-compose -f docker-compose.external.yml
make test
# or
go test -v --timeout 10s ./src/...
# or
gotestsum --junitfile report.xml --format testname --raw-command go test --timeout 10s --json ./src/...
```

### local ci

- mac colima
```
export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
````

```sh
./scripts/gitlab-runner.sh build
./scripts/gitlab-runner.sh rpm
./scripts/gitlab-runner.sh rpm-test
./scripts/gitlab-runner.sh unit-test
./scripts/gitlab-runner.sh integration-test

gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" build
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" rpm
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" test-rpm
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" integration-test
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" build-docker
```

### add dev ui

```sh
./scripts/upkg.sh
```

### Manage dependencies

```sh
go list -u -m all # view available dependency upgrades
go get -u ./... # upgrade all dependencies at once
go get -t -u ./... # upgrade all dependencies at once (test dependencies as well)
go mod tidy
```

#### Check for vulnaribilities
```sh
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```


## Profiling

```sh
./qlt-router --cpuprofile qlt-router.prof
go tool pprof ./qlt-router qlt-router.prof

```

## Other tips

Remove colourful output
```sh
sed -e $'s/\x1b\[[0-9;]*m//g'
```

## Current coverage

[![Pipeline](https://git.ecd.axway.org/cft/qlt-router/badges/master/pipeline.svg)](https://git.ecd.axway.org/cft/qlt-router/)
[![Coverage](https://git.ecd.axway.org/cft/qlt-router/badges/master/coverage.svg)](https://git.ecd.axway.org/cft/qlt-router/)
[![Coverage](https://git.ecd.axway.org/api/v4/projects/7287/jobs/artifacts/master/raw/coverage.svg?job=integration-test)](https://git.ecd.axway.org/cft/qlt-router/)
