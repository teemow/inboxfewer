# Build stage
FROM golang:1.27.1-alpine@sha256:4cb7ac979db5fcc41cae44b2227ba5ab8a51e8807f40d9ba4dee20a0ad960b5b AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary for the target platform
# Docker buildx automatically provides TARGETOS and TARGETARCH
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o inboxfewer .

# Final stage
FROM alpine:3.24@sha256:5b02b42e375f7426f8d65c3af331ca05d9878f9989230354504e0b9dfd431f60

# Upgrade base packages to pick up security patches (CVE-2026-22184: zlib)
# and install ca-certificates for HTTPS in a single layer
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates && \
    apk info zlib | head -1 | grep -q '^zlib-1\.3\.[2-9]'

# Create non-root user
RUN addgroup -g 1000 inboxfewer && \
    adduser -D -u 1000 -G inboxfewer inboxfewer

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/inboxfewer .

# Change ownership
RUN chown -R inboxfewer:inboxfewer /app

# Switch to non-root user
USER inboxfewer

# Expose default HTTP port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/inboxfewer"]

# Default to serve command with streamable-http transport
CMD ["serve", "--transport", "streamable-http", "--http-addr", ":8080"]


