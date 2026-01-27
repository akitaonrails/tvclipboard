# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o tvclipboard .

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/tvclipboard .

# Copy static files (needed for embed)
COPY --from=builder /build/static ./static

# Create non-root user
RUN addgroup -g 1000 tvclipboard && \
    adduser -D -u 1000 -G tvclipboard tvclipboard && \
    chown -R tvclipboard:tvclipboard /app

# Switch to non-root user
USER tvclipboard

# Expose port
EXPOSE 3333

# Set environment variables with defaults
ENV PORT=3333
ENV TVCLIPBOARD_SESSION_TIMEOUT=10

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3333/ || exit 1

# Run the application
CMD ["./tvclipboard"]
