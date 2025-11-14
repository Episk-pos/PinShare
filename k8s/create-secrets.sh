#!/bin/bash
# Script to create Kubernetes secrets for PinShare
# This script reads values from environment variables or prompts for input

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}PinShare Kubernetes Secrets Creator${NC}"
echo "======================================"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed or not in PATH${NC}"
    exit 1
fi

# Check if namespace exists, create if not
if ! kubectl get namespace pinshare &> /dev/null; then
    echo -e "${YELLOW}Creating pinshare namespace...${NC}"
    kubectl create namespace pinshare
    echo -e "${GREEN}✓ Namespace created${NC}"
else
    echo -e "${GREEN}✓ Namespace pinshare already exists${NC}"
fi

echo ""
echo "======================================"
echo "Backend Secrets"
echo "======================================"

# Prompt for backend secrets
read -p "VirusTotal API Token (optional, press Enter to skip): " VT_TOKEN
read -p "Google Client ID (optional, press Enter to skip): " GOOGLE_CLIENT_ID
read -p "Google Client Secret (optional, press Enter to skip): " GOOGLE_CLIENT_SECRET

# Generate or prompt for encryption key
if [ -z "$PS_ENCRYPTION_KEY" ]; then
    echo ""
    echo "Generating random 32-byte encryption key..."
    PS_ENCRYPTION_KEY=$(openssl rand -base64 32)
    echo -e "${YELLOW}Generated key: ${PS_ENCRYPTION_KEY}${NC}"
    echo -e "${YELLOW}Save this key securely if you need to recreate the secret later${NC}"
fi

# Create backend secret
echo ""
echo "Creating backend secret..."
kubectl create secret generic pinshare-backend-secret \
  --from-literal=VT_TOKEN="${VT_TOKEN}" \
  --from-literal=GOOGLE_CLIENT_ID="${GOOGLE_CLIENT_ID}" \
  --from-literal=GOOGLE_CLIENT_SECRET="${GOOGLE_CLIENT_SECRET}" \
  --from-literal=PS_ENCRYPTION_KEY="${PS_ENCRYPTION_KEY}" \
  -n pinshare \
  --dry-run=client -o yaml | kubectl apply -f -

echo -e "${GREEN}✓ Backend secret created/updated${NC}"

echo ""
echo "======================================"
echo "OAuth Broker Secrets"
echo "======================================"

read -p "Create OAuth Broker secrets? (y/n): " CREATE_OAUTH
if [ "$CREATE_OAUTH" = "y" ] || [ "$CREATE_OAUTH" = "Y" ]; then
    read -p "OAuth Broker Google Client ID: " OAUTH_GOOGLE_CLIENT_ID
    read -p "OAuth Broker Google Client Secret: " OAUTH_GOOGLE_CLIENT_SECRET
    read -p "OAuth Redirect URL (default: http://localhost:8888/callback): " REDIRECT_URL
    REDIRECT_URL=${REDIRECT_URL:-http://localhost:8888/callback}
    read -p "OAuth Broker Port (default: 8888): " OAUTH_PORT
    OAUTH_PORT=${OAUTH_PORT:-8888}

    echo ""
    echo "Creating OAuth broker secret..."
    kubectl create secret generic oauth-broker-secret \
      --from-literal=GOOGLE_CLIENT_ID="${OAUTH_GOOGLE_CLIENT_ID}" \
      --from-literal=GOOGLE_CLIENT_SECRET="${OAUTH_GOOGLE_CLIENT_SECRET}" \
      --from-literal=REDIRECT_URL="${REDIRECT_URL}" \
      --from-literal=PORT="${OAUTH_PORT}" \
      -n pinshare \
      --dry-run=client -o yaml | kubectl apply -f -

    echo -e "${GREEN}✓ OAuth broker secret created/updated${NC}"
else
    echo -e "${YELLOW}Skipping OAuth broker secrets${NC}"
fi

echo ""
echo "======================================"
echo -e "${GREEN}✓ All secrets created successfully!${NC}"
echo "======================================"
echo ""
echo "Next steps:"
echo "1. Review secrets: kubectl get secrets -n pinshare"
echo "2. Start Tilt: tilt up"
echo "3. Or deploy manually: kubectl apply -f k8s/base/"
echo ""
