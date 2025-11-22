# Security Guide

This document outlines security best practices and features for deploying and operating inboxfewer.

## Container Security

### Image Scanning

All container images are automatically scanned for vulnerabilities using Trivy:

- **Feature branches**: Scanned on every push and PR
- **Release images**: Scanned during release builds
- **Critical vulnerabilities**: Builds fail if critical CVEs are found
- **Results**: Available in GitHub Security tab

To manually scan an image:

```bash
# Using Trivy
docker pull ghcr.io/teemow/inboxfewer:latest
trivy image ghcr.io/teemow/inboxfewer:latest

# View scan results in GitHub
# https://github.com/teemow/inboxfewer/security/code-scanning
```

### Base Image Security

- **Pinned digests**: Base images use SHA256 digests for reproducibility
- **Minimal images**: Alpine Linux for minimal attack surface
- **Regular updates**: Automated dependency updates via Dependabot

### Runtime Security

The container follows security best practices:

```dockerfile
# Non-root user
USER inboxfewer (UID 1000)

# Read-only root filesystem
RUN chmod -R a-w /app

# Minimal capabilities
# All capabilities dropped in Kubernetes
```

## Kubernetes Security

### Pod Security Standards

The Helm chart is compliant with **Restricted** Pod Security Standards:

```yaml
# Pod Security Context
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

# Container Security Context
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop: [ALL]
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000
```

### Network Policies

NetworkPolicy support for defense-in-depth:

```yaml
# Enable in values.yaml
networkPolicy:
  enabled: true
```

This restricts:
- **Ingress**: Only specific pods can connect
- **Egress**: Only HTTPS (443) and DNS (53) allowed

Example policy:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: inboxfewer
spec:
  podSelector:
    matchLabels:
      app: inboxfewer
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: allowed-client
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS for Google/GitHub APIs
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53  # DNS
```

### Service Account

- **Minimal permissions**: No Kubernetes API access needed
- **Token auto-mount disabled**: `automountServiceAccountToken: false`

### Resource Limits

Prevent resource exhaustion attacks:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

## Secrets Management

### ⚠️ Critical: Never Expose Secrets

**NEVER:**
- ❌ Use `--set` for secrets (exposed in shell history)
- ❌ Commit secrets to Git
- ❌ Log secrets
- ❌ Pass secrets as environment variables in debug mode

**ALWAYS:**
- ✅ Use Kubernetes Secrets
- ✅ Use external secret managers
- ✅ Rotate credentials regularly
- ✅ Use least privilege access

### Recommended Secret Management

#### 1. Kubernetes Secrets (Basic)

```bash
# Create secret
kubectl create secret generic inboxfewer-oauth \
  --from-literal=google-client-id="..." \
  --from-literal=google-client-secret="..."

# Use with Helm
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set existingSecret=inboxfewer-oauth
```

#### 2. External Secrets Operator (Recommended)

Sync secrets from AWS Secrets Manager, Azure Key Vault, GCP Secret Manager, HashiCorp Vault:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: inboxfewer-oauth
spec:
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: inboxfewer-oauth
  data:
  - secretKey: google-client-id
    remoteRef:
      key: /inboxfewer/oauth
      property: client_id
  - secretKey: google-client-secret
    remoteRef:
      key: /inboxfewer/oauth
      property: client_secret
```

#### 3. Sealed Secrets (GitOps)

Encrypt secrets for safe storage in Git:

```bash
# Install kubeseal
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml

# Create and encrypt secret
kubectl create secret generic inboxfewer-oauth \
  --from-literal=google-client-id="..." \
  --from-literal=google-client-secret="..." \
  --dry-run=client -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml

# Safe to commit
git add sealed-secret.yaml
```

### Secret Rotation

Rotate credentials regularly:

```bash
# 1. Create new credentials in Google Cloud Console
# 2. Update secret
kubectl create secret generic inboxfewer-oauth-new \
  --from-literal=google-client-id="NEW_ID" \
  --from-literal=google-client-secret="NEW_SECRET"

# 3. Update deployment
helm upgrade inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set existingSecret=inboxfewer-oauth-new

# 4. Verify working, then delete old secret
kubectl delete secret inboxfewer-oauth
```

