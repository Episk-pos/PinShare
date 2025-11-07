# PinShare Cloud Deployment Guide

## Metadata-Only Archive Node Configuration

This guide explains how to deploy PinShare in a cloud environment as a **metadata-only archive node** that syncs file metadata via P2P gossip but doesn't automatically download files. Files are fetched from IPFS only when users request them through the web interface.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Cloud Deployment                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐         ┌──────────────┐            │
│  │   Frontend   │─────────│   Backend    │            │
│  │  (React App) │  HTTP   │ (PinShare Go)│            │
│  │  Port 3000   │         │  Port 9090   │            │
│  └──────────────┘         └──────┬───────┘            │
│                                   │                     │
│                                   │ P2P Gossip          │
│                                   │ (Metadata Sync)     │
│                                   │                     │
│                           ┌───────▼──────┐             │
│                           │  IPFS Node   │             │
│                           │ (On-Demand)  │             │
│                           └──────────────┘             │
│                                                         │
└─────────────────────────────────────────────────────────┘
                        │
                        │ P2P Network
                        ▼
            ┌──────────────────────┐
            │  Other PinShare Nodes│
            │  (Full or Archive)   │
            └──────────────────────┘
```

## Key Features

1. **Metadata Synchronization**: Receives and stores file metadata via P2P gossip
2. **No Auto-Download**: Files are NOT automatically downloaded when metadata is received
3. **On-Demand Retrieval**: Files are fetched from IPFS only when users click Download or Preview
4. **Web Interface**: React frontend for browsing, searching, and downloading files
5. **Bandwidth Efficient**: Ideal for cloud deployments where you want to minimize storage/bandwidth costs

## Environment Variables

### Required Configuration

```bash
# Archive Mode - CRITICAL: Set this to true for metadata-only mode
PS_FF_ARCHIVE_NODE=true

# Organization and Group (must match your P2P network)
PS_ORGNAME=CypherPunk
PS_GROUPNAME=LabRat250805

# libp2p Port
PS_LIBP2P_PORT=50001

# Feature Flags
PS_FF_SKIP_VT=true                      # Skip VirusTotal checks (no auto-download anyway)
PS_FF_IGNORE_UPLOADS_IN_METADATA=true   # Don't watch upload folder
PS_FF_MOVE_UPLOAD=false                 # Don't move files after processing
```

### Optional Configuration

```bash
# Security - Optional, not required for metadata-only mode
VT_TOKEN=your_virustotal_api_token      # Only if you want security checks on user downloads
```

## Deployment Options

### Option 1: Docker Deployment

1. **Create a Dockerfile** (if not already present):

```dockerfile
# Multi-stage build
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o pinshare main.go

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates ipfs

WORKDIR /app
COPY --from=builder /app/pinshare .

# Initialize IPFS
RUN ipfs init

# Frontend build (optional - serve separately)
# COPY frontend/dist ./frontend/dist

EXPOSE 9090 50001

CMD ["./pinshare", "start"]
```

2. **Create docker-compose.yml**:

```yaml
version: '3.8'

services:
  pinshare-backend:
    build: .
    container_name: pinshare-archive
    environment:
      - PS_FF_ARCHIVE_NODE=true
      - PS_ORGNAME=CypherPunk
      - PS_GROUPNAME=LabRat250805
      - PS_LIBP2P_PORT=50001
      - PS_FF_SKIP_VT=true
      - PS_FF_IGNORE_UPLOADS_IN_METADATA=true
      - PS_FF_MOVE_UPLOAD=false
    ports:
      - "9090:9090"   # API
      - "50001:50001" # libp2p
    volumes:
      - ./cache:/app/cache
      - ./metadata.json:/app/metadata.json
      - ./identity.key:/app/identity.key
    restart: unless-stopped

  pinshare-frontend:
    image: node:18-alpine
    container_name: pinshare-frontend
    working_dir: /app
    command: sh -c "npm install && npm run dev"
    environment:
      - VITE_API_BASE=http://pinshare-backend:9090
    ports:
      - "3000:3000"
    volumes:
      - ./frontend:/app
    depends_on:
      - pinshare-backend
    restart: unless-stopped
