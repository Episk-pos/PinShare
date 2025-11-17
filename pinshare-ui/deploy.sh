#!/bin/bash
set -e

# PinShare Frontend Deployment Script
# Deploys the frontend SPA to S3 and invalidates CloudFront cache

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_JSON="$SCRIPT_DIR/package.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to read config from package.json
get_config_value() {
    local key=$1
    node -p "require('$PACKAGE_JSON').config.$key" 2>/dev/null
}

# Load configuration from package.json
S3_BUCKET=$(get_config_value "s3Bucket")
CLOUDFRONT_ID=$(get_config_value "cloudfrontId")

# Validate configuration
if [ -z "$S3_BUCKET" ] || [ "$S3_BUCKET" = "undefined" ]; then
    log_error "S3 bucket not configured in package.json"
    log_error "Please add 'config.s3Bucket' to package.json"
    exit 1
fi

if [ -z "$CLOUDFRONT_ID" ] || [ "$CLOUDFRONT_ID" = "undefined" ]; then
    log_error "CloudFront distribution ID not configured in package.json"
    log_error "Please add 'config.cloudfrontId' to package.json"
    exit 1
fi

# Function to deploy to S3
deploy_to_s3() {
    log_info "Building frontend..."
    cd "$SCRIPT_DIR"
    npm run build

    log_info "Syncing to S3 bucket: $S3_BUCKET"
    aws s3 sync dist/ "s3://$S3_BUCKET" --delete

    log_info "S3 deployment complete!"
}

# Function to invalidate CloudFront cache
invalidate_cloudfront() {
    log_info "Invalidating CloudFront distribution: $CLOUDFRONT_ID"
    INVALIDATION_ID=$(aws cloudfront create-invalidation \
        --distribution-id "$CLOUDFRONT_ID" \
        --paths "/*" \
        --query 'Invalidation.Id' \
        --output text)

    log_info "CloudFront invalidation created: $INVALIDATION_ID"
    log_info "Cache invalidation in progress (this may take a few minutes)"
}

# Main deployment logic
main() {
    log_info "Starting PinShare frontend deployment..."
    log_info "S3 Bucket: $S3_BUCKET"
    log_info "CloudFront Distribution: $CLOUDFRONT_ID"

    # Check for AWS CLI
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLI is not installed. Please install it first."
        exit 1
    fi

    # Check for Node.js (required for reading package.json)
    if ! command -v node &> /dev/null; then
        log_error "Node.js is not installed. Please install it first."
        exit 1
    fi

    case "${1:-all}" in
        s3)
            deploy_to_s3
            ;;
        uncache|invalidate)
            invalidate_cloudfront
            ;;
        all|"")
            deploy_to_s3
            invalidate_cloudfront
            ;;
        *)
            echo "Usage: $0 [s3|uncache|all]"
            echo ""
            echo "Commands:"
            echo "  s3        - Build and deploy to S3 only"
            echo "  uncache   - Invalidate CloudFront cache only"
            echo "  all       - Deploy to S3 and invalidate cache (default)"
            exit 1
            ;;
    esac

    log_info "Deployment complete!"
    log_info "Frontend URL: https://$S3_BUCKET"
}

main "$@"
