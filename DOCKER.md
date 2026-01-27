# Docker Deployment

This directory contains Docker configuration for running tvclipboard in containers.

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker-compose up -d
```

The application will be available at http://localhost:3333

### Using Docker CLI

```bash
docker run -d \
  --name tvclipboard \
  -p 3333:3333 \
  --restart unless-stopped \
  akitaonrails/tvclipboard:latest
```

## Building the Image

```bash
docker build -t akitaonrails/tvclipboard:latest .
```

### Build with specific tag

```bash
docker build -t akitaonrails/tvclipboard:v1.0.0 .
```

## Environment Variables

The following environment variables can be configured in `docker-compose.yml` or via `-e` flag:

| Variable | Default | Description |
|----------|----------|-------------|
| `PORT` | `3333` | Port to listen on |
| `TVCLIPBOARD_SESSION_TIMEOUT` | `10` | Session timeout in minutes |
| `TVCLIPBOARD_PRIVATE_KEY` | (random) | 32-byte hex private key for token encryption |

### Setting Environment Variables

**Using Docker Compose:**
```yaml
environment:
  - PORT=3333
  - TVCLIPBOARD_SESSION_TIMEOUT=15
  - TVCLIPBOARD_PRIVATE_KEY="a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"
```

**Using Docker CLI:**
```bash
docker run -d \
  -e PORT=3333 \
  -e TVCLIPBOARD_SESSION_TIMEOUT=15 \
  -e TVCLIPBOARD_PRIVATE_KEY="a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456" \
  -p 3333:3333 \
  akitaonrails/tvclipboard:latest
```

### Security Note: Private Key

The `TVCLIPBOARD_PRIVATE_KEY` is used to encrypt session tokens:
- If **not set**: A random key is generated on each container restart (existing QR codes will become invalid)
- If **set**: The same key is used across restarts (QR codes remain valid)

For production use, set this to a persistent value via Docker secrets or environment variables.

## Custom Port Mapping

To use a different port on your host:

```bash
docker run -d -p 9000:3333 akitaonrails/tvclipboard:latest
```

Now access at http://localhost:9000

## Pushing to Docker Hub

### First-time setup

1. Login to Docker Hub:
```bash
docker login
```

2. Build and tag the image:
```bash
docker build -t akitaonrails/tvclipboard:latest .
docker build -t akitaonrails/tvclipboard:v1.0.0 .
```

3. Push to Docker Hub:
```bash
docker push akitaonrails/tvclipboard:latest
docker push akitaonrails/tvclipboard:v1.0.0
```

### Using GitHub Actions (Automated)

Create `.github/workflows/docker.yml`:

```yaml
name: Build and Push Docker Image

on:
  push:
    tags:
      - 'v*.*.*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Login to Docker Hub
        uses: docker/login-action@v2
        with:
          username: ${{ secrets.DOCKER_USERNAME }}
          password: ${{ secrets.DOCKER_PASSWORD }}

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: |
            akitaonrails/tvclipboard:latest
            akitaonrails/tvclipboard:${{ github.ref_name }}
```

**Add secrets to GitHub:**
- `DOCKER_USERNAME`: Your Docker Hub username (akitaonrails)
- `DOCKER_PASSWORD`: Your Docker Hub password or access token

## Multi-Architecture Support

To build for multiple architectures (AMD64, ARM64):

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t akitaonrails/tvclipboard:latest \
  --push \
  .
```

## Health Checks

The container includes a health check that verifies the server is running:

```bash
docker inspect --format='{{.State.Health.Status}}' tvclipboard
```

Health check runs every 30 seconds and attempts to connect to http://localhost:3333/

## Troubleshooting

### Container won't start

Check logs:
```bash
docker logs tvclipboard
```

### Port already in use

Change the port mapping:
```bash
docker run -d -p 9090:3333 akitaonrails/tvclipboard:latest
```

### QR codes expire after restart

This happens if `TVCLIPBOARD_PRIVATE_KEY` is not set. Set it to a fixed value:
```bash
docker run -d \
  -e TVCLIPBOARD_PRIVATE_KEY="your-32-byte-hex-key" \
  -p 3333:3333 \
  akitaonrails/tvclipboard:latest
```

Generate a 32-byte hex key:
```bash
openssl rand -hex 32
```

## Image Details

- **Base Image**: Alpine Linux 3.19 (~7MB)
- **Final Size**: ~22MB
- **Go Version**: 1.25.6
- **User**: Non-root (uid 1000)
- **Port**: 3333

## Security Considerations

- Container runs as non-root user
- Minimal Alpine base image
- No shell access in final image
- Static binary (no CGO dependencies)
- Health check included for monitoring

## Production Tips

1. **Use Docker Secrets** for the private key instead of environment variables
2. **Enable HTTPS** via reverse proxy (Traefik, Nginx)
3. **Set restart policy** to `always` or `unless-stopped`
4. **Monitor health checks** via Docker or orchestrator
5. **Pin specific image versions** instead of using `:latest`
