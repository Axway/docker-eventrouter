# AMPLIFY Sentinel Event Router Docker

## Before you begin

This document assumes a basic understanding of core Docker concepts such as containers, container images, and basic Docker commands.
If needed, see [Get started with Docker](https://docs.docker.com/get-started/) for a primer on container basics.

### Prerequisites

- Docker version 17.11 or higher
- Docker-Compose version 1.17.0 or higher

## How to use the Sentinel Event Router docker-compose.yml file

The docker-compose.yml describes the Sentinel Event Router service. This file allows management of the Sentinel Event Router service.

You can use the ../docker/Dockerfile to build your own 
 or use the official Axway Sentinel Event Router image.

### docker-compose parameters

The following parameters are available in the docker-compose.yml file. Use these parameters to customize the Sentinel Event Router service. The values can be a string, number, or null.
  
 **Parameter**             |  **Values**  |  **Description**
 ------------------------- | :----------: | --------------- 
ER_RECONFIG                |  "YES"/"NO"  |  Parameter used to force reinitialization of Event Router configuration
ER_HEALTHCHECK_INTERVAL    |  \<number>   |  Time between healthchecks. In seconds.
ER_NAME                    |  \<string>   |  Name of the Event Router instance. (The maximum length of this name is 50 characters.)
ER_LOG_LEVEL               |  \<number>   |  The Logging Level of the DISP entity. By default, the value of this parameter is 1. From 0 to 4.
ER_MESSAGE_SIZE            |  \<number>   |  The maximum length of a message.
ER_MAX_MESSAGES            |  \<number>   |  The maximum number of messages that the overflow and batch files can store.
ER_RELAY                   |  "YES"/"NO"  |  Whether or not the Sentinel Event Router is a relay between another Sentinel Event Router and the final destination.
ER_INCOMING_MAX            |  \<number>   |  The maximum number of source applications that can simultaneously connect to the Event Router.
ER_PORT                    |  \<number>   |  The TCP/IP port that the Sentinel Event Router will use to receive messages.
ER_USE_SSL                 |  "YES"/"NO"  |  
ER_CERTIFICATE_FILE        |  \<string>   |  Event Router server certificate. It must refer to a PKCS12 certificate.
ER_CERT_PASSWORD_FILE      |  \<string>   |  Event Router server certificate password.
ER_SSL_CIPHER_SUITE        |  \<string>   |  List of algorithms supported (Up to eight cipher suites, separated by comma.). The list must be in decreasing order of preference.
ER_SSL_VERSION_MIN         |  \<string>   |  List of accepted protocol versions. Values: ssl_3.0, tls_1.0, tls_1.1 or tls_1.2.
TARGET1_NAME               |  \<string>   |  This will be used to create the section in the configuration file. Use capital letters. Note that the target1 will be used as the default target.
TARGET1_LOG_LEVEL          |  \<number>   |  The Logging Level of the DISP entity. By default, the value of this parameter is 1. From 0 to 4.
TARGET1_PORT               |  \<number>   |  TCP/IP port of the Target 1.
TARGET1_ADDRESS            |  \<string>   |  The TCP/IP address of the Target 1.
TARGET1_TIMEOUT            |  \<number>   |  The number of seconds that the Event Router waits for targets to acknowledge receipt of a message that the Event Router sends.
TARGET1_SHORT_WAIT         |  \<number>   |  The number of seconds in the short wait.
TARGET1_LONG_WAIT          |  \<number>   |  The number of seconds in the long wait.
TARGET1_JUMP_WAIT          |  \<number>   |  The number of seconds in the jump wait.
TARGET1_KEEP_CONNECTION    |  \<number>   |  The number of seconds that the Event Router maintains the connections to targets after successfully sending messages to the targets.
TARGET1_HEARTBEAT          |  \<number>   |  The number of minutes between successive emissions of HeartBeat Event messages from the Event Router to the target.
TARGET1_USE_SSL_OUT        |  "YES"/"NO"  |  Enable security profile between the Event Router and the Target 1.
TARGET1_CA_CERT            |  \<string>   |  CA certificate of Target 1.
TARGET1_SSL_CIPHER_SUITE   |  \<string>   |  List of algorithms supported (Up to eight cipher suites, separated by comma.). The list must be in decreasing order of preference.
TARGET1_SSL_VERSION_MIN    |  \<string>   |  List of accepted protocol versions. Values: ssl_3.0, tls_1.0, tls_1.1 or tls_1.2.
USER_TARGET_XML            |  \<string>   |  Path within the container pointing to a user defined target.xml file. The Target Parameters File is an XML file used to set the target parameters, such as routing rules, for specific Event Router targets.

**Note** The block starting by TARGET1, can be repeated for as many targets as needed, just increment the number in the parameter name. (TARGET1_NAME -> TARGET2_NAME, ...)
**Note** When using multiple targets, the parameter USER_TARGET_XML must be specified.

### How to use the official Sentinel Event Router image

1) Download the Sentinel Event Router DockerImage package from [Axway Support](https://support.axway.com/).

2) Unzip the downloaded package.

