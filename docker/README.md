# AMPLIFY Sentinel Event Router Docker

AMPLIFY Sentinel Event Router 2.4.0 SP4 Docker image

## Before you begin

This document assumes a basic understanding of core Docker concepts such as containers, container images, and basic Docker commands.
If needed, see [Get started with Docker](https://docs.docker.com/get-started/) for a primer on container basics.

### Prerequisites

- Docker version 17.11 or higher

## How to use the Sentinel Event Router Dockerfile file

The Dockerfile contains all commands required to assemble a Sentinel Event Router image.

### Dockerfile parameters

The following parameters are available in the Dockerfile file. Use these parameters to customize the Sentinel Event Router image and service. The values can be a string, number, or null.
  
 **Parameter**             |  **Values**  |  **Description**
 ------------------------- | :----------: | --------------- 
ER_NAME                    |  \<string>   |  Name of the Event Router instance. (The maximum length of this name is 50 characters.)
ER_LOG_LEVEL               |  \<number>   |  The Logging Level of the DISP entity. By default, the value of this parameter is 1. From 0 to 4.
ER_MESSAGE_SIZE            |  \<number>   |  The maximum length of a message.
ER_MAX_MESSAGES            |  \<number>   |  The maximum number of messages that the overflow and batch files can store.
ER_RELAY                   |  \<number>   |  
ER_INCOMING_MAX            |  \<number>   |  The maximum number of source applications that can simultaneously connect to the Event Router.
ER_HEALTHCHECK_INTERVAL    |  \<number>   |  Time between healthchecks. In seconds.

## How to build the Docker image

### 1. Build the Docker image from your Dockerfile

#### 1.1. Build using a local Sentinel Event Router package

1) Download the Sentinel Event Router package product package from [Axway Support](https://support.axway.com/).

The Dockerfile is compatible with Sentinel Event Router 2.4.0 SP3 version and higher.

From the [Axway Support](https://support.axway.com/), download the latest package for linux-x86-64.

2) Build the Docker image from your Dockerfile.

From the folder where the Dockerfile is located, using the downloaded package as a build argument, run the command:
```console
docker build --build-arg INSTALL_KIT=SentinelEventRouter_2.4.0_SP3_linux-x86-64_BN12252000.jar -t eventrouter/eventrouter:2.4.0-SP3 .
```

#### 1.2. Build using a Sentinel Event Router package stored on your own HTTP server

1) Download the Sentinel Event Router package product package from [Axway Support](https://support.axway.com/).

The Dockerfile is compatible with Sentinel Event Router 2.4.0 SP3 version and higher.

From the [Axway Support](https://support.axway.com/), download the latest package for linux-x86-64.

2) Build the Docker image from your Dockerfile.

From the folder where the Dockerfile is located, run the command:

```console
docker build --build-arg URL_BASE=https://network.package.location/ -t eventrouter/eventrouter:2.4.0-SP3 .
```
*Note* You can customize the VERSION_BASE, RELEASE_BASE arguments from the Dockerfile to build a Docker image based on a different Sentinel Event Router version/level.

### 2. Check that the Docker image is successfully created

Run the command:

```console
docker images --filter reference=eventrouter/eventrouter
```

You should get an output like:
```console

REPOSITORY                  TAG                 IMAGE ID            CREATED             SIZE
eventrouter/eventrouter     2.4.0-SP3         d9d764b02cc8        18 hours ago          167MB
```

## Copyright

Copyright (c) 2022 Axway Software SA and its affiliates. All rights reserved.

## License

All files in this repository are licensed by Axway Software SA and its affiliates under the Apache License, Version 2.0, available at http://www.apache.org/licenses/.

