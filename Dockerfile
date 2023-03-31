FROM alpine:edge AS build
RUN apk add --no-cache --update go gcc g++ make git
WORKDIR /app/src
COPY go.mod go.sum ./
RUN go mod download

COPY .git .env Makefile ./
COPY ./src/ ./src/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make
RUN /app/src/qlt-router version

FROM alpine
RUN apk add --no-cache ca-certificates

COPY --from=build /app/src/qlt-router /usr/bin/qlt-router

CMD [ "qlt-router" ]
