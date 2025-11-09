# PinShare OAuth Broker

A lightweight OAuth 2.0 broker service for PinShare instances to connect to Google Drive without requiring each user to create their own OAuth credentials.

## Why This Exists

PinShare is designed to be run by individual operators on their own infrastructure. However, Google OAuth requires application credentials (Client ID and Secret). This creates a problem:

- **Embedded credentials**: Anyone can extract them from the code
- **User credentials**: High friction - most users won't create them
- **Shared credentials**: Security risk and rate limit issues

The OAuth Broker solves this by providing a centralized service that:
1. Securely holds the official PinShare OAuth credentials
2. Handles the OAuth flow for any PinShare instance
3. Returns tokens via a simple copy-paste flow
4. Works with localhost and firewalled instances

## Architecture

```
┌─────────────────┐
│ User's PinShare │
│   (localhost)   │
└────────┬────────┘
         │ 1. User visits OAuth broker
         ▼
┌─────────────────┐
│  OAuth Broker   │
│ (auth.pinshare) │◄──── 2. Google OAuth callback
└────────┬────────┘
         │ 3. Display token
         ▼
┌─────────────────┐
│      User       │
│  copies token   │
└────────┬────────┘
         │ 4. Paste into PinShare
         ▼
┌─────────────────┐
│ User's PinShare │
│   stores token  │
└─────────────────┘
```

## Setup for Development

### 1. Create Google OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project (or select existing)
3. Enable the Google Drive API
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
5. Application type: **Web application**
6. Authorized redirect URIs:
   ```
   http://localhost:8888/callback
   ```
7. Save your Client ID and Client Secret

### 2. Configure Environment

Create a `.env` file in the project root:

```bash
# OAuth Broker Credentials
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
```

### 3. Run with Docker Compose

From the PinShare root directory:

```bash
docker compose up oauth-broker
```

The broker will be available at http://localhost:8888

### 4. Run Standalone (Without Docker)

```bash
cd oauth-broker
export GOOGLE_CLIENT_ID=your-client-id
export GOOGLE_CLIENT_SECRET=your-client-secret
export REDIRECT_URL=http://localhost:8888/callback
export PORT=8888

go run main.go
```

## Deployment for Production

### Option 1: Fly.io (Recommended)

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
fly auth login

# Create app
fly launch --name pinshare-oauth-broker

# Set secrets
fly secrets set \
  GOOGLE_CLIENT_ID=your-client-id \
  GOOGLE_CLIENT_SECRET=your-client-secret

# Deploy
fly deploy
```

Update your Google OAuth redirect URI to:
```
https://pinshare-oauth-broker.fly.dev/callback
```

### Option 2: Railway

1. Push this directory to a Git repository
2. Import project in Railway
3. Set environment variables:
   - `GOOGLE_CLIENT_ID`
   - `GOOGLE_CLIENT_SECRET`
   - `REDIRECT_URL=https://your-app.railway.app/callback`
4. Deploy

### Option 3: Any VPS

```bash
# Build Docker image
docker build -t pinshare-oauth-broker .

# Run container
docker run -d \
  -p 8888:8888 \
  -e GOOGLE_CLIENT_ID=your-client-id \
  -e GOOGLE_CLIENT_SECRET=your-client-secret \
  -e REDIRECT_URL=https://auth.yourdomain.com/callback \
  pinshare-oauth-broker
```

## Usage Flow

### For End Users

1. Visit your PinShare instance
2. Navigate to **Settings** → **Google Drive Import**
3. Click **"Get Token from OAuth Broker"**
4. You'll be redirected to the broker (e.g., `https://auth.pinshare.io`)
5. Click **"Authorize with Google Drive"**
6. Sign in with your Google account
7. Copy the displayed token
8. Paste it back into PinShare
9. Done! Your instance can now import from Google Drive

### For PinShare Instance Operators

No setup required! Just use the public broker hosted by the PinShare team, or run your own instance.

## Security Considerations

### What This Service Stores

- **Temporary sessions**: State tokens (expire in 10 minutes)
- **Nothing else**: Tokens are displayed once and never stored

### What Users Should Know

- The token gives access to **your** Google Drive
- The token is tied to **your** PinShare instance
- Tokens can be revoked anytime in [Google Account Settings](https://myaccount.google.com/permissions)
- The broker **never** stores or logs tokens

### Rate Limiting

Google enforces OAuth rate limits per Client ID:
- **100 users** per project (free tier)
- **10,000+ users** (requires verification)

For the public broker, this means:
- First 100 users: Instant authorization
- 100-10k users: May require project verification
- 10k+ users: Automatic (no action needed)

If you exceed limits, users can:
1. Wait for the public broker to be verified, OR
2. Run their own broker with their own credentials, OR
3. Set up OAuth directly in their PinShare instance

## API Endpoints

### `GET /`
Home page with instructions

### `GET /authorize`
Initiates OAuth flow
- Redirects to Google OAuth
- Creates temporary session

### `GET /callback`
OAuth callback from Google
- Exchanges code for token
- Displays token to user

### `GET /health`
Health check endpoint
```json
{"status": "ok"}
```

## Monitoring

### Health Check

```bash
curl http://localhost:8888/health
```

### Logs

```bash
# Docker
docker logs pinshare-oauth-broker

# Systemd
journalctl -u pinshare-oauth-broker -f
```

## Troubleshooting

### "Invalid or expired session"

**Cause**: Session timeout (10 minutes) or invalid state parameter

**Solution**: Go back and click "Authorize" again

### "Authorization failed: access_denied"

**Cause**: User denied Google authorization

**Solution**: User needs to accept the permissions request

### "Failed to exchange authorization code"

**Cause**: Invalid redirect URI or Client Secret

**Solution**:
1. Check `REDIRECT_URL` matches Google Console settings
2. Verify `GOOGLE_CLIENT_SECRET` is correct

## License

MIT License - See main PinShare repository

## Contributing

See main PinShare repository for contribution guidelines
