FROM alpine:edge AS build
RUN apk add --no-cache --update go gcc g++ make git
WORKDIR /app/src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/go/pkg/mod go install gotest.tools/gotestsum@latest
ENV PATH="/root/go/bin:${PATH}"
RUN --mount=type=cache,target=/go/pkg/mod go install github.com/nikolaydubina/go-cover-treemap@latest
RUN --mount=type=cache,target=/go/pkg/mod go get github.com/boumenot/gocover-cobertura
RUN mkdir -p /opt/axway && \
    chmod -R u+x /opt/axway && \
    chgrp -R 0 /opt/axway && \
    chmod -R g=u /opt/axway

COPY .git/ .env Makefile ./
COPY ./src/ ./src/


RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build-musl

USER 1000
ENV LANG=C.UTF-8

COPY --chown=axway:0 ./docker/uid_entrypoint /opt/axway
RUN /app/src/event-router version

FROM alpine
RUN apk add --no-cache ca-certificates && \
    addgroup -g 1000 axway && \
    adduser -D -g '' -h /opt/axway -H -u 1000 -G axway axway

COPY --from=build /app/src/event-router /usr/bin/event-router
COPY --from=build /opt/axway /opt/axway

USER 1000
ENV LANG=C.UTF-8
ENV HOME=/opt/axway

ENTRYPOINT [ "/opt/axway/uid_entrypoint" ]
CMD [ "event-router" ]
