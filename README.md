# NAME

inboxfewer - Archive Gmail threads for closed GitHub issues and provide MCP server for AI assistants

# SYNOPSIS

```
inboxfewer [command] [options]
inboxfewer cleanup [--account ACCOUNT] [--debug]
inboxfewer serve [--transport TYPE] [--http-addr ADDR] [--yolo] [--debug]
```

# DESCRIPTION

inboxfewer is a tool that archives Gmail threads related to closed GitHub issues and pull requests. It can run as a standalone CLI tool or as a Model Context Protocol (MCP) server, providing AI assistants with programmatic access to Gmail, Google Docs, Google Drive, Google Calendar, Google Meet, and Google Tasks.

# INSTALLATION

## Container Images

Run with Docker:

```bash
# Latest stable release
docker run -p 8080:8080 ghcr.io/teemow/inboxfewer:latest

# Specific version
docker run -p 8080:8080 ghcr.io/teemow/inboxfewer:v1.2.3
```

Deploy with Helm:

```bash
# Install from OCI registry
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer

# With custom configuration
helm install inboxfewer oci://ghcr.io/teemow/charts/inboxfewer \
  --set image.tag=v1.2.3 \
  --values my-values.yaml
```

See [docs/deployment.md](docs/deployment.md) for comprehensive deployment documentation.

## Binary Installation

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -L https://github.com/teemow/inboxfewer/releases/latest/download/inboxfewer_linux_amd64.tar.gz | tar xz

# Linux (arm64)
curl -L https://github.com/teemow/inboxfewer/releases/latest/download/inboxfewer_linux_arm64.tar.gz | tar xz

# macOS (amd64)
curl -L https://github.com/teemow/inboxfewer/releases/latest/download/inboxfewer_darwin_amd64.tar.gz | tar xz

# macOS (arm64)
curl -L https://github.com/teemow/inboxfewer/releases/latest/download/inboxfewer_darwin_arm64.tar.gz | tar xz

# Install to PATH
sudo mv inboxfewer /usr/local/bin/
```

Or install from source:

```bash
go install github.com/teemow/inboxfewer@latest
```

# COMMANDS

## cleanup

Archive Gmail threads related to closed GitHub issues and pull requests.

```bash
inboxfewer cleanup [--account ACCOUNT]
```

## serve

Start the MCP server to provide tools for AI assistants.

```bash
inboxfewer serve [--transport TYPE] [--http-addr ADDR] [--yolo]
```

**Transport Types:**
- stdio: Standard input/output (default, for desktop apps)
- streamable-http: HTTP streaming (HTTP server on --http-addr)

## version

Display version information.

```bash
inboxfewer version
```

## generate-docs

Generate markdown documentation for all MCP tools.

```bash
inboxfewer generate-docs           # Output to stdout
inboxfewer generate-docs -o FILE   # Output to file
```

# OPTIONS

```
--account ACCOUNT      Google account name to use (default: "default")
--debug                Enable debug logging
--transport TYPE       MCP transport: stdio, streamable-http (default: stdio)
--http-addr ADDR       HTTP server address for http transport (default: :8080)
--yolo                 Enable write operations in MCP server (default: read-only)
```

# CONFIGURATION

## GitHub Token

Create `~/keys/github-inboxfewer.token` with:

```
<github-username> <github-personal-access-token>
```

## Google OAuth

On first run, authenticate with Google services. OAuth tokens are cached at:

- Linux/Unix: `~/.cache/inboxfewer/google-{account}.token`
- macOS: `~/Library/Caches/inboxfewer/google-{account}.token`
- Windows: `%TEMP%/inboxfewer/google-{account}.token`

## Multi-Account Support

Manage multiple Google accounts using the `--account` flag:

```bash
inboxfewer cleanup --account work
inboxfewer cleanup --account personal
```

See [docs/configuration.md](docs/configuration.md) for details.

# EXAMPLES

Archive Gmail threads for closed issues:

```bash
inboxfewer cleanup
```

Start MCP server for Claude Desktop:

```bash
inboxfewer serve
```

Start MCP server with HTTP transport:

```bash
inboxfewer serve --transport streamable-http --http-addr :8080
```

Enable all write operations:

```bash
inboxfewer serve --yolo
```

# MCP SERVER

The MCP server provides 66 tools for managing Gmail, Google Docs, Drive, Calendar, Meet, and Tasks. By default, the server operates in read-only mode for safety. Use `--yolo` to enable write operations.

**Key Capabilities:**
- Gmail: List, search, archive, send emails, manage filters
- Google Docs: Retrieve document content in markdown or text
- Google Drive: Upload, download, organize, share files
- Google Calendar: Create events, check availability, schedule meetings
- Google Meet: Access recordings, transcripts, and notes
- Google Tasks: Create and manage task lists

See [docs/tools.md](docs/tools.md) for the complete tool reference (auto-generated from code).

# FILES

```
~/keys/github-inboxfewer.token           GitHub credentials
~/.cache/inboxfewer/google-*.token       Google OAuth tokens (Linux/Unix)
~/Library/Caches/inboxfewer/google-*.token   Google OAuth tokens (macOS)
%TEMP%/inboxfewer/google-*.token         Google OAuth tokens (Windows)
```

# SEE ALSO

- [Configuration Guide](docs/configuration.md)
- [Deployment Guide](docs/deployment.md) - Docker, Kubernetes, Helm
- [Security Guide](docs/security.md) - Security best practices and compliance
- [MCP Tools Reference](docs/tools.md)
- [Development Guide](docs/development.md)
- [Debugging Guide](docs/debugging.md)

# AUTHORS

Original concept by Brad Fitzpatrick.
MCP server integration by @teemow.

# LICENSE

See LICENSE file for details.

# BUGS

Report bugs at: https://github.com/teemow/inboxfewer/issues
