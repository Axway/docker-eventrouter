# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2021 Axway Software SA and its affiliates. All rights reserved.
#
# AMPLIFY Sentinel Event Router 2.4.0 SP3 Docker image
#
# Building with:
# docker build -f Dockerfile -t axway/er:2.4.0 .

#
# Stage 1 : build
#

FROM ubuntu:latest AS builder

RUN apt-get update && apt-get install -y \
        unzip \
        vim && \
        rm -rf /var/lib/apt/lists && \
        mkdir -p /opt/axway && \
        addgroup axway && \
        adduser --disabled-password --gecos '' --home /opt/axway --no-create-home --ingroup axway axway && \
        chown -R axway:axway /opt/axway

USER axway
WORKDIR /opt/axway
ENV LANG=C.UTF-8

# Download and install Sentinel Event Router package

ARG PACKAGE="SentinelEventRouter_2.4.0_SP4_linux-x86-64_BN13484000.jar"
ARG URL_BASE="https://delivery.server.com/"
ARG INSTALL_KIT="${URL_BASE}${PACKAGE}"

ADD --chown=axway:axway $INSTALL_KIT installkit.jar

RUN unzip installkit.jar  -d er && \
    mv er/component/* er && \
    rm -rf setup installkit.jar er/component && \
    chmod -R 755 er/bin && \
    mkdir data

# Copying useful scripts

COPY --chown=axway:axway resources/*.sh ./
COPY --chown=axway:axway resources/uid_entrypoint ./

#
# Stage 2 : create final image
#

FROM ubuntu:latest

COPY --from=builder /opt/axway /opt/axway

RUN apt-get -y update && apt-get upgrade -y && apt-get install -y \
        vim \
        tzdata && \
        rm -rf /var/lib/apt/lists && \
        addgroup axway && \
        adduser --disabled-password --gecos '' --home /opt/axway --no-create-home --ingroup axway axway && \
        chown -R axway:0 /opt/axway && \
        chmod -R u+x /opt/axway && \
        chmod -R g=u /opt/axway /etc/passwd

USER axway
WORKDIR /opt/axway
ENV HOME=/opt/axway
ENV LANG=C.UTF-8

ARG BUILD_DATE
ARG BUILD_VERSION="2.4.0_SP4"
ARG BUILD_REVISION="13484000"

LABEL created="${BUILD_DATE}"
LABEL url="https://www.axway.com"
LABEL vendor="Axway"
LABEL maintainer="support@axway.com"
LABEL title="Sentinel Event Router"
LABEL version="${BUILD_VERSION}"
LABEL revision="${BUILD_REVISION}"

# Exposed ports
EXPOSE 1325

# Environment variables

ENV ER_INSTALLDIR               /opt/axway/er
ENV ER_NAME                     SentinelEventRouter
ENV ER_LOG_LEVEL                0
ENV ER_MESSAGE_SIZE             64000
ENV ER_MAX_MESSAGES             10000
ENV ER_RELAY                    0
ENV ER_INCOMING_MAX             20
ENV ER_USE_SSL                  NO
ENV ER_HEALTHCHECK_INTERVAL     10

# Entry point

ENTRYPOINT [ "/opt/axway/uid_entrypoint" ]

CMD [ "./start.sh" ]

HEALTHCHECK --interval=1m \
            --timeout=5s \
            --start-period=30s \
            --retries=3 \
            CMD bash -c /opt/axway/status.sh