3) Load the image.

From the folder where the event_router_2.4.0_SP3.tgz

```console
docker image load -i event_router_2.4.0_SP3.tgz
```

4) Check that the image is successfully loaded.

Run the command:

```console
docker images --filter reference=eventrouter/eventrouter
```

You should get an output like:
```console

REPOSITORY                TAG                 IMAGE ID            CREATED             SIZE
eventrouter/eventrouter   2.4.0-SP3           27a34f72a7a4        18 hours ago        158MB
```

### How to manage the Sentinel Event Router service from your docker-compose.yml file

You can use docker-compose to automate application deployment and customization.

#### 1. Customization

Before you start, customize the parameters in the docker-compose.yml.

Set the image parameter to match the image you want to use. For example: "image: eventrouter/eventrouter:2.4.0-SP3".

#### 2. Data persistence

The Sentinel Event Router docker-compose.yml file defines a volume as a mechanism for persisting data generated by and used by Sentinel Event Router.
The overflow file is placed in this volume so it can be reused when creating and starting a new Sentinel Event Router container.

You can change the volume configuration to use a previously created volume. See [Volumes configuration reference](https://docs.docker.com/compose/compose-file/#volume-configuration-reference) and [Create and manage volumes](https://docs.docker.com/storage/volumes/#create-and-manage-volumes).

#### 3. Create and start the Sentinel Event Router service

From the folder where the docker-compose.yml file is located, run the command:

```console  
docker-compose up  
```

The `up` command builds (if needed), recreates, starts, and attaches to a container for services.  
Unless they are already running, this command also starts any linked services.

You can use the -d option to run containers in the background.

```console  
docker-compose up -d  
```

You can use the -V option to recreate anonymous volumes instead of retrieving data from the previous containers.

```console
docker-compose up -V
```

Run the docker `ps` command to see the running containers.

```console
docker ps
```

#### 4. Stop and remove the Sentinel Event Router service

From the folder where the docker-compose.yml file is located, you can stop the containers using the command:

```console
docker-compose down
```

The `down` command stops containers, and removes containers, networks, anonymous volumes, and images created by `up`.  
You can use the -v option to remove named volumes declared in the `volumes` section of the Compose file, and anonymous volumes attached to containers.

#### 5. Start the Sentinel Event Router service

From the folder where the docker-compose.yml file is located, you can start the Sentinel Event Router service using `start` if it was stopped using `stop`.

```console
docker-compose start
```

#### 7. Stop Sentinel Event Router service

From the folder where the docker-compose.yml file is located, you can stop the containers using the command:

```console
docker-compose stop
```

### Customization

To enable customization, you must define a mapped volume that refers to a local directory containing the targets' file.
In this example, the directory '/opt/app/custom' in the container maps the local directory './custom'. The mapped directory '/opt/app/custom' is in read-only mode.

```
volumes:
  - ./custom:/opt/app/custom:ro
```

#### SSL Certificates

To specify your SSL certificates as the targets' certificate and a Sentinel Event Router server certificate, use the following variables:
- TARGETxx_CA_CERT: The path to the CA certificate of Target xx.
- ER_CERTIFICATE_FILE: The path to the Event Router server certificate. It must refer to a PKCS12 certificate.
- ER_CERT_PASSWORD_FILE: File containing the Event Router server certificate password.

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


#### Custom targets file

For example:

```
service:
    eventrouter:
        environment:
            USER_TARGET_XML:   "/opt/app/custom/target.xml"
```

## Copyright

Copyright (c) 2021 Axway Software SA and its affiliates. All rights reserved.

## License

All files in this repository are licensed by Axway Software SA and its affiliates under the Apache License, Version 2.0, available at http://www.apache.org/licenses/.
