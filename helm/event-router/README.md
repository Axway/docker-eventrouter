# Sentinel Event Router's Helm templates for Kubernetes

## Introduction

This chart bootstraps a Sentinel Event Router deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

  - Kubernetes 1.14+
  - Helm 2.16+
  - Helm 3+

## Installing the Chart

To install the chart with the release name `event-router`:

```console
$ helm install --name event-router ./event-router
```

The command deploys Sentinel Event Router on the Kubernetes cluster in the default configuration. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `event-router` deployment:

```console
$ helm delete event-router
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

**For the cert files or license file, when you desire to use local files you can put all in the path event-router/conf or using a custom path to do it (e.g. ../../config/certs/myfile.p12)**

The following table lists the configurable parameters of the Sentinel Event Router chart and their default values.

Parameter | Description | Default
--- | --- | ---
`replicaCount` | Number of replicas deployed | `1`
`image.repository` | Image repository for docker image | `eventrouter/eventrouter`
`image.tag` | Image tag used for the deployment | `2.4.0-SP3`
`image.pullPolicy` | Pull Policy Action for docker image | `IfNotPresent`
`image.imagePullSecrets` | Secret used for Pulling image | `regcred`
`nameOverride` | New name use for the deployment | `nil`
`fullnameOverride` | Name use for the release | `nil`
`podLabels` | Additional labels | `nil`
`resources` | CPU/memory resource requests/limits | `{"requests":{"cpu":"100m","memory":"280Mi"}}`
`livenessProbe.periodSeconds`     | How often to perform the probe                                                               | 10
`livenessProbe.successThreshold`  | Minimum consecutive successes for the probe to be considered successful after having failed. | 1
`livenessProbe.failureThreshold`  | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3
`readinessProbe.periodSeconds`    | How often to perform the probe                                                               | 10
`readinessProbe.successThreshold` | Minimum consecutive successes for the probe to be considered successful after having failed. | 1
`readinessProbe.failureThreshold` | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | 3
`serviceAccount.create` | Create custom service account for the deployment | `false`
`serviceAccount.name` | Service Account name used for the deployment | `nil`
`rbac.create` | Create custom role base access control (RBAC) for the deployment | `false`
`pspEnable.create` | Create custom pod security policy for user account | `false`
`podAnnotations` | Annotations for pods (example prometheus scraping) | `{}`
`podSecurityContext` | User used no root inside the container | `{}`
`containerSecurityContext` | Restriction inside the pod | `{}`
`priorityClassName` | Name of the priority class to be used | `nil`
`nodeSelector` | Label used to deploy on specific node | `{}`
`tolerations` | Toleration are applied to pods, and allow (but do not require) the pods to schedule onto nodes with matching taints | `[]`
`affinity` | Affinity rules between each pods | `{}`
`eventrouter.maxIncomingConenctions` | The maximum number of source applications that can simultaneously connect to the Event Router. | `1000`
`eventrouter.messageSize` | The maximum length of a message. | `10000`
`eventrouter.maxNumberMessages` | The maximum number of messages that the overflow and batch files can store. | `10000`
`eventrouter.relay` | Whether or not the Sentinel Event Router is a relay between another Sentinel Event Router and the final destination. | `false`
`eventrouter.logLevel` | The Logging Level of the DISP entity. From 0 to 4. | `0`
`eventrouter.ssl.enabled` | Enables TCP/IP port with security profile. | `false`
`eventrouter.ssl.cipherSuite` | List of algorithms supported (Up to eight cipher suites, separated by comma.). The list must be in decreasing order of preference. | `156,60,47`
`eventrouter.ssl.minVersion` | List of accepted protocol versions. Values: ssl_3.0, tls_1.0, tls_1.1 or tls_1.2. | `tls_1.2`
`eventrouter.ssl.cert.secretName` | Name of the secret used to store the Event Router Server certificate (secretname is mandatory) | `eventrouter-cert`
`eventrouter.ssl.cert.createSecretFile` | Create the Event Router Server certificate secret at installation using a local file | `false`
`eventrouter.ssl.cert.localFile` | Relative path to the Event Router Server certificate (you can use conf directory in the helm chart) | `{} (eg. conf/eventrouter.p12)`
`eventrouter.ssl.cert.existingSecretFile` | Name of an existing secret to use | `{}`
`eventrouter.ssl.certPassword.secretName` | Name of the secret used to store the Event Router Server certificate password (secretname is mandatory) | `eventrouter-cert-password`
`eventrouter.ssl.certPassword.createSecretFile` | Create the Event Router Server certificate password at installation using a local file | `false`
`eventrouter.ssl.certPassword.localFile` | Relative path to the Event Router Server certificate password (you can use conf directory in the helm chart) | `{} (eg. conf/eventrouter_p12.pwd)`
`eventrouter.ssl.certPassword.existingSecretFile` | Name of an existing secret to use | `{}`
`eventrouter.userTargetXML.fileName` | Name of the user defined target.xml file (filename is mandatory). The Target Parameters File is an XML file used to set the target parameters, such as routing rules, for specific Event Router targets. | `targets-xml`
`eventrouter.userTargetXML.createConfigMap` | Create a configmap for the user defined target.xml file. | `false`
`eventrouter.userTargetXML.localFile` | Relative path to the user defined target.xml file (you can use conf directory in the helm chart) | `{} (eg. conf/target.xml)`
`eventrouter.userTargetXML.existingSecretFile` | Name of an existing configmap to use | `{}`
`eventrouter.defaultTarget.logLevel` | The Logging Level. From 0 to 4. | `0`
`eventrouter.defaultTarget.maxMessages` | The maximum number of messages that the overflow and batch files can store. | `10000`
`eventrouter.defaultTarget.port` | TCP/IP port of the default target | `1305`
`eventrouter.defaultTarget.address` | The TCP/IP address of the default target | `sentinel`
`eventrouter.defaultTarget.backupPort` | TCP/IP port of the default backup target | `1305`
`eventrouter.defaultTarget.backupAddress` | The TCP/IP address of the default backup target | `sentinel`
`eventrouter.defaultTarget.timeout` | The number of seconds that the Event Router waits for targets to acknowledge receipt of a message that the Event Router sends. | `5`
`eventrouter.defaultTarget.shortWait` | The number of seconds in the short wait. | `10`
`eventrouter.defaultTarget.longWait` | The number of seconds in the long wait. | `300`
`eventrouter.defaultTarget.jumpWait` | The number of seconds in the jump wait. | `20`
`eventrouter.defaultTarget.keepConnection` | The number of seconds that the Event Router maintains the connections to targets after successfully sending messages to the targets. | `30`
`eventrouter.defaultTarget.heartbeat` | The number of minutes between successive emissions of HeartBeat Event messages from the Event Router to the default target. | `0`
`eventrouter.defaultTarget.ssl.enabled` | Enable security profile between the Event Router and the default target. | `false`
`eventrouter.defaultTarget.ssl.cipherSuite` | List of algorithms supported (Up to eight cipher suites, separated by comma.). The list must be in decreasing order of preference. | `156,60,47`
`eventrouter.defaultTarget.ssl.minVersion` | List of accepted protocol versions. Values: ssl_3.0, tls_1.0, tls_1.1 or tls_1.2. | `tls_1.2`
`eventrouter.defaultTarget.ssl.cert.secretName` | Name of the secret used to store the CA certificate of the default target (secretname is mandatory). | `default-ca-cert`
`eventrouter.defaultTarget.ssl.cert.createSecretFile` | Create the CA certificate of the default target secret at installation using a local file | `false`
`eventrouter.defaultTarget.ssl.cert.localFile` | Relative path to the CA certificate of the default target (you can use conf directory in the helm chart) | `{} (eg. conf/default_target_ca_cert.p12)`
`eventrouter.defaultTarget.ssl.cert.existingSecretFile` | Name of an existing secret to use | `{}`
`eventrouter.targets` | List of Event router's targets with their parameters. Note that the first target of the list will be used as the default target if the default is not defined.| `[{"name": "sentinel","maxMessages": 10000,"port": 1305,"address": "sentinel","timetout": 5,"shortWait": 10,"longWait": 300,"jumpWait": 20,"keepConnection": 30,"heartbeat": 0,"ssl.enabled": false}]`
`persistence.enabled` | Enable config persistence using PVC | `true`
`persistence.keep` | Keep persistent volume after helm delete | `false`
`persistence.eventrouterData.storageClass` | Persistent Volume Claim Storage Class | `nil`
`persistence.eventrouterData.accessMode` | Persistent Volume Claim Access Mode for volume | `ReadWriteOnce`
`persistence.eventrouterData.size` | Persistent Volume Claim Storage Request for config volume (see information on resources to chose the good value for your application) | `4Gi`
`persistence.eventrouterData.existingClaim` | Manually managed Persistent Volume Claim | `nil`
`persistence.eventrouterData.nfsPath` | Basepath of the mount point to be used | `nil`
`persistence.eventrouterData.nfsServer` | Hostname of the NFS server | `nil (ip or hostname)`
`persistence.eventrouterData.reclaimPolicy` | Retain, recycle or delete. Only NFS support recycling | `retain`
`persistence.eventrouterData.mountOptions` | Mount options for NFS | `nil`
`extraEnv` | Additional environment variables | `[]`
`service.type` | Create dedicated service for the deployment LoadBalancer, ClusterIP or NodePort | `LoadBalancer`
`service.port` | The TCP/IP listening port for Event Router services | `1325`
`service.nodePort` | When using a NodePort service, you can specify a port from the range 30000-32767 to access the Event Router services | `nil`
`service.hostPort` | Port to expose the Event Router services to the external network. | `nil`
`service.annotations` | Custom annotations for service | `{}`

These parameters can be passed via Helm's `--set` option
```console
$ helm install --name event-router ./event-router \
  --set image.repository=eventrouter/eventrouter \
  --set image.tag=2.4.0-SP3
  --set resources={ "limits":{"cpu":"1000m","memory":"600Mi"},"requests":{"cpu":"200m","memory":"300Mi"}}
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```console
$ helm install --name event-router ./event-router -f my-values.yaml
```

> **Tip**: You can modify and use the default [values.yaml](values.yaml)

## Resources
The resources needed for Sentinel Event Router to run correctly depends on how Sentinel Event Router is used.

#### Disk space (MB)
The needed disk space is given by the followind equation:
Disk space (MB) = eventrouter.maxNumberMessages * 0.004 * Number of targets + 0.100

Number of targets | maxNumberMessages | Disk space (MB)
--- | --- | ---
 1 | 1,000,000 | 4100
 2 | 100,000 | 900
 4 | 500,000 | 8100
