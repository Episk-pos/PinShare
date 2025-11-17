# Kustomize Overlays

This directory contains environment-specific configurations using Kustomize overlays.

## Structure

```
k8s/
├── base/                   # Base manifests (shared across all environments)
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── ipfs/
│   ├── pinshare-backend/
│   ├── pinshare-ui/
│   └── oauth-broker/
├── overlays/
│   ├── dev/               # Development environment (local K8s with Tilt)
│   │   ├── kustomization.yaml
│   │   ├── oauth-broker-secret.yaml       # Localhost OAuth URLs
│   │   └── pinshare-backend-secret.yaml
│   └── production/        # Production environment (Flux GitOps)
│       ├── kustomization.yaml
│       ├── oauth-broker-secret.yaml       # Production OAuth URLs
│       └── pinshare-backend-secret.yaml
```

## Environment Differences

### Development (Tilt)
- **OAuth Broker URL**: `http://localhost:8888`
- **OAuth Redirect URL**: `http://localhost:8888/callback`
- **Backend API**: `http://localhost:9090`
- **IPFS API**: `http://localhost:5001`
- **IPFS Gateway**: `http://localhost:8080`

### Production (Flux)
- **OAuth Broker URL**: `https://oauth.episkopos.community`
- **OAuth Redirect URL**: `https://oauth.episkopos.community/callback`
- **Backend API**: `https://psbackend.episkopos.community`
- **IPFS API**: `https://ipfs.episkopos.community/api/v0`
- **IPFS Gateway**: `https://ipfs.episkopos.community/ipfs`

## Usage

### Development (Tilt)

Tilt automatically uses the dev overlay:

```bash
# From project root
tilt up
```

The Tiltfile is configured to:
1. Build kustomization from `k8s/overlays/dev`
2. Decrypt dev secrets with SOPS

### Production (Flux)

Update your Flux Kustomization to use the production overlay:

```yaml
# In infra repo: clusters/production/apps.yaml or similar
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: pinshare
  namespace: flux-system
spec:
  interval: 5m
  path: ./k8s/overlays/production  # <-- Use production overlay
  sourceRef:
    kind: GitRepository
    name: pinshare
  prune: true
  decryption:
    provider: sops
    secretRef:
      name: sops-age  # Age key for decrypting secrets
```

## Managing Secrets

### Updating Secrets

**Development:**
```bash
# Edit dev secret
sops k8s/overlays/dev/oauth-broker-secret.yaml

# Or set specific value
sops --set '["stringData"]["REDIRECT_URL"] "http://localhost:8888/callback"' \
  k8s/overlays/dev/oauth-broker-secret.yaml
```

**Production:**
```bash
# Edit production secret
sops k8s/overlays/production/oauth-broker-secret.yaml

# Or set specific value
sops --set '["stringData"]["REDIRECT_URL"] "https://oauth.episkopos.community/callback"' \
  k8s/overlays/production/oauth-broker-secret.yaml
```

### Creating New Secrets

Secrets in `k8s/overlays/*/` are automatically encrypted by SOPS based on `.sops.yaml` rules:

```yaml
creation_rules:
  - path_regex: k8s/overlays/.*/.*-secret\.yaml$
    age: age140ptrsjpzlv232gavz0p5clen227nj3fq4fa8d9tgu43pjqamaxqqev5a7
```

## Testing Changes

### Test Dev Overlay Locally

```bash
# Build and view the manifests
kubectl kustomize k8s/overlays/dev

# Decrypt and view secrets
sops --decrypt k8s/overlays/dev/oauth-broker-secret.yaml
```

### Test Production Overlay

```bash
# Build production manifests
kubectl kustomize k8s/overlays/production

# Verify secret values are correct for production
sops --decrypt k8s/overlays/production/oauth-broker-secret.yaml | grep REDIRECT_URL
```

## Migration Notes

This structure was introduced to support different configurations for development and production environments, particularly for OAuth redirect URLs and service endpoints.

**Previous approach**: Single set of secrets in `k8s/secrets/` used by both environments
**New approach**: Environment-specific secrets in overlays with appropriate URLs

## Troubleshooting

### Tilt shows wrong URLs
- Check that `Tiltfile` is using `k8s/overlays/dev`
- Verify dev secrets have localhost URLs:
  ```bash
  sops --decrypt k8s/overlays/dev/oauth-broker-secret.yaml | grep REDIRECT_URL
  ```

### Production deployment fails
- Ensure Flux Kustomization points to `k8s/overlays/production`
- Verify production secrets are properly encrypted
- Check that Flux has SOPS decryption configured

### SOPS encryption fails
- Check `.sops.yaml` rules include `k8s/overlays/`
- Ensure Age key is available: `age-keygen -y ~/.sops/key.txt`
