# Pushing to Docker Hub

## Prerequisites

1. Docker Hub account: `akitaonrails`
2. Docker installed locally

## Build and Push Commands

### 1. Login to Docker Hub

```bash
docker login
# Enter your Docker Hub username and password
```

### 2. Build the image

```bash
# Build with latest tag
docker build -t akitaonrails/tvclipboard:latest .

# Build with version tag
docker build -t akitaonrails/tvclipboard:v1.0.0 .
```

### 3. Push to Docker Hub

```bash
# Push latest
docker push akitaonrails/tvclipboard:latest

# Push version
docker push akitaonrails/tvclipboard:v1.0.0
```

## Complete Workflow

```bash
# Build and tag
docker build -t akitaonrails/tvclipboard:latest .
docker tag akitaonrails/tvclipboard:latest akitaonrails/tvclipboard:v1.0.0

# Push both tags
docker push akitaonrails/tvclipboard:latest
docker push akitaonrails/tvclipboard:v1.0.0

# Verify
docker pull akitaonrails/tvclipboard:latest
```

## After First Push

Once the image is pushed to Docker Hub, you can run it with:

```bash
docker run -d \
  --name tvclipboard \
  -p 3333:3333 \
  --restart unless-stopped \
  akitaonrails/tvclipboard:latest
```

Or using docker-compose (update `docker-compose.yml`):

```yaml
services:
  tvclipboard:
    image: akitaonrails/tvclipboard:latest
    # ... rest of configuration
```

## Multi-Architecture Builds (Optional)

To support both AMD64 and ARM64:

```bash
# Enable buildx
docker buildx create --use

# Build and push for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t akitaonrails/tvclipboard:latest \
  --push \
  .
```

## Automated Builds (GitHub Actions)

See `DOCKER.md` for GitHub Actions workflow configuration.
