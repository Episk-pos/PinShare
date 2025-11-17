# PinShare Frontend Deployment

This document describes how to deploy the PinShare frontend to AWS S3 and CloudFront.

## Prerequisites

1. **AWS CLI** - Installed and configured with appropriate credentials
   ```bash
   aws configure
   ```

2. **Node.js & npm** - For building the frontend and running deployment scripts

3. **Infrastructure** - S3 bucket and CloudFront distribution must be provisioned (via Terraform in the infra repo)

## Configuration

Deployment settings are stored in `package.json` under the `config` section:

```json
{
  "config": {
    "cloudfrontId": "E1N441DRNZRMQ4",
    "s3Bucket": "share.episkopos.community"
  }
}
```

### Updating Configuration

If the CloudFront distribution ID or S3 bucket changes, update these values in `package.json`.

To get the current CloudFront distribution ID from Terraform:
```bash
cd ../infra/terraform
terraform output -raw cloudfront_distribution_id
```

## Deployment Methods

### Method 1: npm Scripts (Recommended)

```bash
# Full deployment (build + S3 sync + CloudFront invalidation)
npm run deploy

# Deploy to S3 only
npm run deploy:s3

# Invalidate CloudFront cache only
npm run deploy:uncache
```

### Method 2: Shell Script

```bash
# Full deployment
./deploy.sh

# Deploy to S3 only
./deploy.sh s3

# Invalidate CloudFront cache only
./deploy.sh uncache
```

## What Happens During Deployment

1. **Build** (`vite build`)
   - Compiles React app
   - Optimizes assets
   - Outputs to `dist/` directory

2. **S3 Sync** (`aws s3 sync`)
   - Uploads `dist/` contents to `s3://share.episkopos.community`
   - `--delete` flag removes old files

3. **CloudFront Invalidation** (`aws cloudfront create-invalidation`)
   - Invalidates all paths (`/*`)
   - Forces CloudFront to fetch fresh content from S3
   - Takes 2-5 minutes to complete

## Infrastructure Details

- **S3 Bucket**: `share.episkopos.community`
- **CloudFront Distribution**: Retrieved from Terraform outputs
- **Domains**:
  - `https://share.episkopos.community` (canonical)
  - `https://pinshare.episkopos.community`
  - `https://ps.episkopos.community`
  - `https://archive.episkopos.community`

## Troubleshooting

### "CloudFront distribution ID not configured in package.json"

Make sure the `config` section in `package.json` has the correct `cloudfrontId`:

```json
{
  "config": {
    "cloudfrontId": "E1N441DRNZRMQ4"
  }
}
```

Get the distribution ID from Terraform if needed:
```bash
cd ../infra/terraform
terraform output -raw cloudfront_distribution_id
```

### "The specified bucket does not exist"

Verify the S3 bucket name in `package.json` matches the actual bucket:

```json
{
  "config": {
    "s3Bucket": "share.episkopos.community"
  }
}
```

### AWS Credentials Issues

Ensure AWS CLI is configured:

```bash
aws configure
aws sts get-caller-identity  # Verify credentials work
```

### CloudFront Invalidation Stuck

CloudFront invalidations can take 5-15 minutes. Check status:

```bash
# List invalidations (use your distribution ID from package.json)
aws cloudfront list-invalidations --distribution-id E1N441DRNZRMQ4
```

## Manual Deployment

If the scripts don't work, you can deploy manually:

```bash
# 1. Build
npm run build

# 2. Sync to S3 (use bucket from package.json config)
aws s3 sync dist/ s3://share.episkopos.community --delete

# 3. Create invalidation (use distribution ID from package.json config)
aws cloudfront create-invalidation --distribution-id E1N441DRNZRMQ4 --paths "/*"
```

## CI/CD Integration

These scripts can be integrated into GitHub Actions or other CI/CD systems:

```yaml
# Example GitHub Actions workflow
- name: Deploy to Production
  run: |
    cd pinshare-ui
    npm ci
    npm run deploy
  env:
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    AWS_DEFAULT_REGION: us-east-1
```
