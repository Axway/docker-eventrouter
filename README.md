# qlt-router

## Contribute

### Development Prerequisites

- go
- make
- docker / docker-compose

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

###  

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