```

3. **Deploy**:

```bash
docker-compose up -d
```

### Option 2: Cloud Platform Deployment (AWS, GCP, Azure)

#### AWS ECS/Fargate

1. **Create Task Definition**:

```json
{
  "family": "pinshare-archive",
  "containerDefinitions": [
    {
      "name": "pinshare",
      "image": "your-ecr-repo/pinshare:latest",
      "portMappings": [
        { "containerPort": 9090, "protocol": "tcp" },
        { "containerPort": 50001, "protocol": "tcp" }
      ],
      "environment": [
        { "name": "PS_FF_ARCHIVE_NODE", "value": "true" },
        { "name": "PS_ORGNAME", "value": "CypherPunk" },
        { "name": "PS_GROUPNAME", "value": "LabRat250805" },
        { "name": "PS_LIBP2P_PORT", "value": "50001" }
      ],
      "mountPoints": [
        {
          "sourceVolume": "metadata-storage",
          "containerPath": "/app/metadata.json"
        }
      ]
    }
  ],
  "volumes": [
    {
      "name": "metadata-storage",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-xxxxx"
      }
    }
  ]
}
```

2. **Set up Load Balancer**:
   - Application Load Balancer for HTTP/HTTPS (port 9090)
   - Network Load Balancer for P2P traffic (port 50001)

#### Google Cloud Run

```bash
# Build and push image
gcloud builds submit --tag gcr.io/PROJECT_ID/pinshare

# Deploy with environment variables
gcloud run deploy pinshare-archive \
  --image gcr.io/PROJECT_ID/pinshare \
  --platform managed \
  --port 9090 \
  --set-env-vars="PS_FF_ARCHIVE_NODE=true,PS_ORGNAME=CypherPunk,PS_GROUPNAME=LabRat250805" \
  --allow-unauthenticated
```

**Note**: Cloud Run has limitations with P2P networking. Consider GCE or GKE for full P2P support.

### Option 3: Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pinshare-archive
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pinshare
  template:
    metadata:
      labels:
        app: pinshare
    spec:
      containers:
      - name: pinshare
        image: your-registry/pinshare:latest
        ports:
        - containerPort: 9090
          name: api
        - containerPort: 50001
          name: libp2p
        env:
        - name: PS_FF_ARCHIVE_NODE
          value: "true"
        - name: PS_ORGNAME
          value: "CypherPunk"
        - name: PS_GROUPNAME
          value: "LabRat250805"
        - name: PS_LIBP2P_PORT
          value: "50001"
        volumeMounts:
        - name: metadata
          mountPath: /app/metadata.json
        - name: cache
          mountPath: /app/cache
      volumes:
      - name: metadata
        persistentVolumeClaim:
          claimName: pinshare-metadata
      - name: cache
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: pinshare-service
spec:
  type: LoadBalancer
  ports:
  - port: 9090
    name: api
  - port: 50001
    name: libp2p
  selector:
    app: pinshare
```

## Frontend Deployment

### Option 1: Static Hosting (Recommended)

Build and deploy the frontend separately to a CDN:

```bash
cd frontend
npm install
npm run build

# Deploy to S3 + CloudFront, Netlify, Vercel, etc.
# Example for Netlify:
netlify deploy --prod --dir=dist
```

Update `vite.config.js` to point to your backend:

```javascript
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'https://your-backend-url.com',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '')
      }
    }
  }
})
```

### Option 2: Serve from Backend

Serve the built frontend from the Go backend using a static file server.

## Networking Considerations

### Firewall Rules

**Inbound**:
- Port 9090: HTTP API (restrict to your frontend domain or make public)
- Port 50001: libp2p P2P connections (must be publicly accessible)

**Outbound**:
- Allow all (for P2P peer discovery and IPFS gateway access)

### NAT Traversal

PinShare uses libp2p's built-in NAT traversal:
- UPnP/NAT-PMP
- Hole punching
- Circuit relay

For cloud deployments, ensure your instance has a public IP or use a relay node.

### Bootstrap Peers

To connect to the PinShare network, you'll need bootstrap peer addresses. Add them via the API:

```bash
curl -X POST http://localhost:9090/p2p/peers \
  -H "Content-Type: application/json" \
  -d '{"multiaddr": "/ip4/PEER_IP/tcp/50001/p2p/PEER_ID"}'
```

## Monitoring and Observability

### Prometheus Metrics

PinShare exposes metrics at `/metrics`:

