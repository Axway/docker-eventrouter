include .env
#export $(shell sed 's/=.*//' .env)

VERSION := ${VERSION}
NAME := event-router
DATE := $(shell date +'%Y-%m-%d_%H:%M:%S')
BUILD := $(shell git rev-parse HEAD | cut -c1-8)
#LDFLAGS :=-ldflags '-s -w -X=main.Version=$(VERSION) -X=main.Build=$(BUILD) -X=main.Date=$(DATE)'
LDFLAGS :=-ldflags '-s -w  -X=main.Version=$(VERSION) -X=main.Build=$(BUILD) -X=main.Date=$(DATE)'
IMAGE := $(NAME)
REGISTRY := registry.dctest.docker-cluster.axwaytest.net/internal
PUBLISH := $(REGISTRY)/$(IMAGE)

.PHONY: docker all certs deps

all: build

pack:
	tar cvfJ $(NAME)-$(VERSION).tar.xz ./event-router ./README.*.md

build:
	(cd src/main ; CGO_ENABLED=0 go build -o ../../$(NAME) $(LDFLAGS))

build-musl:
	(cd src/main ; CGO_ENABLED=0 go build -o ../../$(NAME) -tags musl $(LDFLAGS))

build-glibc:
	(cd src/main ; CGO_ENABLED=0 go build -o ../../$(NAME) $(LDFLAGS))

build-linux-x86:
	(cd src/main ; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../$(NAME) $(LDFLAGS))

docker-external-up:
	docker-compose -f docker-compose-external.yml up -d

docker-external-down:
	docker-compose -f docker-compose-external.yml down --remove-orphans -v

#dev: build
#	find ./src -name "*.go" | entr ./event-router --config ./event-router.conf

dev:
	ls -d ./event-router.conf src/* | entr -r sh -c "make && ./$(NAME) --config ./$(NAME).conf "

docker-test:
	./scripts/run-integration-test-local.sh
	# docker-compose -f docker-compose.test.yml down
	# docker-compose -f docker-compose.test.yml build
	# docker-compose -f docker-compose.test.yml run sut  || (docker-compose -f docker-compose.test.yml logs -t | sort -k 3 ; docker-compose -f docker-compose.test.yml down ; exit 1)
	# docker-compose -f docker-compose.test.yml down

docker-test-logs:
	docker-compose -f docker-compose.test.yml logs

clean:
	rm -f $(NAME) $(NAME).tar.gz

test-integration:
	# go test -v -timeout=5s ./src/...
	# go test --cover --short --timeout 5s ./src/...
	CGO_ENABLED=0 gotestsum --junitfile report.xml --format testname --raw-command go test --cover --timeout 12s --tags musl  --coverprofile=coverage.txt --covermode=atomic --coverpkg "$(shell go list ./src/...  | tr '\n' ",")" --json ./src/... 
	go tool cover -func coverage.txt
	go run github.com/boumenot/gocover-cobertura < coverage.txt > coverage.xml
	go-cover-treemap -coverprofile coverage.txt  > coverage.svg

test-unit:
	# go test --cover --short --timeout 5s ./src/...
	# -coverpkg $(go list ./src/...) 
	gotestsum --junitfile report.xml --format testname --raw-command go test --cover --short --timeout 12s --tags musl --coverprofile=coverage.txt --covermode=atomic --coverpkg "$(shell go list ./src/...  | tr '\n' ",")" --json ./src/... 
	go tool cover -func coverage.txt
	go run github.com/boumenot/gocover-cobertura < coverage.txt > coverage.xml
	go-cover-treemap -coverprofile coverage.txt  > coverage.svg

test-specific:
	go test -v $$(ls *.go | grep -v "_test.go") $(ARGS)

deps-install:
	go mod download

docker-run:
	docker-compose up

docker:
	docker build -t $(IMAGE) docker

docker-publish-all: docker-publish docker-publish-version

docker-login:
	echo "${DOCKER_PASSWORD}" | docker login -u "${DOCKER_USERNAME}" --password-stdin

docker-publish-version:
	docker tag $(IMAGE) $(PUBLISH):$(VERSION)
	docker push $(PUBLISH):$(VERSION)

docker-publish: docker
	docker tag $(IMAGE) $(PUBLISH):latest
	docker push $(PUBLISH):latest

certs:
	openssl genrsa -out certs/server.key 2048
	openssl req -new -x509 -sha256 -key certs/server.key -out certs/server.pem -days 3650 -subj "/C=FR/ST=Paris/L=La Defense/O=Axway/CN=event-router"
	openssl x509 -text -noout -in certs/server.pem
	#cp certs/server.pem tests/test/certs/event-router.pem

