# Custom Sub2API Deployment Guide

This fork includes custom modifications and automatic upstream syncing.

## Custom Modifications

- Fixed duplicate `/v1` in OpenAI responses URL

## Quick Start

### 1. Push Local Changes

```bash
git push origin main
```

### 2. Configure GitHub Actions

1. Go to your repository Settings → Actions → General
2. Set "Workflow permissions" to "Read and write permissions"
3. Save changes

### 3. Set Package Visibility

After the first successful build:
1. Go to your GitHub profile → Packages
2. Find `sub2api` package
3. Package settings → Change visibility → Public

### 4. Deploy with Custom Image

```bash
cd deploy
cp .env.example .env
# Edit .env with your configuration
docker-compose -f docker-compose.custom.yml up -d
```

## Automatic Syncing

The GitHub Action runs daily at 2 AM UTC to:
- Sync with upstream repository
- Build new Docker image if changes detected
- Push to `ghcr.io/peter5842/sub2api:latest`

Manual trigger: Actions tab → "Sync Upstream and Build Docker" → Run workflow

## Local Build

```bash
./build-custom.sh          # Build with 'latest' tag
./build-custom.sh v1.0.0   # Build with custom tag
```

## Verification

```bash
# Check deployment
docker-compose -f docker-compose.custom.yml ps
curl http://localhost:8080/health

# View logs
docker-compose -f docker-compose.custom.yml logs -f sub2api
```

## Image Tags

- `ghcr.io/peter5842/sub2api:latest` - Latest build
- `ghcr.io/peter5842/sub2api:<commit-sha>` - Specific commit
