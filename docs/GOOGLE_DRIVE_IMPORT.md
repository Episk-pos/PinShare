# Google Drive Import Feature

## Overview

The Google Drive Import feature allows PinShare users to import files from their Google Drive into the decentralized PinShare network. This enables users to:

- Liberate data from centralized cloud storage
- Share files via P2P/IPFS without relying on Google's infrastructure
- Maintain a decentralized backup of important files

## Architecture

### Components

1. **OAuth 2.0 Authentication** - Secure authentication with Google using PKCE flow
2. **Google Drive API Integration** - Browse and download files from Drive
3. **Background Job Queue** - Process imports asynchronously with worker pool
4. **Database Layer** - Track users, import jobs, and file status (SQLite)
5. **Import Pipeline** - Integrate with existing upload pipeline (validation → hashing → scanning → IPFS → P2P)

### Database Schema

```
users
├── id (auto-increment)
├── google_id (unique)
├── email
├── encrypted_access_token
├── encrypted_refresh_token
├── token_expiry
├── created_at
└── updated_at

import_jobs
├── id (UUID)
├── user_id (FK → users)
├── status (pending|running|completed|failed|cancelled)
├── total_files
├── completed_files
├── failed_files
├── total_bytes
├── transferred_bytes
├── started_at
├── completed_at
└── options (JSON)

import_files
├── id (auto-increment)
├── job_id (FK → import_jobs)
├── drive_file_id
├── file_name
├── file_size
├── status (pending|downloading|hashing|scanning|uploading|completed|failed)
├── progress (0-100)
├── sha256_hash
├── ipfs_cid
├── error_message
├── retry_count
├── started_at
└── completed_at
```

## Setup Instructions

### 1. Prerequisites

- IPFS daemon running (`ipfs daemon`)
- Security scanning tool (P2P-Sec, VirusTotal API, ClamAV, or Chromium for web scraping)
- Google Cloud Platform account

### 2. Google Cloud Setup

1. **Create a Google Cloud Project**
   - Go to https://console.cloud.google.com
   - Create a new project or select an existing one

2. **Enable Google Drive API**
   - Navigate to "APIs & Services" → "Library"
   - Search for "Google Drive API"
   - Click "Enable"

3. **Create OAuth 2.0 Credentials**
   - Go to "APIs & Services" → "Credentials"
   - Click "Create Credentials" → "OAuth client ID"
   - Choose "Web application"
   - Set authorized redirect URI: `http://localhost:9090/api/google-drive/callback`
   - Copy the Client ID and Client Secret

### 3. PinShare Configuration

1. **Copy environment template**
   ```bash
   cp .env.example .env
   ```

2. **Generate encryption key** (32 bytes for AES-256)
   ```bash
   openssl rand -base64 32
   ```

3. **Configure .env file**
   ```env
   # Google OAuth Credentials
   GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-client-secret
   GOOGLE_REDIRECT_URL=http://localhost:9090/api/google-drive/callback

   # Security
   PS_ENCRYPTION_KEY=your-32-byte-encryption-key

   # Import Settings
   PS_DATABASE_FILE=pinshare.db
   PS_TEMP_DOWNLOAD_DIR=./tmp/gdrive
   PS_MAX_CONCURRENT_JOBS=3
   PS_MAX_FILE_SIZE_MB=1024
   ```

4. **Start PinShare**
   ```bash
   go run main.go
   ```

### 4. Verify Setup

Check the startup logs for:
```
[INFO] Database initialized successfully
[INFO] Job queue started with 3 workers
[INFO] Google Drive import enabled
[INFO] Registering Google Drive import API routes
```

If you see:
```
[WARNING] Google Drive import disabled (OAuth credentials not configured)
```

Then check that all three OAuth environment variables are set correctly.

## API Endpoints

### Authentication

#### 1. Initiate OAuth Flow
```http
POST /api/google-drive/authorize
```

**Response:**
```json
{
  "authorizationUrl": "https://accounts.google.com/o/oauth2/v2/auth?...",
  "state": "random-state-value"
}
```

