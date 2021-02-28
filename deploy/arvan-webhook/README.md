# Cert Manager Webhook Arvan

If you're using [cert-manager](https://cert-manager.io) in your k8s and [ArvanCloud](https://www.arvancloud.com/en) as a dns provider, this plugin integrates with arvancloud api service to make DNS01 challenge possible for you.

## Introduction

This chart setups a weebhook for [cert-manager](https://cert-manager.io) to resolve dns challenges on [ArvanCloud](https://www.arvancloud.com/en).

## Prerequisites

- Kubernetes 1.19+
- Helm 3.1.0
- cert-manager v1.2.0

## Installing the Chart

To install the chart with the release name `my-release`:

Use helm to install this webhook:

```console
$ helm repo add hbx https://hbahadorzadeh.github.io/helm-chart/
$ helm install my-release -n cert-manager hbx/cert-manager-webhook-arvan
```


These commands deploy webohook on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release -n cert-manager
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

The following tables lists the configurable parameters of the webhook chart and their default values.

### Common parameters

| Parameter                 | Description                                     | Default                                                 |
| ------------------------- | ----------------------------------------------- | ------------------------------------------------------- |
| `image.repository`        | Docker image registry                           | `hbahadorzadeh/cert-manager-webhook-arvan`              |
| `image.tag`               | Docker image tag                                | `latest`                                                |
| `image.pullPolicy`        | Deployment pull policy                          | `IfNotPresent`                                          |
| `affinity`                | Affinity for pod assignment (evaluated as a template)                                                                | `{}`                           |
| `nodeSelector`            | Node labels for pod assignment                                                                                       | `{}` (evaluated as a template) |
| `tolerations`             | Tolerations for pod assignment                                                                                       | `[]` (evaluated as a template) |
| `resources`               | The resources limits for the container                                                                               | `{}` (evaluated as a template) |

### Webhook parameters

| Parameter                | Description                                                                                                          | Default            |
| ------------------------ | ---------------------------------------------------------------------------------------------------- | --------------------------- |
| `groupName`              | ApiService Group name                                                                                | `hbahadorzadeh.github`      |
| `nameOverride`           | String to partially override airflow.fullname template with a string (will prepend the release name) | `nil`                          || `fullnameOverride`       | ApiService Group name                           | `hbahadorzadeh.github`      |
| `fullnameOverride`       | String to fully override airflow.fullname template with a string                                     | `nil`                          |
| `deployment.loglevel`    | Defines output log level                                                                             | `2`                          |
| `credentialsSecretRef`   | String to set as name for secret used to store private key                            | `arvan-credentials`                          |
| `service.type`           | String defines service type                                                        | `ClusterIP`                          |
| `service.port`           | String defines service port                                                        | `443`                          |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm install my-release \
               --set deployment.loglevel=6 \
               hbx/cert-manager-webhook.arvan
```

The above command sets the loglevel.

> NOTE: Once this chart is deployed, it is not possible to change the application's access credentials, such as usernames or passwords, using Helm. To change these application credentials after deployment, delete any persistent volumes (PVs) used by the chart and re-deploy it, or use the application's built-in administrative tools if available.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```console
$ helm install my-release -f values.yaml hbx/cert-manager-webhook.arvan
```

> **Tip**: You can use the default [values.yaml](values.yaml)

### Setting Pod's affinity

This chart allows you to set your custom affinity using the `affinity` parameter. Find more information about Pod's affinity in the [kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity).

As an alternative, you can use of the preset configurations for pod affinity, pod anti-affinity, and node affinity available at the [bitnami/common](https://github.com/bitnami/charts/tree/master/bitnami/common#affinities) chart. To do so, set the `podAffinityPreset`, `podAntiAffinityPreset`, or `nodeAffinityPreset` parameters.
## Troubleshooting

Find more information about how to deal with common errors related to Bitnami’s Helm charts in [this troubleshooting guide](https://docs.bitnami.com/general/how-to/troubleshoot-helm-chart-issues).

### 0.1.0

First release! :)
