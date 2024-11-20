Using the Helm chart package manager, you can use the delivered chart to bootstrap an Axway Event Router deployment on a  [Kubernetes](http://kubernetes.io/) or [Red Hat OpenShift](https://www.openshift.com/) cluster.

## Prerequisites

*   Kubernetes 1.14 or higher, or Red Hat OpenShift 4.12 or higher
*   Helm 3 or higher

## Installing the chart

To install the chart with the release name `event-router`:

```console
$ helm install --name event-router ./event-router
```

The command deploys Axway Event Router in the default configuration on the Kubernetes or Red Hat OpenShift cluster. The  [configuration](https://markdowntohtml.com/#configuration)  section lists the parameters that can be configured during installation.

List all releases using the command: `helm list`

  

## Uninstalling the chart

To uninstall/delete the `event-router` deployment:

```console
$ helm delete event-router
```

The command removes all the Kubernetes or Red Hat OpenShift components associated with the chart and deletes the release.

## Configuration

**If you want to use local files for the certicates, you can put all in the path `event-router/conf`, or use a custom path (e.g. `../../config/certs/myfile.p12`)**

The following table lists the configurable Axway Event Router chart parameters and their default values.

Parameter | Description | Default
--- | --- | ---
`replicaCount` | Number of replicas deployed | `1`
`image.repository` | Image repository for docker image | `docker.repository.axway.com/sentineleventrouter-docker-prod/3.0/event-router`
`image.tag` | Image tag used for the deployment | `3.0`
`image.pullPolicy` | Pull Policy Action for docker image | `IfNotPresent`
`image.pullSecrets` | Secret used for Pulling image | `[{"name": "regcred"}]`
`nameOverride` | New name use for the deployment | `nil`
`fullnameOverride` | Name use for the release | `nil`
`podLabels` | Additional labels | `nil`
`resources` | CPU/memory resource requests/limits | `{"requests":{"cpu":"100m","memory":"280Mi"}}`
`livenessProbe.periodSeconds`     | How often to perform the probe | 10
`livenessProbe.successThreshold`  | Minimum consecutive successes for the probe to be considered successful after having failed. | 1
`livenessProbe.failureThreshold`  | Minimum consecutive failures for the probe to be considered failed after having succeeded. | 3
`readinessProbe.periodSeconds`    | How often to perform the probe | 10
`readinessProbe.successThreshold` | Minimum consecutive successes for the probe to be considered successful after having failed. | 1
`readinessProbe.failureThreshold` | Minimum consecutive failures for the probe to be considered failed after having succeeded. | 3
`serviceAccount.create` | Create custom service account for the deployment | `false`
`serviceAccount.name` | Service Account name used for the deployment | `nil`
`rbac.create` | Create custom role base access control (RBAC) for the deployment | `false`
`pspEnable` | Create custom pod security policy for user account | `false`
`podAnnotations` | Annotations for pods (example prometheus scraping) | `{}`
`podSecurityContext` | User used no root inside the container | `{"runAsUser": 1000, "runAsGroup": 0,"fsGroup": 1000}`
`containerSecurityContext` | Restriction inside the pod | `{"runAsNonRoot": true, "runAsUser": 1000,"runAsGroup": 0}`
`priorityClassName` | Name of the priority class to be used | `nil`
`nodeSelector` | Label used to deploy on specific node | `{}`
`tolerations` | Toleration are applied to pods, and allow (but do not require) the pods to schedule onto nodes with matching taints | `[]`
`affinity` | Affinity rules between each pods | `{}`
`event-router.accept_general_conditions` | Set parameter to `true` if you accept the applicable General Terms and Conditions, located at https://www.axway.com/en/legal/contract-documents | `false`
`event-router.fqdn` | A fully qualified domain name (FQDN) or an  IP address used to connect to your Event Router deployment. | ``
`event-router.instanceId` |  | `eventrouter-1`
`event-router.logLevel` | Log level. Supported values: trace, debug, info, warn, error, fatal | `info`
`event-router.logUseLocalTime` | The log uses local time. | `false`
`event-router.httpPort.port` | Port used to expose Prometheus metrics | `8080`
`event-router.configuration.fileName` | Name of the user configuration file (filename is mandatory) | `event-router-yml`
`event-router.configuration.createConfigMap` | Create a configmap for the user configuration file | `false`
`event-router.configuration.localFile` | Relative path to the user configuration file. You can use the conf directory in the Helm chart. | `{}` (eg. `conf/event-router.yml`)
`event-router.configuration.existingConfigMap.keyRef` | Name of the reference key inside an existing configmap | `{}`
`event-router.certificates` | Certificates definitions for the Event Router service | `[]`
`persistence.enabled` | Enable config persistence using PVC | `true`
`persistence.keep` | Keep persistent volume after helm delete | `false`
`persistence.eventrouterData.storageClass` | Persistent Volume Claim Storage Class | `nil`
`persistence.eventrouterData.accessMode` | Persistent Volume Claim Access Mode for volume | `ReadWriteOnce`
`persistence.eventrouterData.size` | Persistent Volume Claim Storage Request for config volume (see information on resources to chose the good value for your application) | `4Gi`
`persistence.eventrouterData.existingClaim` | Manually managed Persistent Volume Claim | `nil`
`persistence.eventrouterData.nfsPath` | Basepath of the mount point to be used | `nil`
`persistence.eventrouterData.nfsServer` | The hostname of the NFS server | `nil (ip or hostname)`
`persistence.eventrouterData.reclaimPolicy` | Retain, recycle or delete. Only NFS support recycling | `retain`
`persistence.eventrouterData.mountOptions` | Mount options for NFS | `nil`
`extraVolumeMounts` | Additionnal volume mounts to add | `[]`
`extraEnv` | Additional environment variables | `[]`
`service.type` | Create dedicated service for the deployment LoadBalancer, ClusterIP or NodePort | `ClusterIP`
`service.ports` | Ports definitions for the service. | `[{"name": "qltIn","port": 1325}]`
`service.annotations` | Custom annotations for the service. | `{}`

These parameters can be passed via Helm's `--set` option
```console
$ helm install --name event-router ./event-router \
  --set image.repository=docker.repository.axway.com/sentineleventrouter-docker-prod/3.0/eventrouter \
  --set image.tag=3.0.202411
  --set resources={"limits":{"cpu":"1","memory":"512Mi"},"requests":{"cpu":"0.2","memory":"128Mi"}}
```

Alternatively, you can provide a YAML file that specifies the parameter values during the chart installation. For example:

```console
$ helm install --name event-router ./event-router -f my-values.yaml
```

You can modify and use the default [values.yaml](./values.yaml).

#### Previously created secrets and configmaps
To use previously created secrets and configmaps, create secrets as follows:
```console
kubectl create secret generic <secret_name> --from-file=<key_reference>=<path_to_file>
```
or
```console
kubectl create secret generic <secret_name> --from-literal=<key_reference>=<secret_value>
```
This gives you the following configuration in the `values.yaml`:

```console
secretName: <secret_name>
createSecretFile: false
existingSecretFile:
  keyRef: <key_reference>
```

Similarly, create the configmaps as:

```console
kubectl create configmap <configmap_name> --from-file=<key_reference>=<path_to_file>
```

or

```console
kubectl create configmap <configmap_name> --from-literal=<key_reference>=<secret_value>
```

This gives you the following configuration in the `values.yaml`:

```console
fileName: <secret_name>
createConfigMap: false
existingConfigMap:
  keyRef: <key_reference>
```
in the [`values.yaml`](./values.yaml).

## Support arbitrary user IDs
Axway Event Router image is OpenShift compatible, which means that you can start it with a random user ID (UID) and the group ID (GID) of the root user (GID=0). If you want to run the image with a user other than the default one, axway (UID=1000), you MUST set the user's GID to 0. If you try to use a different group, the container exits with errors. OpenShift randomly assigns a UID when it starts the container, but you can also use this flexible UID when running the image manually. This might be useful if you want to mount folders from the host system on Linux; the UID should be set to the same ID as your host user.

In a Kubernetes or OpenShift environment, you can modify this using the `runAsUser` and `runAsGroup` entries in the `value.yml` file.If the GID is set to 0, the user can be any UID. If the UID is not 1000 (axway), the user is automatically created when entering the container.

## Copyright

Copyright (c) 2024 Axway Software SA and its affiliates. All rights reserved.

## License

All files in this repository are licensed by Axway Software SA and its affiliates under the Apache License, Version 2.0, available at http://www.apache.org/licenses/.

