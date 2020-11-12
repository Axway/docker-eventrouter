# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2019 Axway Software SA and its affiliates. All rights reserved.
#
## AMPLIFY Sentinel Event Router 2.4.0 SP3 Docker image
#
# Building with:
# docker build -f Dockerfile -t axway/er:2.4.0 .

#####
# OS PREPARATION

FROM ubuntu:focal

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

#####
ARG VERSION_BASE="2.4.0"
ARG VERSION_UP="SP3"
ARG RELEASE_BASE="BN13026000"
ARG PACKAGE="SentinelEventRouter_${VERSION_BASE}_${VERSION_UP}_linux-x86-64_${RELEASE_BASE}.jar"
ARG URL_BASE="https://delivery.server.com/"
ARG INSTALL_KIT="${URL_BASE}${PACKAGE}"

ADD --chown=axway:axway $INSTALL_KIT installkit.jar

#####
# LABELS
LABEL vendor=Axway
LABEL com.axway.eventrouter.os="ubuntu"
LABEL com.axway.eventrouter.version="${VERSION_BASE}-${VERSION_UP}"
LABEL com.axway.eventrouter.release-date="2020-09-25"
LABEL com.axway.ubuntu.version=rolling
LABEL maintainer="support@axway.com"

LABEL version="1.0"
LABEL description="Sentinel Event Router ${VERSION_BASE}-${VERSION_UP} Docker image"

#####
# DOWNLOAD AND INSTALL PRODUCTS

ENV ER_INSTALLDIR /opt/axway/er
RUN unzip installkit.jar  -d er && \
    mv er/component/* er && \
    rm -rf setup installkit.jar er/component && \
    chmod -R 755 er/bin && \
    mkdir data

#####
# PRODUCTS CONFIGURATION

# - ENV VARIABLES
ENV ER_NAME             SentinelEventRouter
ENV ER_LOG_LEVEL        0
ENV ER_MESSAGE_SIZE     64000
ENV ER_MAX_MESSAGES     10000
ENV ER_RELAY            0
ENV ER_INCOMING_MAX     20
ENV ER_USE_SSL          NO
ENV ER_HEALTHCHECK_INTERVAL  10

#####
# COPYING USEFUL SCRIPTS

COPY --chown=axway:axway resources/*.sh ./

#####
# START POINT

CMD [ "./start.sh" ]

HEALTHCHECK --interval=1m \
            --timeout=5s \
            --start-period=30s \
            --retries=3 \
            CMD bash -c /opt/axway/status.sh

