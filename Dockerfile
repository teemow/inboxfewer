# Build stage
FROM golang:1.26.2-alpine@sha256:1fb7391fd54a953f15205f2cfe71ba48ad358c381d4e2efcd820bfca921cd6c6 AS builder

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
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

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


