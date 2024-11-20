This page describes downloading, customizing, deploying, and managing the Axway Event Router. The compose.yml describes the Axway Event Router service and allows you to manage it.

## Prerequisites

This document assumes a basic understanding of core Docker concepts such as containers, container images, and basic Docker commands. See [Get started with Docker](https://docs.docker.com/get-started/) for a primer on container basics or  [Docker Compose Overview](https://docs.docker.com/compose/)  for details on using Docker Compose. Additionally, you require:

*   Docker 17.11 or higher
*   Meet the sizing requirements:
    *   Disk space for data persistence: Calculate based on the file writers configured with the maximum file size \* maximum number of files
    *   Memory: 128 MB
*   If you plan to use SSL/TLS, please see [Set up certificates and keys](https://markdowntohtml.com/#set%20up%20certificates%20and%20keys)
*   Disk space for the Docker image: 30 MB

No license key is required.

## Download the Axway Event Router package

1.  Download the EventRouter\_3.0\_DockerImage\_linux-x86-64.tar.gz package from [Axway Repository](https://repository.axway.com/catalog?products=a1E7S000003RwTJUA0&versions=a1F7S000001x9GGUAY).

2.  To load the image the AxwayEventRouter\_3.0.XXXXXXXX\_DockerImage.tar.gz, run:

    ```console
    docker image load -i AxwayEventRouter_3.0.XXXXXXXX_DockerImage.tar.gz
    ```

3.  Check that the image is successfully loaded:

    ```console
    docker images --filter reference=eventrouter/eventrouter
    ```

    Expected output:
    
    ```console

    REPOSITORY                TAG                 IMAGE ID            CREATED             SIZE
    eventrouter/eventrouter   3.0.XXXXXXXX        27a34f72a7a4        18 hours ago        29.3MB
    ```

## Configure the Event Router

The ./conf/event-router.yml file is responsible for all Axway Event Router configurations. See the  [Axway Event Router](https://docs.axway.com/bundle?cluster=true&exclude_metadata_filter.field=display-type&exclude_metadata_filter.value=inline&labelkey=prod-sentinel-420&rpp=20&sort.field=title&sort.value=asc)  documentation for details on editing this file.

## Deploy the Event Router service using Compose

You can use the Docker Compose compose.yml file to automate application deployment and customization.

1.  Customize the parameters in the compose.yml. Be sure to set the image parameter to match the image you are using. For example, "image: eventrouter/eventrouter:3.0.XXXXXXXX".

 **Parameter**              |  **Values**  |  **Description**
 -------------------------- | :----------: | --------------- 
 ACCEPT_GENERAL_CONDITIONS  |  "YES"/"NO"  | Set to YES to accept the  General Terms and Conditions. See https://www.axway.com/en/legal/contract-documents
 ER_LOG_LEVEL               |  \<string>   | Defines log level. Supported values: trace, debug, info, warn, error, fatal
 ER_CONFIG_FILE             |  \<string>   | Sets the configuration file pathname. (Default is '/opt/axway/event-router.yml')
 ER_PORT                    |  \<number>   | HTTP server port
 ER_HOST                    |  \<string>   | Host address to listen to. (Default is '0.0.0.0')
 ER_LOG_USE_LOCALTIME       | \<boolean>   | Log uses local time

2.  Set data persistence.  
    The Axway Event Router compose.yml file defines a volume as a mechanism for persisting data generated and used by the Event Router. However, this volume is only needed if the Event Router uses files (file-writer and file-reader). You can change the volume configuration to use a previously created volume, as described in [Volumes top-level element](https://docs.docker.com/compose/compose-file/07-volumes) and [Create and manage volumes](https://docs.docker.com/storage/volumes/#create-and-manage-volumes).

3.  Create and run the Axway Event Router service. From the compose.yml file, run:
    ```console
    docker compose up -d
    ```
    The `up` command builds, recreates, starts, and attaches to a container for services. Unless they are already running, this command also starts any linked services.

## Manage the Event Router service

Run the `docker ps` command to see the running containers.

```console
docker ps
```

Run the `docker compose logs` command to see the container logs.

```console
docker compose logs
```

#### Stop the Event Router service

From the compose.yml file, stop the containers using the command:

```console
docker compose stop
```

#### Start the Event Router service

From the compose.yml file, start the Axway Event Router service using `start` if it was stopped using `stop`.

```console
docker compose start
```

#### Stop and remove the Event Router service

From  the compose.yml file, stop the containers using the command:

```console
docker compose down
```

The `down` command stops and removes containers, networks, anonymous volumes, and images created by `up`. You can use the -v option to remove named volumes declared in the `volumes` section of the Compose file and anonymous volumes attached to containers.

## Set up certificates and keys

The compose.yml mounts certificates and keys as Docker secrets and defines associated environment variables that refer to your event-router.yml configuration file.

The secrets section maps the files on the host in the ./conf directory to the secrets qlt\_server\_cert.pem (an x509 certificate) and qlt\_server\_key.pem (its private key).

```secrets:
  qlt_server_cert.pem:
    file: ./conf/qlt_server_cert.pem
  qlt_server_key.pem:
    file: ./conf/qlt_server_key.pem
```
To enable the eventrouter service secrets:

```services:
  eventrouter:
    ...
    environment:
      ER_QLT_SERVER_CERT: /run/secrets/qlt_server_cert.pem
      ER_QLT_SERVER_KEY: /run/secrets/qlt_server_key.pem
    ...
    secrets:
    - qlt_server_cert.pem
    - qlt_server_key.pem
    
```
The environment variables ER_QLT_SERVER_CERT and ER_QLT_SERVER_KEY refer to the enabled secrets. You can use them in your stream definition in the event-router.yml file as follows:

```...
streams:
  - name: "QLT_input"
    disable: false
    description: "QLT to file"
    upstream: ""
    reader:
      type: qlt-server-reader
      conf:
        port: 1325
        cert: $ER_QLT_SERVER_CERT
        certKey: $ER_QLT_SERVER_KEY
```

## Support arbitrary user IDs

The Axway Event Router image is compatible with OpenShift, which means it can be run with a random user ID (UID). When OpenShift starts the container, it automatically assigns a random UID but always uses a group ID (GID) of 0 (the root group). If you want to run the container with a specific user instead of the default `axway` user (UID=1000), you must set the GID to 0. Attempting to use a different GID will result in the container exiting with an error.

In OpenShift, the container typically uses a random UID for security reasons. However, if you need the container to run with a specific UID, for example, to mount folders from the host system on Linux, you can configure it manually to match your host user’s UID.

In a **Docker Compose** environment, you can modify the user running the container by setting the `user` directive in your `docker-compose.yml` file. For example, you can set the `user` entry to use a specific UID and GID, or leave it as the default to have it run with the image’s default UID (1000). If you change the UID, ensure the GID is still set to 0. If the UID is not `1000` (the default `axway` user), Docker automatically creates the user in the container.

## Copyright

Copyright (c) 2024 Axway Software SA and its affiliates. All rights reserved.

## License

All files in this repository are licensed by Axway Software SA and its affiliates under the Apache License, Version 2.0, available at http://www.apache.org/licenses/.