## Network Security

### TLS/HTTPS

Always use TLS for external access:

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  tls:
  - hosts:
    - inboxfewer.example.com
    secretName: inboxfewer-tls
  hosts:
  - host: inboxfewer.example.com
    paths:
    - path: /
      pathType: Prefix
```

### Internal Communication

For internal service-to-service communication:

- Use ClusterIP services (default)
- Enable NetworkPolicy
- Consider service mesh (Istio, Linkerd) for mTLS

## Monitoring & Auditing

### Security Monitoring

Monitor for security events:

```bash
# Pod security policy violations
kubectl get events --field-selector type=Warning

# Failed authentication attempts
kubectl logs -l app=inboxfewer | grep "authentication failed"

# Resource exhaustion
kubectl top pods -l app=inboxfewer
```

### Audit Logging

Enable Kubernetes audit logging to track:
- Secret access
- Pod creation/modification
- Network policy changes

### Vulnerability Monitoring

- **GitHub Security Alerts**: Automatically enabled
- **Dependabot**: Automated dependency updates
- **Trivy**: Continuous image scanning

## Compliance

### CIS Kubernetes Benchmarks

The deployment meets CIS Kubernetes Benchmark requirements:

| Control | Status | Implementation |
|---------|--------|----------------|
| 5.2.1 Minimize privileged containers | ✅ PASS | No privileged containers |
| 5.2.2 Minimize capabilities | ✅ PASS | All capabilities dropped |
| 5.2.3 Minimize root containers | ✅ PASS | `runAsNonRoot: true` |
| 5.2.6 Minimize host networking | ✅ PASS | No host networking |
| 5.2.7 Minimize privilege escalation | ✅ PASS | `allowPrivilegeEscalation: false` |
| 5.2.8 Read-only root filesystem | ✅ PASS | `readOnlyRootFilesystem: true` |
| 5.7.3 Apply security context | ✅ PASS | Comprehensive security contexts |

### NIST 800-190

Compliant with NIST Container Security guidelines:

- ✅ Image vulnerability scanning
- ✅ Secrets management
- ✅ Network segmentation (via NetworkPolicy)
- ✅ Runtime protection (security contexts)
- ✅ Resource limits

## Incident Response

### Security Incident Procedure

1. **Detect**: Monitor security alerts
2. **Isolate**: Use NetworkPolicy to isolate affected pods
3. **Investigate**: Collect logs and forensics
4. **Remediate**: Update images, rotate credentials
5. **Review**: Post-mortem and lessons learned

### Emergency Credential Rotation

```bash
# 1. Immediately revoke compromised credentials
# In Google Cloud Console / GitHub Settings

# 2. Create new credentials
# Follow provider-specific procedures

# 3. Update Kubernetes secret
kubectl create secret generic inboxfewer-oauth-emergency \
  --from-literal=google-client-id="NEW_ID" \
  --from-literal=google-client-secret="NEW_SECRET"

# 4. Restart pods with new credentials
kubectl set env deployment/inboxfewer --from=secret/inboxfewer-oauth-emergency
kubectl rollout restart deployment/inboxfewer

# 5. Verify and clean up
kubectl delete secret inboxfewer-oauth
```

## Security Checklist

Before deploying to production:

- [ ] Enable NetworkPolicy
- [ ] Use external secret manager (not Kubernetes secrets)
- [ ] Enable TLS/HTTPS for Ingress
- [ ] Set resource limits
- [ ] Enable audit logging
- [ ] Configure monitoring and alerting
- [ ] Review and minimize RBAC permissions
- [ ] Enable Pod Security Standards
- [ ] Scan images for vulnerabilities
- [ ] Rotate default credentials
- [ ] Enable encryption at rest for secrets
- [ ] Configure backup and disaster recovery
- [ ] Document incident response procedures
- [ ] Conduct security review/penetration testing

## Reporting Security Issues

To report security vulnerabilities:

1. **DO NOT** create public GitHub issues
2. Email security concerns to the maintainers (see repository contact)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fixes (if any)

## Additional Resources

- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/security-best-practices/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [NIST 800-190: Container Security](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-190.pdf)
- [OWASP Container Security](https://owasp.org/www-project-container-security/)


