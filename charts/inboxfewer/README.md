# inboxfewer Helm Chart

This Helm chart deploys the inboxfewer MCP server on a Kubernetes cluster.

## Overview

inboxfewer is a Model Context Protocol (MCP) server that provides AI assistants with programmatic access to Gmail, Google Docs, Google Drive, Google Calendar, Google Meet, and Google Tasks.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+

## Installing the Chart

### From OCI Registry (Recommended)

The chart is automatically published to GitHub Container Registry:

```bash
# Install latest version
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer

# Install specific version
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer --version 0.1.0

# Install with custom values
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set image.tag=v1.2.3 \
  --values my-values.yaml
```

### From Local Source

For development and testing:

```bash
helm install my-inboxfewer ./charts/inboxfewer
```

### Feature Branch Testing

Feature branch charts include the branch name in the version:

```bash
helm install inboxfewer-test \
  oci://ghcr.io/teemow/charts/inboxfewer \
  --version 0.1.0-feature-xyz-abc123
```

## Uninstalling the Chart

To uninstall/delete the `my-inboxfewer` deployment:

```bash
helm delete my-inboxfewer
```

## ⚠️ Security Best Practices

### **CRITICAL: Never Use `--set` for Secrets in Production!**

**DO NOT** pass secrets via command line or commit them to values files:

```bash
# ❌ UNSAFE - Secrets exposed in shell history and Helm history
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set google.clientSecret="my-secret-123"

# ❌ UNSAFE - Never commit secrets to version control
# values.yaml
google:
  clientSecret: "my-secret-123"  # DON'T DO THIS!
```

**DO** use Kubernetes secrets or external secret managers:

```bash
# ✅ SAFE - Create secret separately
kubectl create secret generic inboxfewer-oauth \
  --from-literal=google-client-id="your-client-id" \
  --from-literal=google-client-secret="your-client-secret"

# ✅ SAFE - Reference existing secret
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set existingSecret=inboxfewer-oauth
```

