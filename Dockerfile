# Build stage
FROM golang:1.25.3-alpine AS builder

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
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

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


