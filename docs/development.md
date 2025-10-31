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
│       ├── ci.yaml       # Continuous integration
│       └── auto-release.yaml # Automated releases
├── main.go               # Application entry point
├── Makefile              # Build automation
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

The project uses GitHub Actions for automated releases:

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

### Supported Platforms

Releases include pre-built binaries for:
- Linux (amd64, arm64)
- macOS/Darwin (amd64, arm64)
- Windows (amd64, arm64)

### Manual Release

To test the release process locally:

```bash
# Dry run (no actual release)
make release-dry-run

# Create local release artifacts
make release-local
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

