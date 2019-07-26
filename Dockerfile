FROM golang:alpine AS build
RUN apk add --no-cache make git ca-certificates
WORKDIR /app/src
COPY Makefile .deps ./
RUN make deps-install

COPY .git ./
COPY ./src/ ./src/
RUN make

FROM alpine

ENV QLT_PORT 1883
ENV QLT_HOST 0.0.0.0

COPY --from=build /app/src/qlt-router /usr/bin/qlt-router

CMD [ "qlt-router" ]