```bash
curl http://localhost:9090/metrics
```

### Logging

Monitor logs for:
- Metadata sync events: `Applied and saved update from peer`
- Archive mode confirmations: `Archive mode enabled - skipping automatic download`
- API requests: `Fetching file X from IPFS`

### Health Checks

```bash
# Check P2P status
curl http://localhost:9090/p2p/status

# Check file count
curl http://localhost:9090/files | jq 'length'
```

## Storage Considerations

### Metadata Storage

- **Size**: Metadata is small (~1-2KB per file)
- **Growth**: Depends on network activity
- **Backup**: Regular backups of `metadata.json` recommended

### Cache Storage

- **Size**: Variable (files are cached after download)
- **Cleanup**: Implement cache cleanup policy based on disk space
- **Mount**: Use persistent volume for cache directory

## Security Best Practices

1. **TLS/HTTPS**: Use a reverse proxy (nginx, Caddy) for HTTPS
2. **Rate Limiting**: Implement rate limiting on API and download endpoints
3. **CORS**: Configure CORS properly for your frontend domain
4. **Content Validation**: Enable security scanning if downloading user content
5. **Access Control**: Consider adding authentication for the API

## Cost Optimization

### For Cloud Deployment

1. **Use spot/preemptible instances** (metadata is persistent, can handle restarts)
2. **Auto-scaling**: Not typically needed for archive nodes
3. **Bandwidth**: Downloads are on-demand, reducing egress costs
4. **Storage**: Use cheap object storage (S3, GCS) for cache with lifecycle policies

### Estimated Costs (AWS Example)

- **t3.small EC2**: ~$15/month
- **EBS Storage (50GB)**: ~$5/month
- **Data Transfer**: Variable (only when users download)
- **Total**: ~$20-50/month depending on usage

## Troubleshooting

### Node Not Connecting to Peers

1. Check firewall allows port 50001
2. Verify bootstrap peers are reachable
3. Check logs for connection attempts

### Files Not Downloading

1. Ensure IPFS daemon is running
2. Check IPFS has access to IPFS network
3. Verify CID exists in IPFS network

### Frontend Can't Reach Backend

1. Check CORS configuration
2. Verify proxy settings in `vite.config.js`
3. Check API is accessible from frontend domain

## Example: Complete AWS Deployment

```bash
# 1. Launch EC2 instance
aws ec2 run-instances --image-id ami-xxxxx --instance-type t3.small

# 2. SSH into instance
ssh ec2-user@your-instance-ip

# 3. Install dependencies
sudo yum install -y docker git golang
sudo systemctl start docker

# 4. Clone and build
git clone https://github.com/yourorg/pinshare.git
cd pinshare
go build -o pinshare main.go

# 5. Set up IPFS
wget https://dist.ipfs.io/go-ipfs/latest/go-ipfs_linux-amd64.tar.gz
tar -xvzf go-ipfs_linux-amd64.tar.gz
cd go-ipfs
sudo ./install.sh
ipfs init

# 6. Create systemd service
sudo tee /etc/systemd/system/pinshare.service <<EOF
[Unit]
Description=PinShare Archive Node
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/home/ec2-user/pinshare
Environment="PS_FF_ARCHIVE_NODE=true"
Environment="PS_ORGNAME=CypherPunk"
Environment="PS_GROUPNAME=LabRat250805"
ExecStart=/home/ec2-user/pinshare/pinshare start
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 7. Start services
sudo systemctl daemon-reload
sudo systemctl enable pinshare
sudo systemctl start pinshare

# 8. Check status
sudo systemctl status pinshare
curl http://localhost:9090/p2p/status
```

## Maintenance

### Updating

```bash
# Pull latest code
git pull origin main

# Rebuild
go build -o pinshare main.go

# Restart service
sudo systemctl restart pinshare
```

### Backup

```bash
# Backup metadata
tar -czf pinshare-backup-$(date +%Y%m%d).tar.gz metadata.json identity.key

# Upload to S3
aws s3 cp pinshare-backup-*.tar.gz s3://your-backup-bucket/
```

## Support and Resources

- **Documentation**: https://github.com/yourorg/pinshare/docs
- **Issues**: https://github.com/yourorg/pinshare/issues
- **Community**: Join our Discord/Slack

## License

See main PinShare LICENSE file.
