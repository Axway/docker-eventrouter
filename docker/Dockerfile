FROM alpine:edge AS build
RUN apk add --no-cache --update go make git
WORKDIR /app/src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/go/pkg/mod go install gotest.tools/gotestsum@latest
ENV PATH="/root/go/bin:${PATH}"
RUN --mount=type=cache,target=/go/pkg/mod go install github.com/nikolaydubina/go-cover-treemap@latest
RUN --mount=type=cache,target=/go/pkg/mod go get github.com/boumenot/gocover-cobertura

COPY .git/ .env Makefile ./
COPY ./src/ ./src/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build-musl

RUN /app/src/event-router version

FROM alpine
RUN apk add --no-cache ca-certificates && \
    adduser -D -g '' -h /opt/axway -H -u 1000 -G root axway && \
    mkdir -p /opt/axway/data && \
    chown -R axway:0 /opt/axway && \
    chmod -R u+x /opt/axway && \
    chmod -R g=u /opt/axway /etc/passwd

COPY --from=build /app/src/event-router /usr/bin/event-router
COPY --chown=axway:0 ./docker/uid_entrypoint /opt/axway

USER 1000
WORKDIR /opt/axway
ENV HOME=/opt/axway
ENV LANG=C.UTF-8

ARG BUILD_DATE
ARG VERSION="3.0"
ARG BUILD_REVISION

LABEL title="Axway Event Router"
LABEL created="${BUILD_DATE}"
LABEL version="${VERSION}"
LABEL revision="${BUILD_REVISION}"
LABEL vendor="Axway"
LABEL url="https://www.axway.com"
LABEL maintainer="support@axway.com"

EXPOSE 1325
EXPOSE 8080

ENTRYPOINT [ "/opt/axway/uid_entrypoint" ]
CMD [ "event-router" ]
