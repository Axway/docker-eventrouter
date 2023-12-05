# Axway Event Router Docker

## Before you begin

This document assumes a basic understanding of core Docker concepts such as containers, container images, and basic Docker commands.
If needed, see [Get started with Docker](https://docs.docker.com/get-started/) for a primer on container basics or [Docker Compose Overview](https://docs.docker.com/compose/) for details on using Docker Compose.

### Prerequisites

- Docker version 17.11 or higher

## How to use the Axway Event Router compose.yml file

The compose.yml describes the Axway Event Router service. This file allows management of the Axway Event Router service.

### How to use the official Axway Event Router image

1) Download the Axway Event Router DockerImage package from [Axway Support](https://support.axway.com/).

2) Load the image.

From the folder where the AxwayEventRouter_3.0.XXXXXXXX_DockerImage.tar.gz is located, run the command:

```console
docker image load -i AxwayEventRouter_3.0.XXXXXXXX_DockerImage.tar.gz
```

3) Check that the image is successfully loaded.

Run the command:

```console
docker images --filter reference=eventrouter/eventrouter
```

You should get an output like:
```console

REPOSITORY                TAG                 IMAGE ID            CREATED             SIZE
eventrouter/eventrouter   3.0.XXXXXXXX           27a34f72a7a4        18 hours ago        29.3MB
```

### Configuring Axway Event Router

All the configuration of the Axway Event Router is done via the file qlt-router.yml.
Information on how to write this file can be found in the Axway Event Router's documentation.

### How to manage the Axway Event Router service from your compose.yml file

You can use docker compose to automate application deployment and customization.

#### 1. Customization

Before you start, customize the parameters in the compose.yml.

Set the image parameter to match the image you want to use. For example: "image: eventrouter/eventrouter:3.0.XXXXXXXX".

#### 2. Data persistence

The Axway Event Router compose.yml template file mounts a host's directory as a data directory inside the container.

You can change this file to use volumes, as described in [Volumes configuration reference](https://docs.docker.com/compose/compose-file/#volume-configuration-reference) and [Create and manage volumes](https://docs.docker.com/storage/volumes/#create-and-manage-volumes).

#### 3. Create and start the Axway Event Router service

From the folder where the compose.yml file is located, run the command:

```console
docker compose up
```

The `up` command builds (if needed), recreates, starts, and attaches to a container for services.
Unless they are already running, this command also starts any linked services.

You can use the -d option to run containers in the background.

```console
docker compose up -d
```

You can use the -V option to recreate anonymous volumes instead of retrieving data from the previous containers.

```console
docker compose up -V
```

Run the `docker ps` command to see the running containers.

```console
docker ps
```

#### 4. Stop and remove the Axway Event Router service

From the folder where the compose.yml file is located, you can stop the containers using the command:

```console
docker compose down
```

The `down` command stops containers, and removes containers, networks, anonymous volumes, and images created by `up`.
You can use the -v option to remove named volumes declared in the `volumes` section of the Compose file, and anonymous volumes attached to containers.

#### 5. Start the Axway Event Router service

From the folder where the compose.yml file is located, you can start the Axway Event Router service using `start` if it was stopped using `stop`.

```console
docker compose start
```

#### 6. Stop Axway Event Router service

From the folder where the compose.yml file is located, you can stop the containers using the command:

```console
docker compose stop
```

### Customization

#### SSL Certificates

To specify your SSL certificates as the targets' certificate and a Axway Event Router server certificate, use the following variables:

TO COMPLETE

For example:
```
service:
    eventrouter:
        environment:
            TARGET1_CA_CERT:       "/run/secrets/target1_ca_cert.pem"
            ER_CERTIFICATE_FILE:   "/run/secrets/eventrouter.p12"
            ER_CERT_PASSWORD_FILE: "/run/secrets/eventrouter_p12.pwd"
secrets:
    target1_ca_cert.pem:
        file: ./conf/target1_ca_cert.pem
    eventrouter.p12:
       file: ./conf/eventrouter.p12
    eventrouter_p12.pwd:
        file: ./conf/eventrouter_p12.pwd
```

If one of the specified certificates has changed, when the container starts it is automatically updated so that the container always uses the certificate located in the local directory.

## Copyright

Copyright (c) 2023 Axway Software SA and its affiliates. All rights reserved.

## License

All files in this repository are licensed by Axway Software SA and its affiliates under the Apache License, Version 2.0, available at http://www.apache.org/licenses/.
