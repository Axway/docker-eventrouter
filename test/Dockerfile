# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2021 Axway Software SA and its affiliates. All rights reserved.
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
        curl \
        sed \
        netcat \
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
ARG RELEASE_BASE="BN10968000"
ARG PACKAGE="SentinelUniversalAgent_${VERSION_BASE}_Install_linux-x86-64_${RELEASE_BASE}.zip"
ARG URL_BASE="https://delivery.server.com/"
ARG INSTALL_KIT="${URL_BASE}${PACKAGE}"

ADD --chown=axway:axway $INSTALL_KIT installkit.zip


#####
# DOWNLOAD AND INSTALL PRODUCTS

ENV UA_INSTALLDIR /opt/axway/ua
RUN unzip installkit.zip -d p && \
    unzip p/Components/SentinelUniversalAgent_V${VERSION_BASE}/SentinelUniversalAgent_${VERSION_BASE}_linux-x86-64.jar -d ua && \
    rm -rf p installkit.jar && \
    chmod -R 755 ua/bin

COPY test.sh .
CMD [ "./test.sh" ]
