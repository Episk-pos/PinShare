# PinShare Kubernetes Deployment

This directory contains Kubernetes manifests for deploying PinShare services and a Tiltfile for local development with Tilt.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start with Tilt](#quick-start-with-tilt)
- [Manual Kubernetes Deployment](#manual-kubernetes-deployment)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)
- [Comparison with Docker Compose](#comparison-with-docker-compose)

## Prerequisites

### For Tilt Development (Recommended)

- [Tilt](https://docs.tilt.dev/install.html) (v0.30+)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Local Kubernetes cluster:
  - [kind](https://kind.sigs.k8s.io/) (recommended)
  - [minikube](https://minikube.sigs.k8s.io/)
  - [Docker Desktop with Kubernetes](https://docs.docker.com/desktop/kubernetes/)

### For Manual Deployment

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Access to a Kubernetes cluster (local or remote)

## Quick Start with Tilt

### 1. Create a Local Kubernetes Cluster

**Using kind:**
```bash
# Create a kind cluster
kind create cluster --name pinshare

# Verify cluster is running
kubectl cluster-info --context kind-pinshare
```

**Using minikube:**
```bash
# Start minikube
minikube start --cpus=4 --memory=8192

# Verify cluster is running
kubectl cluster-info
```

### 2. Create Required Secrets

Before running Tilt, you need to create Kubernetes secrets for sensitive configuration:

```bash
# Create the pinshare namespace
kubectl create namespace pinshare

# Create backend secrets
kubectl create secret generic pinshare-backend-secret \
  --from-literal=VT_TOKEN="" \
  --from-literal=GOOGLE_CLIENT_ID="" \
  --from-literal=GOOGLE_CLIENT_SECRET="" \
  --from-literal=PS_ENCRYPTION_KEY="" \
  -n pinshare

# Create OAuth broker secrets
kubectl create secret generic oauth-broker-secret \
  --from-literal=GOOGLE_CLIENT_ID="" \
  --from-literal=GOOGLE_CLIENT_SECRET="" \
  --from-literal=REDIRECT_URL="http://localhost:8888/callback" \
  --from-literal=PORT="8888" \
  -n pinshare
```

**Note:** For production deployments, populate these with actual values:
- `VT_TOKEN`: Your VirusTotal API token (optional)
- `GOOGLE_CLIENT_ID`: Google OAuth client ID (optional, for Google Drive import)
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret (optional)
- `PS_ENCRYPTION_KEY`: 32-byte AES encryption key for OAuth tokens (required if using Google Drive)

To generate a secure encryption key:
```bash
openssl rand -base64 32
```

### 3. Configure Tilt for Your Cluster

Edit the `Tiltfile` in the project root and adjust the `allow_k8s_contexts()` line to match your cluster:

```python
# For kind
allow_k8s_contexts('kind-pinshare')

# For minikube
allow_k8s_contexts('minikube')

# For Docker Desktop
allow_k8s_contexts('docker-desktop')
```

### 4. Start Tilt

```bash
# From the project root directory
tilt up
```

This will:
- Build Docker images for all services
- Deploy Kubernetes manifests
- Set up port forwarding
- Open the Tilt UI in your browser

**Press `space` or navigate to http://localhost:10350** to open the Tilt UI.

### 5. Access Services

Once all services are running (green in Tilt UI):

- **UI:** http://localhost:5174
- **Backend API:** http://localhost:9090
- **IPFS API:** http://localhost:5001
- **IPFS Gateway:** http://localhost:8080
- **IPFS Web UI:** http://localhost:5001/webui
- **OAuth Broker:** http://localhost:8888

### 6. Development Workflow

**Hot Reload is Automatic:**

- **Frontend (UI):** Edit files in `pinshare-ui/src/` - Vite will hot reload automatically
- **Backend:** Edit Go files - Tilt will rebuild and restart the service
- **OAuth Broker:** Edit Go files - Tilt will rebuild and restart the service

**View Logs:**
- Click on any service in the Tilt UI to see real-time logs
- Or use kubectl: `kubectl logs -f deployment/pinshare-backend -n pinshare`

**Restart a Service:**
- In Tilt UI, click the service and press the restart button
- Or use kubectl: `kubectl rollout restart deployment/pinshare-backend -n pinshare`

### 7. Stop Tilt

```bash
# Press Ctrl+C in the terminal where Tilt is running, or:
tilt down
```

This will delete all Kubernetes resources but preserve persistent volumes.

## Manual Kubernetes Deployment

If you prefer to deploy without Tilt:

### 1. Build Docker Images

```bash
# Build backend
docker build -t pinshare-backend:latest .

# Build UI
docker build -t pinshare-ui:latest ./pinshare-ui

# Build OAuth broker
docker build -t oauth-broker:latest ./oauth-broker
```

### 2. Load Images into Cluster

**For kind:**
```bash
kind load docker-image pinshare-backend:latest --name pinshare
kind load docker-image pinshare-ui:latest --name pinshare
kind load docker-image oauth-broker:latest --name pinshare
```

**For minikube:**
```bash
# Use minikube's Docker daemon
eval $(minikube docker-env)
# Then rebuild images (they'll be built directly in minikube)
```

### 3. Create Secrets

Same as in [Step 2 of Quick Start](#2-create-required-secrets).

### 4. Deploy Kubernetes Manifests

```bash
# Deploy in order
kubectl apply -f k8s/base/namespace.yaml
kubectl apply -f k8s/dev/hostpath-pv.yaml
kubectl apply -f k8s/base/pinshare-backend/
kubectl apply -f k8s/base/pinshare-ui/
kubectl apply -f k8s/base/oauth-broker/
```

### 5. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n pinshare

# Check services
kubectl get svc -n pinshare

# View logs
kubectl logs -f deployment/pinshare-backend -n pinshare
```

### 6. Port Forward (if not using Ingress)

```bash
# Backend API
kubectl port-forward -n pinshare svc/pinshare-backend 9090:9090 &

# UI
kubectl port-forward -n pinshare svc/pinshare-ui 5174:5174 &

# IPFS API
kubectl port-forward -n pinshare svc/pinshare-backend 5001:5001 &

# IPFS Gateway
kubectl port-forward -n pinshare svc/pinshare-backend 8080:8080 &

# OAuth Broker
kubectl port-forward -n pinshare svc/oauth-broker 8888:8888 &
```

## Configuration

### Environment Variables (ConfigMap)

Edit `k8s/base/pinshare-backend/configmap.yaml` to modify:

- `PS_ORGNAME`: Organization name (default: "Cypherpunk")
- `PS_GROUPNAME`: Group name (default: "TestLab")
- `PS_LIBP2P_PORT`: libp2p port (default: "50001")
- Feature flags: `PS_FF_*` variables

### Secrets

Edit or recreate secrets to modify:

- VirusTotal API token
- Google OAuth credentials
- Encryption keys

To update a secret:
```bash
kubectl delete secret pinshare-backend-secret -n pinshare
kubectl create secret generic pinshare-backend-secret \
  --from-literal=VT_TOKEN="your-new-token" \
  ... \
  -n pinshare

# Restart deployment to pick up new secret
kubectl rollout restart deployment/pinshare-backend -n pinshare
```

### Persistent Storage

**Local Development (HostPath):**

The `k8s/dev/hostpath-pv.yaml` creates PersistentVolumes backed by host directories:
- `/tmp/pinshare-data`: Backend application data
- `/tmp/pinshare-ipfs`: IPFS data

**Production:**

Replace `hostPath` volumes with appropriate storage:
- Cloud provider storage classes (AWS EBS, GCP PD, Azure Disk)
- NFS or other network storage
- Rook/Ceph for self-hosted storage

Update `k8s/base/pinshare-backend/pvc.yaml` to use the appropriate `storageClassName`.

## Architecture

### Services

```
┌─────────────────┐
│   pinshare-ui   │  (Port 5174)
│   React + Vite  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ pinshare-backend│  (Port 9090 - API)
│  Go + IPFS      │  (Port 5001 - IPFS API)
│                 │  (Port 8080 - Gateway)
│                 │  (Port 4001 - Swarm)
│                 │  (Port 50001 - libp2p)
└────────┬────────┘
         │
         ├─ PVC: backend-data (10Gi)
         └─ PVC: ipfs-data (10Gi)

┌─────────────────┐
│  oauth-broker   │  (Port 8888)
│   Go OAuth2     │
└─────────────────┘
```

### Resource Requirements

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|-------------|-----------|----------------|--------------|
| Backend | 250m        | 1000m     | 512Mi          | 2Gi          |
| UI      | 100m        | 500m      | 128Mi          | 512Mi        |
| OAuth   | 50m         | 200m      | 64Mi           | 256Mi        |

Adjust these in the respective `deployment.yaml` files based on your needs.

### Init Containers

**pinshare-backend** has an init container that:
1. Initializes IPFS if not already done
2. Configures IPFS CORS headers for browser access
3. Sets up IPFS API access controls

### Health Checks

**Backend:**
- **Liveness:** Checks IPFS daemon is responsive
- **Readiness:** Checks API endpoint `/api/v1/files` returns successfully

**UI & OAuth Broker:**
- **Liveness & Readiness:** HTTP GET to root path

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n pinshare

# Describe pod for events
kubectl describe pod <pod-name> -n pinshare

# View logs
kubectl logs <pod-name> -n pinshare

# Check for previous container logs (if pod restarted)
kubectl logs <pod-name> -n pinshare --previous
```

### Backend Failing Health Checks

The backend has a 90-second initial delay for IPFS initialization. If it's still failing:

```bash
# Check backend logs
kubectl logs -n pinshare deployment/pinshare-backend

# Check IPFS initialization
kubectl exec -it -n pinshare deployment/pinshare-backend -- ipfs id
```

### PersistentVolume Issues

```bash
# Check PV and PVC status
kubectl get pv,pvc -n pinshare

# If using hostPath, ensure directories exist on the node
ls -la /tmp/pinshare-data /tmp/pinshare-ipfs
```

### Image Pull Errors

For local development, ensure images are loaded into your cluster:

```bash
# For kind
kind load docker-image pinshare-backend:latest --name pinshare

# For minikube
eval $(minikube docker-env)
docker images | grep pinshare  # Verify images exist
```

### Secret Missing

```bash
# Verify secrets exist
kubectl get secrets -n pinshare

# View secret contents (base64 encoded)
kubectl get secret pinshare-backend-secret -n pinshare -o yaml
```

### Port Forwarding Not Working

```bash
# Kill existing port-forwards
pkill -f "kubectl port-forward"

# Re-establish port-forward
kubectl port-forward -n pinshare svc/pinshare-ui 5174:5174
```

### Tilt Issues

```bash
# Check Tilt version
tilt version

# Verify Kubernetes context
kubectl config current-context

# Check if context is allowed in Tiltfile
# Edit Tiltfile and adjust allow_k8s_contexts()

# Clear Tilt cache
rm -rf ~/.tilt-dev/

# Restart Tilt with verbose logging
tilt up --stream
```

## Comparison with Docker Compose

| Feature | Docker Compose | Tilt + Kubernetes |
|---------|----------------|-------------------|
| **Startup Time** | Fast (~30s) | Medium (~2min first time, ~30s subsequent) |
| **Hot Reload** | Limited (UI only) | Full (UI + Backend + OAuth) |
| **Resource Isolation** | Container-level | Pod + Namespace-level |
| **Service Discovery** | DNS (service names) | Kubernetes DNS (service.namespace.svc) |
| **Log Aggregation** | `docker-compose logs` | Tilt UI or `kubectl logs` |
| **Port Management** | docker-compose.yml | Tiltfile + Services |
| **Persistent Storage** | Volume mounts | PersistentVolumes + Claims |
| **Production Parity** | Low | High |
| **Debugging** | Docker logs | Tilt UI + kubectl + k9s |
| **Multi-developer** | Potential port conflicts | Namespace isolation |
| **Learning Curve** | Low | Medium-High |

### When to Use Each

**Use Docker Compose when:**
- Quick one-off testing
- Simplicity is paramount
- No Kubernetes knowledge required
- Working on a single service

**Use Tilt + Kubernetes when:**
- Developing cloud-native features
- Testing Kubernetes-specific behavior
- Need better resource management
- Preparing for production deployment
- Working with multiple developers on shared cluster
- Want superior development experience with Tilt UI

## Additional Resources

- [Tilt Documentation](https://docs.tilt.dev/)
- [Kubernetes Documentation](https://kubernetes.io/docs/home/)
- [kind Quick Start](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)

## License

Same as parent project.
