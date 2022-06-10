# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2022 Axway Software SA and its affiliates. All rights reserved.
#
# AMPLIFY Sentinel Event Router 2.4.0 SP4 Docker image
#
# Building with:
# docker build -f Dockerfile -t axway/er:2.4.0 .

#####
# OS PREPARATION

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y \
        unzip \
        curl \
        sed \
        netcat \
        vim \
        apt-utils && \
        # docker add
        curl -sSL https://get.docker.com/ | sh && \
        rm -rf /var/lib/apt/lists && \
        mkdir -p /opt/axway && \
        adduser --uid 1000 --disabled-password --gecos '' --home /opt/axway --no-create-home --gid 0 axway && \
        usermod -aG docker axway && \
        chown -R axway:0 /opt/axway

USER 1000
WORKDIR /opt/axway
ENV LANG=C.UTF-8

#####
ARG VERSION_BASE="2.4.0"
ARG RELEASE_BASE="BN10968000"
ARG PACKAGE="SentinelUniversalAgent_${VERSION_BASE}_Install_linux-x86-64_${RELEASE_BASE}.zip"
ARG URL_BASE="https://delivery.server.com/"
ARG INSTALL_KIT="${URL_BASE}${PACKAGE}"

ADD --chown=axway:0 $INSTALL_KIT installkit.zip


#####
# DOWNLOAD AND INSTALL PRODUCTS

ENV UA_INSTALLDIR /opt/axway/ua
RUN unzip installkit.zip -d p && \
    unzip p/Components/SentinelUniversalAgent_V${VERSION_BASE}/SentinelUniversalAgent_${VERSION_BASE}_linux-x86-64.jar -d ua && \
    rm -rf p installkit.jar && \
    chmod -R 755 ua/bin && \
    chmod -R u+x /opt/axway && \
    chmod -R g=u /opt/axway

COPY test.sh .
CMD [ "./test.sh" ]
