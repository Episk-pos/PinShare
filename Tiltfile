# -*- mode: Python -*-

# PinShare Tiltfile for local Kubernetes development
# This provides an alternative to docker-compose with better resource management,
# hot reload capabilities, and the Tilt UI for monitoring services.

# Configuration
allow_k8s_contexts('kind-pinshare')  # Adjust this to match your local K8s context
# For minikube, use: allow_k8s_contexts('minikube')
# Or comment out to allow any context

# Load Kubernetes manifests using Kustomize
# Development overlay includes localhost URLs for OAuth and other services
k8s_yaml(local('kubectl kustomize k8s/overlays/dev'))

# Decrypt dev-specific secrets with SOPS
k8s_yaml(local('sops --decrypt k8s/overlays/dev/pinshare-backend-secret.yaml'))
k8s_yaml(local('sops --decrypt k8s/overlays/dev/oauth-broker-secret.yaml'))

# Build Docker images
# Backend
# Note: live_update disabled for multi-stage builds as Go is not available in runtime stage
docker_build(
    'ghcr.io/episk-pos/pinshare-backend',
    '.',
    dockerfile='Dockerfile',
    ignore=['pinshare-ui/', 'oauth-broker/', 'k8s/', '.git/'],
)

# UI - Development mode with Vite hot reload
docker_build(
    'ghcr.io/episk-pos/pinshare-ui',
    './pinshare-ui',
    dockerfile='./pinshare-ui/Dockerfile',
    target='development',
    live_update=[
        # Sync source files to leverage Vite's hot reload
        sync('./pinshare-ui/src', '/app/src'),
        sync('./pinshare-ui/public', '/app/public'),
        sync('./pinshare-ui/index.html', '/app/index.html'),
        sync('./pinshare-ui/vite.config.js', '/app/vite.config.js'),
        sync('./pinshare-ui/tailwind.config.js', '/app/tailwind.config.js'),
        sync('./pinshare-ui/postcss.config.js', '/app/postcss.config.js'),
        # Reinstall dependencies if package.json changes
        sync('./pinshare-ui/package.json', '/app/package.json'),
        sync('./pinshare-ui/package-lock.json', '/app/package-lock.json'),
        run('cd /app && npm install', trigger=['./pinshare-ui/package.json', './pinshare-ui/package-lock.json']),
    ],
)

# OAuth Broker
docker_build(
    'ghcr.io/episk-pos/oauth-broker',
    './oauth-broker',
    dockerfile='./oauth-broker/Dockerfile',
    live_update=[
        # Sync Go source files
        sync('./oauth-broker', '/app'),
        # Rebuild on Go file changes
        run('cd /app && go build -o /usr/local/bin/oauth-broker .',
            trigger=['./oauth-broker/**/*.go', './oauth-broker/go.mod', './oauth-broker/go.sum']),
    ],
)

# Resource labels for organization in Tilt UI
k8s_resource(
    'ipfs',
    port_forwards=[
        '5001:5001',   # IPFS API
        '8080:8080',   # IPFS Gateway
        '4001:4001',   # IPFS Swarm
    ],
    labels=['infrastructure'],
)

k8s_resource(
    'pinshare-backend',
    port_forwards=[
        '9090:9090',   # PinShare API
        '50001:50001', # libp2p
    ],
    labels=['backend'],
    resource_deps=['ipfs'],  # Backend depends on IPFS
)

k8s_resource(
    'pinshare-ui',
    port_forwards='5174:5174',
    labels=['frontend'],
    resource_deps=['pinshare-backend'],
)

k8s_resource(
    'oauth-broker',
    port_forwards='8888:8888',
    labels=['auth'],
)

# Local registry (optional, uncomment if using a local registry)
# default_registry('localhost:5000')

print("""
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║  PinShare Development Environment with Tilt                   ║
║                                                                ║
║  Services will be available at:                               ║
║  • UI:            http://localhost:5174                       ║
║  • Backend API:   http://localhost:9090                       ║
║  • IPFS API:      http://localhost:5001                       ║
║  • IPFS Gateway:  http://localhost:8080                       ║
║  • OAuth Broker:  http://localhost:8888                       ║
║                                                                ║
║  Architecture:                                                 ║
║  • IPFS runs as separate StatefulSet                          ║
║  • PinShare backend connects to IPFS service                  ║
║  • Secrets managed with SOPS encryption                       ║
║                                                                ║
║  Populate secrets before first run:                           ║
║  $ ./k8s/secrets/populate-from-env.sh                         ║
║                                                                ║
║  View logs and service status in the Tilt UI!                 ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
""")
