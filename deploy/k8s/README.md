# KMU Hub — Kubernetes Deployment

> **Status:** Scaffold only. Not production-ready yet.
> Currently using Docker Compose on Hetzner VPS for beta.

## When to migrate to K8s

- Multiple customers requiring isolation (multi-tenant)
- Need for auto-scaling beyond a single server
- High-availability requirements (99.9%+ uptime)

## Structure

```
k8s/
  base/              # Base Kustomize manifests
  overlays/
    staging/          # Staging-specific patches
    production/       # Production-specific patches
  ingress/            # Ingress controller config
```

## Prerequisites

- k3s or managed Kubernetes (Hetzner Cloud)
- kubectl + kustomize
- Container registry (GitHub Container Registry or Hetzner Registry)
- cert-manager for TLS
- sealed-secrets or external-secrets for secret management

## Migration from Docker Compose

1. Push images to container registry
2. Apply base manifests: `kubectl apply -k overlays/production/`
3. Configure ingress with TLS
4. Migrate PostgreSQL data
5. Update DNS
6. Verify health endpoints
