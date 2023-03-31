FROM golang:alpine AS build
RUN apk add --no-cache ca-certificates

RUN apk add --no-cache make git gcc musl-dev
WORKDIR /app/src
COPY go.mod go.sum ./
RUN go mod download

COPY .git Makefile ./
COPY ./src/ ./src/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
     make

FROM alpine
RUN apk add --no-cache ca-certificates

ENV QLT_PORT 1883
ENV QLT_HOST 0.0.0.0

COPY --from=build /app/src/qlt-router /usr/bin/qlt-router

CMD [ "qlt-router" ]
