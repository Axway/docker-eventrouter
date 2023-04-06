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

```sh
docker-compose -f docker-compose-external.yml
make test
# or
go test -v -t 10 ./src/...
```

### local ci

```sh
gitlab-runner exec shell build-bin
gitlab-runner exec shell build-docker
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

#### CHeck for vulnarimbilities
```sh
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```