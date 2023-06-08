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
docker-compose -f docker-compose-external.yml
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
gitlab-runner exec docker --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" build
gitlab-runner exec docker --docker-volumes "$PWD/artefacts:/artefacts" rpm
./rpm/test-rpm.sh 
gitlab-runner exec docker build-docker
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
