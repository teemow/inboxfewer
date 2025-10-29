.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: build
build: ## Build the binary
	@echo "Building inboxfewer..."
	@go build -o inboxfewer .

.PHONY: install
install: ## Install the binary to GOPATH/bin
	@echo "Installing inboxfewer..."
	@go install .

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f inboxfewer
	@rm -rf dist/
	@go clean

.PHONY: run
run: ## Run the application
	@go run .

##@ Testing

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -cover ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

##@ Code Quality

.PHONY: fmt
fmt: ## Run go fmt
	@echo "Running go fmt..."
	@go fmt ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed)
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

.PHONY: lint-yaml
lint-yaml: ## Run YAML linter (requires yamllint installed)
	@echo "Running YAML linter..."
	@yamllint .github/workflows/*.yaml .goreleaser.yaml

.PHONY: tidy
tidy: ## Run go mod tidy
	@echo "Running go mod tidy..."
	@go mod tidy

.PHONY: check
check: fmt vet test lint-yaml ## Run all checks (fmt, vet, test, lint-yaml)

##@ Release

.PHONY: release-dry-run
release-dry-run: ## Test the release process without publishing (requires goreleaser)
	@echo "Running release dry-run..."
	@goreleaser release --snapshot --clean --skip=announce,publish,validate

.PHONY: release-local
release-local: ## Create a release locally (requires goreleaser)
	@echo "Running local release..."
	@goreleaser release --clean

##@ Build Variants

.PHONY: build-linux
build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o inboxfewer-linux .

.PHONY: build-darwin
build-darwin: ## Build for macOS
	@echo "Building for macOS..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o inboxfewer-darwin .

.PHONY: build-windows
build-windows: ## Build for Windows
	@echo "Building for Windows..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o inboxfewer-windows.exe .

.PHONY: build-all
build-all: build-linux build-darwin build-windows ## Build for all platforms