**Usage:** Open the `authorizationUrl` in a browser to authenticate with Google.

#### 2. OAuth Callback
```http
POST /api/google-drive/callback?code={code}&state={state}
```

**Response:**
```json
{
  "success": true,
  "user": {
    "googleId": "123456789",
    "email": "user@example.com"
  }
}
```

**Usage:** This endpoint is called automatically by Google after authentication.

#### 3. Check Auth Status
```http
GET /api/google-drive/auth-status?google_id={googleId}
```

**Response:**
```json
{
  "authenticated": true,
  "email": "user@example.com",
  "tokenExpired": false
}
```

#### 4. Revoke Access
```http
DELETE /api/google-drive/revoke?google_id={googleId}
```

**Response:**
```json
{
  "success": true
}
```

### File Operations

#### 5. List Drive Folders/Files
```http
GET /api/google-drive/folders?google_id={googleId}&path={folderId}
```

**Parameters:**
- `google_id` (required): User's Google ID
- `path` (optional): Folder ID to browse (empty = root)

**Response:**
```json
[
  {
    "id": "1abc...",
    "name": "My Folder",
    "mimeType": "application/vnd.google-apps.folder",
    "size": 0,
    "modifiedTime": "2025-01-01T12:00:00Z",
    "parents": ["root"],
    "isFolder": true
  },
  {
    "id": "2def...",
    "name": "document.pdf",
    "mimeType": "application/pdf",
    "size": 1024000,
    "modifiedTime": "2025-01-01T12:00:00Z",
    "parents": ["1abc..."],
    "isFolder": false
  }
]
```

#### 6. Preview Import
```http
POST /api/google-drive/preview-import
```

**Request:**
```json
{
  "googleId": "123456789",
  "fileIds": ["2def..."],
  "folderIds": ["1abc..."],
  "recursive": true
}
```

**Response:**
```json
{
  "files": [
    {
      "id": "2def...",
      "name": "document.pdf",
      "size": 1024000,
      "modifiedTime": "2025-01-01T12:00:00Z"
    }
  ],
  "totalSize": 1024000,
  "totalCount": 1
}
```

#### 7. Start Import Job
```http
POST /api/google-drive/import
```

**Request:**
```json
{
  "googleId": "123456789",
  "fileIds": ["2def..."],
  "folderIds": ["1abc..."],
  "recursive": true,
  "preserveHierarchy": false,
  "skipDuplicates": true
}
```

**Options:**
- `recursive`: Include subfolders
- `preserveHierarchy`: Keep folder structure (future feature)
- `skipDuplicates`: Skip files that already exist in PinShare (by SHA256)

**Response:**
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending"
}
```

#### 8. Get Import Job Status
```http
GET /api/google-drive/import/{jobId}/status
```

**Response:**
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "progress": {
    "totalFiles": 10,
    "completedFiles": 3,
    "failedFiles": 1,
    "percentComplete": 30,
    "bytesTransferred": 3072000,
    "totalBytes": 10240000
  },
  "files": [
    {
      "driveId": "2def...",
      "fileName": "document.pdf",
      "status": "completed",
      "progress": 100,
      "error": "",
      "hash": "abc123...",
      "cid": "QmXyz..."
    },
    {
      "driveId": "3ghi...",
      "fileName": "image.jpg",
      "status": "downloading",
      "progress": 50,
      "error": "",
      "hash": "",
      "cid": ""
    }
  ],
  "startedAt": "2025-01-01T12:00:00Z",
  "completedAt": null
}
```

#### 9. Get Import History
```http
GET /api/google-drive/import/history?google_id={googleId}&limit=20&offset=0
```

**Response:**
```json
[
  {
    "jobId": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "totalFiles": 10,
    "completedFiles": 9,
    "failedFiles": 1,
    "totalBytes": 10240000,
    "transferredBytes": 9216000,
    "startedAt": "2025-01-01T12:00:00Z",
    "completedAt": "2025-01-01T12:15:00Z"
  }
]
```

