# inboxfewer Helm Chart

This Helm chart deploys the inboxfewer MCP server on a Kubernetes cluster.

## Overview

inboxfewer is a Model Context Protocol (MCP) server that provides AI assistants with programmatic access to Gmail, Google Docs, Google Drive, Google Calendar, Google Meet, and Google Tasks.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+

## Installing the Chart

To install the chart with the release name `my-inboxfewer`:

```bash
helm install my-inboxfewer ./charts/inboxfewer
```

## Uninstalling the Chart

To uninstall/delete the `my-inboxfewer` deployment:

```bash
helm delete my-inboxfewer
```

## Configuration

The following table lists the configurable parameters of the inboxfewer chart and their default values.

### Application Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/teemow/inboxfewer` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (overrides chart appVersion) | `""` |
| `config.transport` | Transport type (stdio or streamable-http) | `streamable-http` |
| `config.httpAddr` | HTTP server address | `:8080` |
| `config.yolo` | Enable write operations (default: read-only) | `false` |
| `config.debug` | Enable debug logging | `false` |
| `config.disableStreaming` | Disable streaming for HTTP transport | `false` |

### Google OAuth Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `google.clientId` | Google OAuth Client ID | `""` |
| `google.clientSecret` | Google OAuth Client Secret | `""` |
| `existingSecret` | Name of existing secret with OAuth credentials | `""` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Kubernetes Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.targetPort` | Container port | `8080` |

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.hosts` | Ingress hosts | `[{host: inboxfewer.local, paths: [{path: /, pathType: Prefix}]}]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Resource Limits

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

### Autoscaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable horizontal pod autoscaling | `false` |
| `autoscaling.minReplicas` | Minimum number of replicas | `1` |
| `autoscaling.maxReplicas` | Maximum number of replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization | `80` |

## Examples

### Basic Installation

```bash
helm install inboxfewer ./charts/inboxfewer
```

### With Google OAuth Credentials

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set google.clientId="your-client-id" \
  --set google.clientSecret="your-client-secret"
```

### With Ingress Enabled

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=inboxfewer.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

### With Write Operations Enabled

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set config.yolo=true
```

### Using Existing Secret

Create a secret with Google OAuth credentials:

```bash
kubectl create secret generic inboxfewer-oauth \
  --from-literal=google-client-id="your-client-id" \
  --from-literal=google-client-secret="your-client-secret"
```

Then install with:

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set existingSecret=inboxfewer-oauth
```

## Using the Image from GitHub Container Registry

The chart is configured to use images from GitHub Container Registry (ghcr.io). Images are automatically built and pushed by the GitHub Actions workflow when code is pushed to the main branch or tags are created.

To use a specific version:

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set image.tag=v1.0.0
```

To use the latest version from main:

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set image.tag=latest
```

## Security

The chart follows security best practices:

- Runs as non-root user (UID 1000)
- Read-only root filesystem
- Drops all capabilities
- Does not allow privilege escalation

## Support

For issues and questions, visit: https://github.com/teemow/inboxfewer/issues


