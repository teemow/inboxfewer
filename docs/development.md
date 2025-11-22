# Development Guide

This document provides information for developers who want to contribute to inboxfewer or build it from source.

## Table of Contents

- [Getting Started](#getting-started)
- [Building from Source](#building-from-source)
- [Running Tests](#running-tests)
- [Code Quality](#code-quality)
- [Project Structure](#project-structure)
- [Makefile Targets](#makefile-targets)
- [Release Process](#release-process)
- [Contributing](#contributing)

## Getting Started

### Prerequisites

- Go 1.23 or later
- Make (optional, for using Makefile targets)
- Git

### Clone the Repository

```bash
git clone https://github.com/teemow/inboxfewer.git
cd inboxfewer
```

### Install Dependencies

```bash
go mod download
```

## Building from Source

### Basic Build

```bash
make build
```

This creates the `inboxfewer` binary in the current directory.

### Install to GOPATH

```bash
make install
```

This installs the binary to `$GOPATH/bin/inboxfewer`.

### Multi-Platform Builds

Build for specific platforms:

```bash
# Linux
make build-linux

# macOS
make build-darwin

# Windows
make build-windows

# All platforms
make build-all
```

Binaries are created in the current directory with platform-specific names (`inboxfewer-linux`, `inboxfewer-darwin`, `inboxfewer-windows.exe`).

## Running Tests

### Run All Tests

```bash
make test
```

### Run Tests with Coverage

```bash
make test-coverage
```

This generates a coverage report at `coverage.html`.

### Run Specific Tests

```bash
go test ./internal/gmail/...
go test -run TestSpecificFunction ./internal/gmail/
```

## Code Quality

### Format Code

```bash
make fmt
```

This runs `go fmt` on all Go files.

### Run Linter

```bash
make lint
```

This requires [golangci-lint](https://golangci-lint.run/) to be installed.

### Run YAML Linter

```bash
make lint-yaml
```

This requires [yamllint](https://github.com/adrienverge/yamllint) to be installed.

### Run All Checks

```bash
make check
```

This runs fmt, vet, test, and lint-yaml.

### Tidy Dependencies

```bash
make tidy
```

This runs `go mod tidy` to clean up dependencies.

## Dockerfiles

The project uses two Dockerfiles for different purposes:

### Dockerfile (Feature Branches)

**Purpose:** Build from source for feature branches and development

**Used by:** `.github/workflows/docker-build.yml`

**Build process:**
- Multi-stage build
- Compiles Go binary during image build
- Single architecture (amd64)
- Ideal for quick iteration and testing

**When to use:**
- Feature branch development
- Local testing
- Integration testing before merge
- Quick prototype deployments

**Build locally:**
```bash
docker build -t inboxfewer:dev .
```

### Dockerfile.release (Production Releases)

**Purpose:** Use pre-built, optimized binaries from GitHub releases

**Used by:** `.github/workflows/docker-release.yml`

**Build process:**
- Single-stage build (no compilation)
- Uses binaries built by GoReleaser
- Multi-architecture (amd64, arm64)
- Smaller, faster builds
- Consistent with released binaries

**When to use:**
- Production deployments
- Official releases
- Multi-architecture support needed
- Maximum optimization required

**Build locally:**
```bash
# First create release binaries
make release-local

# Copy binaries
mkdir -p binaries
cp dist/inboxfewer_linux_amd64*/inboxfewer binaries/inboxfewer_linux_amd64
cp dist/inboxfewer_linux_arm64*/inboxfewer binaries/inboxfewer_linux_arm64

# Build multi-arch image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.release \
  -t inboxfewer:release .
```

## Project Structure

```
inboxfewer/
├── cmd/                    # Command implementations
│   ├── root.go            # Root command
│   ├── cleanup.go         # Cleanup command
│   ├── serve.go           # MCP server command
│   └── version.go         # Version command
├── internal/
│   ├── gmail/             # Gmail client and utilities
│   │   ├── client.go      # Gmail API client
│   │   ├── attachments.go # Attachment retrieval
│   │   ├── doc_links.go   # Google Docs URL extraction
│   │   ├── classifier.go  # Thread classification
│   │   ├── filters.go     # Gmail filter management
│   │   ├── unsubscribe.go # Unsubscribe functionality
│   │   └── types.go       # Type definitions
│   ├── docs/              # Google Docs client
│   │   ├── client.go      # Docs API client
│   │   ├── converter.go   # Document conversion
│   │   └── types.go       # Type definitions
│   ├── drive/             # Google Drive client
│   │   ├── client.go      # Drive API client
│   │   └── types.go       # Type definitions
│   ├── calendar/          # Google Calendar client
│   │   ├── client.go      # Calendar API client
│   │   └── types.go       # Type definitions
│   ├── meet/              # Google Meet client
│   │   ├── client.go      # Meet API client
│   │   └── types.go       # Type definitions
│   ├── tasks/             # Google Tasks client
│   │   ├── client.go      # Tasks API client
│   │   └── types.go       # Type definitions
│   ├── google/            # Unified Google OAuth
│   │   └── oauth.go       # OAuth token management
│   ├── github/            # GitHub types and utilities
│   │   └── types.go       # Type definitions
│   ├── server/            # MCP server context
│   │   └── context.go     # Context management
│   └── tools/             # MCP tool implementations
│       ├── google_tools/  # OAuth tools
│       ├── gmail_tools/   # Gmail tools
│       ├── docs_tools/    # Google Docs tools
│       ├── drive_tools/   # Google Drive tools
│       ├── calendar_tools/# Calendar tools
│       ├── meet_tools/    # Meet tools
│       └── tasks_tools/   # Tasks tools
├── docs/                  # Documentation
│   ├── configuration.md   # Configuration guide
│   ├── tools.md          # MCP tools reference
│   ├── development.md    # This file
│   └── debugging.md      # Debugging guide
├── scripts/              # Utility scripts
│   └── start-mcp-server.sh # Development server script
├── .github/              # GitHub Actions workflows
│   └── workflows/
│       ├── ci.yaml               # Continuous integration
│       ├── auto-release.yaml     # Automated releases (binaries)
│       ├── docker-build.yml      # Docker images (feature branches)
│       ├── docker-release.yml    # Docker images (releases)
│       └── helm-release.yml      # Helm chart publishing
├── charts/               # Helm charts
│   └── inboxfewer/
│       ├── Chart.yaml    # Chart metadata
│       ├── values.yaml   # Default configuration
│       ├── templates/    # Kubernetes manifests
│       └── README.md     # Chart documentation
├── Dockerfile            # Container build from source (dev/feature branches)
├── Dockerfile.release    # Container build from binaries (production)
├── main.go               # Application entry point
├── Makefile              # Build automation
├── .goreleaser.yaml      # GoReleaser configuration
└── go.mod                # Go module definition
```

## Makefile Targets

### Development

- `make build` - Build the binary
- `make install` - Install the binary to GOPATH/bin
- `make clean` - Clean build artifacts
- `make run` - Run the application

### Testing

- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make vet` - Run go vet

### Code Quality

- `make fmt` - Run go fmt
- `make lint` - Run golangci-lint
- `make lint-yaml` - Run YAML linter
- `make tidy` - Run go mod tidy
- `make check` - Run all checks (fmt, vet, test, lint-yaml)

### Release

- `make release-dry-run` - Test the release process without publishing
- `make release-local` - Create a release locally

### Multi-Platform Builds

- `make build-linux` - Build for Linux
- `make build-darwin` - Build for macOS
- `make build-windows` - Build for Windows
- `make build-all` - Build for all platforms

### Help

- `make help` - Display available targets

## Release Process

### Automated Releases

The project uses GitHub Actions for a fully automated release pipeline:

1. **CI Checks** (`.github/workflows/ci.yaml`)
   - Runs on every PR and push to main
   - Executes tests, linting, and formatting checks
   - Validates the release process with a dry-run

2. **Auto Release** (`.github/workflows/auto-release.yaml`)
   - Triggers on merged PRs to main
   - Automatically increments the patch version
   - Creates a git tag
   - Builds binaries for multiple platforms using GoReleaser
   - Publishes a GitHub release with artifacts

3. **Docker Release** (`.github/workflows/docker-release.yml`)
   - Triggers after Auto Release completes successfully
   - Downloads pre-built binaries from the GitHub release
   - Builds multi-architecture Docker images (amd64, arm64)
   - Pushes to GitHub Container Registry (GHCR)
   - Tags: `latest`, `v1.2.3`, `v1.2`, `v1`

4. **Docker Build for Feature Branches** (`.github/workflows/docker-build.yml`)
   - Triggers on PRs and feature branch pushes
   - Builds single-arch (amd64) images from source
   - Enables integration testing before merge
   - Tags: `pr-123`, `feature-branch-name`, `sha-abc123`

5. **Helm Chart Release** (`.github/workflows/helm-release.yml`)
   - Triggers on changes to `charts/**`
   - Packages and publishes Helm charts to GHCR
   - Supports both stable releases and feature branch testing

### Supported Platforms

**Binaries:**
- Linux (amd64, arm64)
- macOS/Darwin (amd64, arm64)
- Windows (amd64, arm64)

**Container Images:**
- linux/amd64
- linux/arm64

**Helm Charts:**
- Published to OCI registry at `oci://ghcr.io/teemow/charts/inboxfewer`

### Manual Release

To test the release process locally:

```bash
# Dry run (no actual release)
make release-dry-run

# Create local release artifacts
make release-local
```

### Container Image Builds

#### Production Images (Multi-Arch)

Production images are built automatically after each release:

```bash
# These are built automatically by CI after release
# Uses Dockerfile.release with pre-built binaries
# Supports: linux/amd64, linux/arm64
# Tags: latest, v1.2.3, v1.2, v1
```

To test the release Dockerfile locally:

```bash
# First create release binaries
make release-local

# Copy binaries to expected location
mkdir -p binaries
cp dist/inboxfewer_linux_amd64*/inboxfewer binaries/inboxfewer_linux_amd64
cp dist/inboxfewer_linux_arm64*/inboxfewer binaries/inboxfewer_linux_arm64

# Build multi-arch image locally (requires Docker Buildx)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.release \
  -t ghcr.io/teemow/inboxfewer:test \
  .
```

#### Feature Branch Images (Single-Arch)

Feature branch images are built automatically on every push:

```bash
# Built from source using Dockerfile
# Supports: linux/amd64 only (for faster CI)
# Tags: pr-42, feature-branch-name, sha-abc123
```

To test the feature branch Dockerfile locally:

```bash
# Build from source (standard Dockerfile)
docker build -t inboxfewer:local .

# Run locally
docker run -p 8080:8080 inboxfewer:local

# Test with custom command
docker run inboxfewer:local serve --debug
```

### Helm Chart Development

#### Testing Chart Changes Locally

```bash
# Lint the chart
helm lint charts/inboxfewer

# Template the chart (dry-run)
helm template inboxfewer charts/inboxfewer \
  --values charts/inboxfewer/values.yaml

# Install locally (requires local Kubernetes cluster)
helm install inboxfewer charts/inboxfewer \
  --set image.tag=local \
  --set googleAuth.enabled=false \
  --dry-run --debug

# Install for real
helm install inboxfewer charts/inboxfewer \
  --set image.tag=latest
```

#### Chart Versioning

Charts are versioned in `charts/inboxfewer/Chart.yaml`:

```yaml
apiVersion: v2
name: inboxfewer
version: 0.1.0  # Chart version
appVersion: "1.2.3"  # App version (matches release)
```

When making chart changes:

1. Update `version` in `Chart.yaml` (following semver)
2. Update `appVersion` if needed
3. Commit changes to `charts/**`
4. On merge to main, chart is automatically published

#### Feature Branch Chart Testing

Feature branches get special chart versions:

```bash
# Chart version includes branch name and commit
# Example: 0.1.0-feature-xyz-abc123

# Install feature branch chart
helm install inboxfewer-test \
  oci://ghcr.io/teemow/charts/inboxfewer \
  --version 0.1.0-feature-xyz-abc123
```

## Contributing

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters
5. Commit your changes
6. Push to your fork
7. Create a pull request

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add tests for new functionality
- Update documentation as needed

### Commit Messages

Use conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(gmail): Add support for email forwarding

Implement the gmail_forward_email tool to allow forwarding
emails to new recipients while preserving the original content.

Closes #123
```

### Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Add a clear description of changes
4. Reference any related issues
5. Wait for review and approval

### Testing Guidelines

- Write unit tests for new functions
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for high test coverage

### Documentation

- Add GoDoc comments for exported functions
- Update relevant documentation files
- Include examples where helpful
- Keep README.md concise (detailed docs go in `docs/`)

## Container and Kubernetes Deployment

For detailed information about deploying inboxfewer using Docker and Kubernetes:

- **[Deployment Guide](deployment.md)** - Comprehensive guide to container images, Helm charts, and deployment scenarios
- Container registry: `ghcr.io/teemow/inboxfewer`
- Helm charts: `oci://ghcr.io/teemow/charts/inboxfewer`

Quick deployment:

```bash
# Run with Docker
docker run -p 8080:8080 ghcr.io/teemow/inboxfewer:latest

# Deploy with Helm
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer
```

## Debugging

See [debugging.md](debugging.md) for information on debugging the MCP server.

### Quick Start

Start the server with debug logging:

```bash
./scripts/start-mcp-server.sh
```

In another terminal, use mcp-debug:

```bash
mcp-debug --repl --endpoint http://localhost:8080/mcp
```

For rebuild and restart:

```bash
./scripts/start-mcp-server.sh --restart
```

## Common Development Tasks

### Adding a New MCP Tool

1. Define the tool in the appropriate `*_tools/` package
2. Implement the handler function
3. Register the tool in the tool list
4. Add tests for the new tool
5. Regenerate the documentation: `./inboxfewer generate-docs -o docs/tools.md`

### Adding a New Google API Integration

1. Create a new client in `internal/<service>/`
2. Define types in `types.go`
3. Implement client methods in `client.go`
4. Add OAuth scopes to `internal/google/oauth.go` if needed
5. Create MCP tools in `internal/tools/<service>_tools/`
6. Add tests for the client and tools
7. Update documentation

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get -u github.com/example/package
go mod tidy
```

### Running Locally

```bash
# Build and run cleanup
make build
./inboxfewer cleanup

# Build and run MCP server
make build
./inboxfewer serve --debug

# Generate tool documentation
./inboxfewer generate-docs -o docs/tools-generated.md

# Run directly without building
go run . serve --debug
```

### Generating Tool Documentation

The tool documentation in `docs/tools.md` is **automatically generated** from the registered MCP tools. This ensures it's always 100% accurate and in sync with the actual tool implementations.

```bash
# Generate documentation and update docs/tools.md
./inboxfewer generate-docs -o docs/tools.md
```

The command introspects all registered tools and extracts their:
- Names and descriptions
- Arguments (type, required/optional, descriptions)
- Categorization (Gmail, Drive, Calendar, etc.)

**Important:** When adding new tools or modifying existing ones, you **must regenerate** the documentation:

```bash
# After adding or modifying tools
make build
./inboxfewer generate-docs -o docs/tools.md
git add docs/tools.md
```

This is a required step before committing any tool-related changes.

## Troubleshooting

### Build Errors

If you encounter build errors:

1. Ensure you have the correct Go version (`go version`)
2. Clean and rebuild: `make clean && make build`
3. Update dependencies: `go mod tidy`

### Test Failures

If tests fail:

1. Run tests with verbose output: `go test -v ./...`
2. Run specific failing test: `go test -v -run TestName ./path/to/package`
3. Check for environment issues (credentials, network)

### Import Errors

If you see import errors:

1. Run `go mod tidy`
2. Verify `go.mod` and `go.sum` are correct
3. Clear module cache: `go clean -modcache`

## Getting Help

- Open an issue on GitHub for bugs or feature requests
- Check existing issues and documentation first
- Provide detailed information when reporting issues

## License

See LICENSE file for details.