## Import Pipeline

Each file goes through the following stages:

1. **Pending** - File queued for import
2. **Downloading** - Downloading from Google Drive
3. **Hashing** - Computing SHA256 hash
4. **Scanning** - Security scan (VirusTotal/ClamAV/P2P-Sec)
5. **Uploading** - Adding to IPFS
6. **Completed** - Metadata stored and P2P broadcast

If any stage fails, the file is marked as **Failed** with an error message.

## Security Features

### Token Encryption

- Access and refresh tokens are encrypted using AES-256-GCM
- Encryption key must be 32 bytes (set via `PS_ENCRYPTION_KEY`)
- Tokens are never stored in plaintext

### OAuth PKCE

- Implements Proof Key for Code Exchange (PKCE) for additional security
- Uses SHA256 challenge method
- State parameter prevents CSRF attacks

### File Security

All imported files go through the same security pipeline as uploaded files:

1. File type validation (against allowlist)
2. Malware scanning (VirusTotal/ClamAV/P2P-Sec)
3. Files that fail security checks are rejected

### Rate Limiting

Respects Google Drive API quotas:
- Concurrent downloads limited by `PS_MAX_CONCURRENT_JOBS`
- Token refresh handled automatically

## Troubleshooting

### Import not starting

**Problem:** Import job stays in "pending" status

**Solutions:**
- Check job queue is running: Look for `[INFO] Job queue started with N workers`
- Check worker logs for errors
- Verify IPFS daemon is running

### Authentication fails

**Problem:** OAuth callback returns an error

**Solutions:**
- Verify redirect URI matches exactly in Google Cloud Console
- Check client ID and secret are correct
- Ensure Google Drive API is enabled in your project

### Files fail security scan

**Problem:** Files marked as "failed" with "Security scan detected malware"

**Solutions:**
- Verify security scanning is working: Check `SecurityCapability` in logs
- Test security scanner with known clean files
- Check VirusTotal API key if using VT (level 2)

### Token expired

**Problem:** `"tokenExpired": true` in auth-status

**Solutions:**
- Tokens are refreshed automatically when needed
- If refresh fails, re-authenticate via `/api/google-drive/authorize`

## Performance Tuning

### Concurrent Jobs

Adjust based on your system resources:

```env
# Lower for low-memory systems
PS_MAX_CONCURRENT_JOBS=1

# Higher for powerful systems
PS_MAX_CONCURRENT_JOBS=10
```

### File Size Limits

Prevent downloading extremely large files:

```env
# 1GB limit
PS_MAX_FILE_SIZE_MB=1024

# 5GB limit
PS_MAX_FILE_SIZE_MB=5120

# No limit
PS_MAX_FILE_SIZE_MB=0
```

### Temp Directory

Use a fast disk for temporary downloads:

```env
# Use SSD for faster processing
PS_TEMP_DOWNLOAD_DIR=/mnt/ssd/gdrive-temp

# Use RAM disk for maximum speed (requires enough RAM)
PS_TEMP_DOWNLOAD_DIR=/dev/shm/gdrive-temp
```

## Future Enhancements

### Phase 2: Enhanced Monitoring (Planned)
- Real-time progress updates via WebSocket
- Bandwidth/speed metrics
- Detailed error tracking
- Prometheus metrics

### Phase 3: Continuous Sync (Planned)
- Auto-detect Drive changes
- Scheduled sync intervals
- Conflict resolution
- Multi-folder sync

### Phase 4: Additional Providers (Planned)
- Dropbox integration
- OneDrive integration
- S3 bucket import
- Bidirectional sync (PinShare → Drive)

## Related Documentation

- [Main README](../README.md)
- [OpenAPI Specification](../docs/spec/basemetadata.openapi.spec.yaml)
- [Issue #4: Google Drive Import](https://github.com/Episk-pos/PinShare/issues/4)

## License

Same as PinShare main project.
