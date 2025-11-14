#!/bin/bash
# Script to populate SOPS-encrypted secrets from .env file
# Usage: ./populate-from-env.sh

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Populating SOPS-encrypted secrets from .env file${NC}"
echo "=============================================="

# Check if .env exists
if [ ! -f ".env" ]; then
    echo -e "${RED}Error: .env file not found in project root${NC}"
    exit 1
fi

# Check if sops is installed
if ! command -v sops &> /dev/null; then
    echo -e "${RED}Error: sops is not installed${NC}"
    exit 1
fi

# Source .env file (without exposing to logs)
set -a
source .env 2>/dev/null || true
set +a

echo ""
echo -e "${YELLOW}Decrypting and updating backend secret...${NC}"

# Create temporary file for backend secret
BACKEND_TEMP=$(mktemp)
trap "rm -f $BACKEND_TEMP" EXIT

# Decrypt current secret
sops --decrypt k8s/secrets/pinshare-backend-secret.yaml > "$BACKEND_TEMP"

# Update values using yq or sed (yq is cleaner but sed is more portable)
if command -v yq &> /dev/null; then
    # Use yq if available
    yq eval ".stringData.VT_TOKEN = \"${VT_TOKEN:-}\"" -i "$BACKEND_TEMP"
    yq eval ".stringData.GOOGLE_CLIENT_ID = \"${GOOGLE_CLIENT_ID:-}\"" -i "$BACKEND_TEMP"
    yq eval ".stringData.GOOGLE_CLIENT_SECRET = \"${GOOGLE_CLIENT_SECRET:-}\"" -i "$BACKEND_TEMP"
    yq eval ".stringData.PS_ENCRYPTION_KEY = \"${PS_ENCRYPTION_KEY:-}\"" -i "$BACKEND_TEMP"
else
    # Fallback to sed (less reliable for complex values)
    sed -i "s|VT_TOKEN: \".*\"|VT_TOKEN: \"${VT_TOKEN:-}\"|" "$BACKEND_TEMP"
    sed -i "s|GOOGLE_CLIENT_ID: \".*\"|GOOGLE_CLIENT_ID: \"${GOOGLE_CLIENT_ID:-}\"|" "$BACKEND_TEMP"
    sed -i "s|GOOGLE_CLIENT_SECRET: \".*\"|GOOGLE_CLIENT_SECRET: \"${GOOGLE_CLIENT_SECRET:-}\"|" "$BACKEND_TEMP"
    sed -i "s|PS_ENCRYPTION_KEY: \".*\"|PS_ENCRYPTION_KEY: \"${PS_ENCRYPTION_KEY:-}\"|" "$BACKEND_TEMP"
fi

# Re-encrypt
sops --encrypt "$BACKEND_TEMP" > k8s/secrets/pinshare-backend-secret.yaml

echo -e "${GREEN}✓ Backend secret updated${NC}"

echo ""
echo -e "${YELLOW}Decrypting and updating OAuth broker secret...${NC}"

# Create temporary file for OAuth broker secret
OAUTH_TEMP=$(mktemp)
trap "rm -f $BACKEND_TEMP $OAUTH_TEMP" EXIT

# Decrypt current secret
sops --decrypt k8s/secrets/oauth-broker-secret.yaml > "$OAUTH_TEMP"

# Update values
if command -v yq &> /dev/null; then
    yq eval ".stringData.GOOGLE_CLIENT_ID = \"${GOOGLE_CLIENT_ID:-}\"" -i "$OAUTH_TEMP"
    yq eval ".stringData.GOOGLE_CLIENT_SECRET = \"${GOOGLE_CLIENT_SECRET:-}\"" -i "$OAUTH_TEMP"
else
    sed -i "s|GOOGLE_CLIENT_ID: \".*\"|GOOGLE_CLIENT_ID: \"${GOOGLE_CLIENT_ID:-}\"|" "$OAUTH_TEMP"
    sed -i "s|GOOGLE_CLIENT_SECRET: \".*\"|GOOGLE_CLIENT_SECRET: \"${GOOGLE_CLIENT_SECRET:-}\"|" "$OAUTH_TEMP"
fi

# Re-encrypt
sops --encrypt "$OAUTH_TEMP" > k8s/secrets/oauth-broker-secret.yaml

echo -e "${GREEN}✓ OAuth broker secret updated${NC}"

echo ""
echo -e "${GREEN}=============================================="
echo -e "Secrets populated successfully!"
echo -e "=============================================="${NC}
echo ""
echo "To apply to Kubernetes:"
echo "  sops --decrypt k8s/secrets/pinshare-backend-secret.yaml | kubectl apply -f -"
echo "  sops --decrypt k8s/secrets/oauth-broker-secret.yaml | kubectl apply -f -"
echo ""
