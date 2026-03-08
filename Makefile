# IPTW Cross-Platform Build Makefile
# This Makefile supports building for multiple platforms and architectures

# Application name and version
APP_NAME = iptw
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -w -s"
LDFLAGS_WINDOWS = -ldflags "-H windowsgui -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -w -s"
BUILD_FLAGS = $(LDFLAGS) -trimpath

# Build directory
BUILD_DIR = build
DIST_DIR = dist

# Platforms and architectures to build for
PLATFORMS = \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 
# Main entry point
MAIN_PACKAGE = ./cmd/iptw

# Default target
.PHONY: all
all: clean build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)

# Create build directories
.PHONY: dirs
dirs:
	@mkdir -p $(BUILD_DIR) $(DIST_DIR)

# Executable extension (.exe on Windows)
BINARY_EXT = $(if $(filter Windows_NT,$(OS)),.exe,)

# Build for current platform
.PHONY: build
build: dirs
	@echo "🔨 Building $(APP_NAME) for current platform..."
	@go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)$(BINARY_EXT) $(MAIN_PACKAGE)
	@echo "✅ Build complete: $(BUILD_DIR)/$(APP_NAME)$(BINARY_EXT)"

# Build for all platforms
.PHONY: build-all
build-all: dirs
	@echo "🔨 Building $(APP_NAME) for all platforms..."
	@$(foreach platform,$(PLATFORMS),$(call build_platform,$(platform)))
	@echo "✅ All builds complete!"

# Function to build for a specific platform
define build_platform
	$(eval GOOS := $(word 1,$(subst /, ,$(1))))
	$(eval GOARCH := $(word 2,$(subst /, ,$(1))))
	$(eval BINARY_NAME := $(APP_NAME)$(if $(filter windows,$(GOOS)),.exe))
	$(eval OUTPUT_DIR := $(BUILD_DIR)/$(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH))
	$(eval PLATFORM_BUILD_FLAGS := $(if $(filter windows,$(GOOS)),$(LDFLAGS_WINDOWS) -trimpath,$(BUILD_FLAGS)))
	$(eval CGO_FLAG := $(if $(filter darwin,$(GOOS)),CGO_ENABLED=1,CGO_ENABLED=0))
	@echo "  📦 Building for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	@if GOOS=$(GOOS) GOARCH=$(GOARCH) $(CGO_FLAG) go build $(PLATFORM_BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE) 2>/dev/null; then \
		echo "    ✅ $(GOOS)/$(GOARCH) build complete"; \
	else \
		echo "    ⚠️  $(GOOS)/$(GOARCH) cross-compilation failed"; \
		rm -rf $(OUTPUT_DIR); \
	fi
	@cp README.md $(OUTPUT_DIR)/ 2>/dev/null || true
	@cp -r config $(OUTPUT_DIR)/ 2>/dev/null || true
endef

# Create release archives
.PHONY: package
package: build-all
	@echo "📦 Creating release packages..."
	@cd $(BUILD_DIR) && \
	for dir in $(APP_NAME)-$(VERSION)-*; do \
		if [ -d "$$dir" ]; then \
			echo "  📦 Packaging $$dir..."; \
			if [[ "$$dir" == *"windows"* ]]; then \
				zip -r "../$(DIST_DIR)/$$dir.zip" "$$dir/"; \
			else \
				tar -czf "../$(DIST_DIR)/$$dir.tar.gz" "$$dir/"; \
			fi; \
		fi; \
	done
	@echo "✅ All packages created in $(DIST_DIR)/"

# Development build with live reload support
.PHONY: dev
dev:
	@echo "🚀 Starting development build..."
	@go run $(MAIN_PACKAGE) -config config/config.yaml

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, running go vet instead..."; \
		go vet ./...; \
	fi

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "🧹 Tidying dependencies..."
	@go mod tidy

# Security audit
.PHONY: audit
audit:
	@echo "🔒 Running security audit..."
	@go list -json -m all | nancy sleuth

# Install development dependencies
.PHONY: install-dev-deps
install-dev-deps:
	@echo "📦 Installing development dependencies..."
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.11.1
	@echo "✅ Development dependencies installed"

# Quick build and run
.PHONY: run
run: build
	@echo "🚀 Running $(APP_NAME)..."
	@./$(BUILD_DIR)/$(APP_NAME)


.PHONY: release
release: clean test lint build-all package
	@echo "🎉 Release $(VERSION) is ready!"
	@echo "📦 Packages available in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

# Install locally
.PHONY: install
install: build
	@echo "📥 Installing $(APP_NAME) locally..."
	@cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)
	@echo "✅ $(APP_NAME) installed to /usr/local/bin/$(APP_NAME)"

# Uninstall locally
.PHONY: uninstall
uninstall:
	@echo "🗑️ Uninstalling $(APP_NAME)..."
	@rm -f /usr/local/bin/$(APP_NAME)
	@echo "✅ $(APP_NAME) uninstalled"

# Show version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Help target
.PHONY: help
help:
	@echo "IPTW Build System"
	@echo "================="
	@echo ""
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for all supported platforms"
	@echo "  package        - Create release packages"
	@echo "  release        - Full release build (test, lint, build-all, package)"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  tidy           - Tidy dependencies"
	@echo "  audit          - Security audit"
	@echo "  dev            - Development build with config"
	@echo "  run            - Build and run"
	@echo "  install        - Install locally to /usr/local/bin"
	@echo "  uninstall      - Uninstall from /usr/local/bin"
	@echo "  version        - Show version information"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION        - Override version (default: git tag or v0.1.0)"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build for current platform"
	@echo "  make release                  # Create full release"
	@echo "  VERSION=v1.0.0 make package   # Build with custom version"