**Recommended Secret Management Solutions:**
- [External Secrets Operator](https://external-secrets.io/) - Sync secrets from external providers (AWS Secrets Manager, Azure Key Vault, GCP Secret Manager, HashiCorp Vault)
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) - Encrypt secrets in Git
- [SOPS](https://github.com/mozilla/sops) - Encrypt secrets with age/PGP

### Image Verification

All images are scanned with Trivy for vulnerabilities. To verify image integrity:

```bash
# Check latest scan results in GitHub Security tab
# https://github.com/teemow/inboxfewer/security/code-scanning

# Verify image digest
docker pull ghcr.io/teemow/inboxfewer:v1.2.3
docker inspect ghcr.io/teemow/inboxfewer:v1.2.3 --format='{{.RepoDigests}}'
```

### Network Security

Enable NetworkPolicy for defense-in-depth:

```yaml
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
      - podSelector:
          matchLabels:
            app: allowed-client
  egress:
    - to:
      - namespaceSelector: {}
      ports:
      - protocol: TCP
        port: 443  # HTTPS for Google/GitHub APIs
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

### Health Probes

| Parameter | Description | Default |
|-----------|-------------|---------|
| `livenessProbe.enabled` | Enable liveness probe | `true` |
| `livenessProbe.initialDelaySeconds` | Initial delay before probe starts | `15` |
| `livenessProbe.periodSeconds` | How often to perform probe | `20` |
| `livenessProbe.timeoutSeconds` | Probe timeout | `5` |
| `livenessProbe.failureThreshold` | Failure threshold | `3` |
| `readinessProbe.enabled` | Enable readiness probe | `true` |
| `readinessProbe.initialDelaySeconds` | Initial delay before probe starts | `5` |
| `readinessProbe.periodSeconds` | How often to perform probe | `10` |
| `readinessProbe.timeoutSeconds` | Probe timeout | `3` |
| `readinessProbe.failureThreshold` | Failure threshold | `3` |

**Note:** Health probes use TCP socket checks by default, which verify the port is accepting connections.

### Ingress Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.hosts` | Ingress hosts | `[{host: inboxfewer.local, paths: [{path: /, pathType: Prefix}]}]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

**Note:** The chart uses both the annotation (`kubernetes.io/ingress.class`) and spec field (`spec.ingressClassName`) for maximum compatibility with both legacy and modern ingress controllers.

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

### Pod Disruption Budget

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podDisruptionBudget.enabled` | Enable PodDisruptionBudget | `false` |
| `podDisruptionBudget.maxUnavailable` | Maximum unavailable pods during disruptions | `1` |

### Network Policy

| Parameter | Description | Default |
|-----------|-------------|---------|
| `networkPolicy.enabled` | Enable NetworkPolicy for pod network isolation | `false` |
| `networkPolicy.policyTypes` | Policy types (Ingress, Egress) | `[Ingress, Egress]` |
| `networkPolicy.ingress` | Ingress rules (who can connect to this pod) | See values.yaml |
| `networkPolicy.egress` | Egress rules (where this pod can connect) | See values.yaml |

**Note:** NetworkPolicy requires a CNI plugin that supports it (Calico, Cilium, Weave Net). It provides defense-in-depth by restricting network access to/from pods.

### Grafana Dashboards

| Parameter | Description | Default |
|-----------|-------------|---------|
| `grafanaDashboards.enabled` | Enable Grafana dashboard ConfigMaps | `false` |
| `grafanaDashboards.namespace` | Namespace for dashboard ConfigMaps (defaults to release namespace) | `""` |
| `grafanaDashboards.labels` | Labels for Grafana sidecar discovery | `{grafana_dashboard: "1"}` |
| `grafanaDashboards.annotations` | Additional annotations for dashboard ConfigMaps | `{}` |
| `grafanaDashboards.folder` | Grafana folder for dashboards | `Inboxfewer` |
| `grafanaDashboards.datasources.prometheus` | Prometheus data source name | `Prometheus` |
| `grafanaDashboards.datasources.loki` | Loki data source name | `Loki` |
| `grafanaDashboards.datasources.tempo` | Tempo data source name | `Tempo` |
| `grafanaDashboards.dashboards.administrator.enabled` | Enable Administrator dashboard | `true` |
| `grafanaDashboards.dashboards.security.enabled` | Enable Security Operations dashboard | `true` |
| `grafanaDashboards.dashboards.endUser.enabled` | Enable End-User dashboard | `true` |

**Note:** Dashboard ConfigMaps are created with labels that match the default Grafana sidecar configuration in kube-prometheus-stack. The sidecar automatically imports dashboards from ConfigMaps with the `grafana_dashboard: "1"` label.

### Volumes and Storage

| Parameter | Description | Default |
|-----------|-------------|---------|
| `volumes` | Additional volumes for the pod | `[{name: cache, emptyDir: {}}]` |
| `volumeMounts` | Additional volume mounts for the container | `[{name: cache, mountPath: /home/inboxfewer/.cache}]` |

**Note:** The default configuration includes a cache volume (`emptyDir`) to allow OAuth token storage with `readOnlyRootFilesystem: true`. Tokens are ephemeral and lost on pod restart. For persistent tokens, replace `emptyDir` with a `persistentVolumeClaim`.

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

### With NetworkPolicy Enabled

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set networkPolicy.enabled=true
```

**Note:** Ensure your Kubernetes cluster supports NetworkPolicy (requires compatible CNI plugin).

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

### With Grafana Dashboards

Enable automatic dashboard provisioning for Grafana (requires kube-prometheus-stack or similar with sidecar):

```bash
helm install inboxfewer ./charts/inboxfewer \
  --set grafanaDashboards.enabled=true \
  --set grafanaDashboards.namespace=monitoring \
  --set grafanaDashboards.datasources.prometheus="Prometheus" \
  --set grafanaDashboards.datasources.loki="Loki"
```

This creates ConfigMaps containing three dashboards:
- **Administrator** - Service health, performance, and Kubernetes resources
- **Security Operations** - Audit trails, anomaly detection, and incident investigation
- **End-User** - AI agent activity visibility and tool usage transparency

**Note:** The Grafana sidecar must be configured to watch the namespace where dashboards are created. By default, kube-prometheus-stack watches all namespaces for ConfigMaps with the `grafana_dashboard: "1"` label. Adjust `grafanaDashboards.datasources.*` to match your configured data source names.

### With Persistent OAuth Token Storage

By default, OAuth tokens are stored in an `emptyDir` volume and lost on pod restart. To persist tokens across restarts:

```bash
# Create a PVC for token storage
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: inboxfewer-cache
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF

# Install with persistent cache
helm install inboxfewer ./charts/inboxfewer \
  --set volumes[0].name=cache \
  --set volumes[0].persistentVolumeClaim.claimName=inboxfewer-cache \
  --set volumeMounts[0].name=cache \
  --set volumeMounts[0].mountPath=/home/inboxfewer/.cache
```

## Versioning Strategy

The Helm chart follows **independent versioning** best practices:

- **Chart Version** (`version` in Chart.yaml): Only incremented when chart templates or configuration change
- **App Version** (`appVersion` in Chart.yaml): **Automatically updated** during each release to match the application version
- **Image Tag**: Defaults to `appVersion`, users can override with `--set image.tag=v1.2.3`

### Automatic AppVersion Updates

The `appVersion` in `Chart.yaml` is **automatically updated** by the auto-release workflow:

1. PR is merged to main
2. Auto-release workflow determines next version (e.g., `v1.2.3`)
3. **Workflow updates `appVersion` in Chart.yaml to `1.2.3`**
4. Changes are committed to main
5. Git tag is created and release is published
6. Docker images are built with matching version

This ensures that:
- ✅ Default image tag is always a specific, pinned version (not `latest`)
- ✅ Deployments are reproducible and predictable
- ✅ No manual updates required
- ✅ Chart appVersion always matches the latest release

### When to Bump Chart Version

Chart version should only be incremented when:
- ✅ Chart templates are modified
- ✅ New configuration options are added
- ✅ Dependencies change
- ✅ Breaking changes to chart usage

Chart version should **NOT** be bumped when:
- ❌ Only the application version changes (appVersion is auto-updated)
- ❌ Only documentation updates
- ❌ Container image updates (without chart changes)

### Specifying Application Version

```bash
# Use the default version (appVersion from Chart.yaml)
# This will be the latest release version (e.g., 1.2.3)
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer

# Pin to a specific older version
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set image.tag=v1.2.2

# Use a specific chart version with specific app version
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --version 0.1.1 \
  --set image.tag=v1.2.3
```

## Automated Publishing

Both the Helm chart and container images are automatically published to GitHub Container Registry (GHCR) via GitHub Actions:

### Container Images

**Release Images** (`.github/workflows/docker-release.yml`):
- Triggered after successful releases
- Multi-architecture: linux/amd64, linux/arm64
- Uses pre-built binaries from GoReleaser
- Tags: `latest`, `v1.2.3`, `v1.2`, `v1`

**Feature Branch Images** (`.github/workflows/docker-build.yml`):
- Built on every PR and feature branch push
- Single architecture: linux/amd64 (faster CI)
- Built from source code
- Tags: `pr-42`, `feature-branch-name`, `sha-abc123`

### Helm Charts

Charts are published when changes are detected in `charts/**`:
- Main branch: Stable versions (e.g., `0.1.0`)
- Feature branches: Test versions (e.g., `0.1.0-feature-xyz-abc123`)

### Using Specific Versions

**Production (pinned version):**
```bash
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --version 0.1.0 \
  --set image.tag=v1.2.3
```

**Production (latest):**
```bash
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set image.tag=latest
```

**Feature branch testing:**
```bash
helm install inboxfewer-test oci://ghcr.io/teemow/charts/inboxfewer \
  --version 0.1.0-feature-xyz-abc123 \
  --set image.tag=feature-xyz
```

## Security

The chart follows security best practices:

- Runs as non-root user (UID 1000)
- Read-only root filesystem
- Drops all capabilities
- Does not allow privilege escalation
- Service account token auto-mount disabled (app doesn't need Kubernetes API access)
- Rolling updates with zero downtime (maxUnavailable: 0, maxSurge: 1)
- TCP health probes enabled by default for pod health monitoring

## Documentation

For comprehensive deployment information, see:
- **[Deployment Guide](../../docs/deployment.md)** - Complete guide to Docker, Kubernetes, and Helm deployments
- **[Development Guide](../../docs/development.md)** - Development workflows and release process
- **[Configuration Guide](../../docs/configuration.md)** - Application configuration

## Support

For issues and questions, visit: https://github.com/teemow/inboxfewer/issues


