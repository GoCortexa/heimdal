# Heimdal Monorepo Build System
# Supports building both Hardware (Raspberry Pi) and Desktop (Windows/macOS/Linux) products

# Build variables
BINARY_DIR := bin
HARDWARE_BINARY := heimdal-hardware
DESKTOP_BINARY := heimdal-desktop
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Go build flags
GO := go
GOFLAGS := -trimpath
CGO_ENABLED := 1

# Cross-compilation toolchains
ARM64_CC := aarch64-linux-gnu-gcc
MINGW_CC := x86_64-w64-mingw32-gcc

# Create binary directory
$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

#############################################
# Build Targets
#############################################

.PHONY: all
all: build-all

.PHONY: build-all
build-all: build-hardware build-desktop-all

# Hardware product (ARM64 Linux for Raspberry Pi)
.PHONY: build-hardware
build-hardware: $(BINARY_DIR)
	@echo "Building hardware binary for ARM64 Linux..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) \
	CC=$(ARM64_CC) \
	$(GO) build $(GOFLAGS) \
	-o $(BINARY_DIR)/$(HARDWARE_BINARY)-arm64 \
	-ldflags="$(LDFLAGS) -extldflags '-static'" \
	./cmd/heimdal-hardware
	@echo "Hardware binary built: $(BINARY_DIR)/$(HARDWARE_BINARY)-arm64"

# Desktop products - all platforms
.PHONY: build-desktop-all
build-desktop-all: build-desktop-windows build-desktop-macos build-desktop-linux

# Desktop for Windows (amd64)
.PHONY: build-desktop-windows
build-desktop-windows: $(BINARY_DIR)
	@echo "Building desktop binary for Windows amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
	CC=$(MINGW_CC) \
	$(GO) build $(GOFLAGS) \
	-o $(BINARY_DIR)/$(DESKTOP_BINARY)-windows-amd64.exe \
	-ldflags="$(LDFLAGS) -H windowsgui" \
	./cmd/heimdal-desktop
	@echo "Windows desktop binary built: $(BINARY_DIR)/$(DESKTOP_BINARY)-windows-amd64.exe"

# Desktop for macOS (amd64 and arm64)
.PHONY: build-desktop-macos
build-desktop-macos: build-desktop-macos-amd64 build-desktop-macos-arm64

.PHONY: build-desktop-macos-amd64
build-desktop-macos-amd64: $(BINARY_DIR)
	@echo "Building desktop binary for macOS amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
	$(GO) build $(GOFLAGS) \
	-o $(BINARY_DIR)/$(DESKTOP_BINARY)-macos-amd64 \
	-ldflags="$(LDFLAGS)" \
	./cmd/heimdal-desktop
	@echo "macOS amd64 desktop binary built: $(BINARY_DIR)/$(DESKTOP_BINARY)-macos-amd64"

.PHONY: build-desktop-macos-arm64
build-desktop-macos-arm64: $(BINARY_DIR)
	@echo "Building desktop binary for macOS arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) \
	$(GO) build $(GOFLAGS) \
	-o $(BINARY_DIR)/$(DESKTOP_BINARY)-macos-arm64 \
	-ldflags="$(LDFLAGS)" \
	./cmd/heimdal-desktop
	@echo "macOS arm64 desktop binary built: $(BINARY_DIR)/$(DESKTOP_BINARY)-macos-arm64"

# Desktop for Linux (amd64)
.PHONY: build-desktop-linux
build-desktop-linux: $(BINARY_DIR)
	@echo "Building desktop binary for Linux amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
	$(GO) build $(GOFLAGS) \
	-o $(BINARY_DIR)/$(DESKTOP_BINARY)-linux-amd64 \
	-ldflags="$(LDFLAGS)" \
	./cmd/heimdal-desktop
	@echo "Linux desktop binary built: $(BINARY_DIR)/$(DESKTOP_BINARY)-linux-amd64"

# Native build (for current platform)
.PHONY: build-native
build-native: $(BINARY_DIR)
	@echo "Building for native platform..."
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/heimdal -ldflags="$(LDFLAGS)" ./cmd/heimdal-desktop
	@echo "Native binary built: $(BINARY_DIR)/heimdal"

#############################################
# Clean Targets
#############################################

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

.PHONY: clean-all
clean-all: clean
	@echo "Cleaning all generated files..."
	@$(GO) clean -cache -testcache -modcache
	@echo "Clean all complete"

#############################################
# Development Targets
#############################################

.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@echo "Dependencies downloaded"

.PHONY: tidy
tidy:
	@echo "Tidying go.mod..."
	$(GO) mod tidy
	@echo "Tidy complete"

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Format complete"

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "Vet complete"

.PHONY: lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "Lint complete"

#############################################
# Help Target
#############################################

.PHONY: help
help:
	@echo "Heimdal Monorepo Build System"
	@echo ""
	@echo "Build Targets:"
	@echo "  build-all                 - Build all binaries (hardware + desktop)"
	@echo "  build-hardware            - Build hardware binary (ARM64 Linux)"
	@echo "  build-desktop-all         - Build all desktop binaries"
	@echo "  build-desktop-windows     - Build Windows desktop binary"
	@echo "  build-desktop-macos       - Build macOS desktop binaries (amd64 + arm64)"
	@echo "  build-desktop-macos-amd64 - Build macOS amd64 desktop binary"
	@echo "  build-desktop-macos-arm64 - Build macOS arm64 desktop binary"
	@echo "  build-desktop-linux       - Build Linux desktop binary"
	@echo "  build-native              - Build for current platform"
	@echo ""
	@echo "Test Targets:"
	@echo "  test                      - Run all tests"
	@echo "  test-unit                 - Run unit tests only"
	@echo "  test-property             - Run property-based tests"
	@echo "  test-integration          - Run integration tests"
	@echo "  test-platform-windows     - Run Windows-specific tests"
	@echo "  test-platform-macos       - Run macOS-specific tests"
	@echo "  test-platform-linux       - Run Linux-specific tests"
	@echo "  test-coverage             - Run tests with coverage report"
	@echo ""
	@echo "Development Targets:"
	@echo "  deps                      - Download dependencies"
	@echo "  tidy                      - Tidy go.mod"
	@echo "  fmt                       - Format code"
	@echo "  vet                       - Run go vet"
	@echo "  lint                      - Run linter"
	@echo ""
	@echo "Clean Targets:"
	@echo "  clean                     - Remove build artifacts"
	@echo "  clean-all                 - Remove all generated files"
	@echo ""
	@echo "Package Targets:"
	@echo "  package-windows           - Create Windows installer"
	@echo "  package-macos             - Create macOS DMG"
	@echo "  package-linux             - Create Linux packages (deb/rpm)"
	@echo ""
	@echo "Cross-compilation Requirements:"
	@echo "  - aarch64-linux-gnu-gcc for ARM64 Linux"
	@echo "  - x86_64-w64-mingw32-gcc for Windows"
	@echo "  - Xcode command line tools for macOS"


#############################################
# Test Targets
#############################################

.PHONY: test
test:
	@echo "Running all tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "All tests complete"

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GO) test -v -race -short ./...
	@echo "Unit tests complete"

.PHONY: test-property
test-property:
	@echo "Running property-based tests..."
	$(GO) test -v -race ./test/property/...
	@echo "Property-based tests complete"

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -race ./test/integration/...
	@echo "Integration tests complete"

.PHONY: test-platform-windows
test-platform-windows:
	@echo "Running Windows-specific tests..."
	GOOS=windows $(GO) test -v -race ./internal/platform/desktop_windows/...
	@echo "Windows platform tests complete"

.PHONY: test-platform-macos
test-platform-macos:
	@echo "Running macOS-specific tests..."
	GOOS=darwin $(GO) test -v -race ./internal/platform/desktop_macos/...
	@echo "macOS platform tests complete"

.PHONY: test-platform-linux
test-platform-linux:
	@echo "Running Linux-specific tests..."
	GOOS=linux $(GO) test -v -race ./internal/platform/desktop_linux/... ./internal/platform/linux_embedded/...
	@echo "Linux platform tests complete"

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@$(GO) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

.PHONY: test-verbose
test-verbose:
	@echo "Running all tests with verbose output..."
	$(GO) test -v -race -coverprofile=coverage.out ./... 2>&1 | tee test-output.log
	@echo "Test output saved to test-output.log"

#############################################
# Package Targets
#############################################

.PHONY: package-windows
package-windows: build-desktop-windows
	@echo "Creating Windows installer..."
	@if [ ! -d "build/installers/windows" ]; then \
		echo "Windows installer scripts not found in build/installers/windows/"; \
		exit 1; \
	fi
	@echo "Note: Windows installer creation requires NSIS or WiX"
	@echo "Run: makensis build/installers/windows/heimdal-installer.nsi"

.PHONY: package-macos
package-macos: build-desktop-macos
	@echo "Creating macOS package..."
	@if [ ! -d "build/package/macos" ]; then \
		echo "macOS packaging scripts not found in build/package/macos/"; \
		exit 1; \
	fi
	@echo "Note: macOS packaging requires create-dmg or pkgbuild"
	@echo "Run: ./build/package/macos/create-dmg.sh"

.PHONY: package-linux
package-linux: build-desktop-linux
	@echo "Creating Linux packages..."
	@if [ ! -d "build/package/linux" ]; then \
		echo "Linux packaging scripts not found in build/package/linux/"; \
		exit 1; \
	fi
	@echo "Note: Linux packaging requires dpkg-deb and rpmbuild"
	@echo "Run: ./build/package/linux/create-deb.sh && ./build/package/linux/create-rpm.sh"

#############################################
# CI/CD Targets
#############################################

.PHONY: ci
ci: deps fmt vet test-coverage
	@echo "CI pipeline complete"

.PHONY: ci-full
ci-full: deps fmt vet lint test-coverage build-all
	@echo "Full CI pipeline complete"

#############################################
# Install Targets (for development)
#############################################

.PHONY: install
install: build-native
	@echo "Installing binary to /usr/local/bin..."
	@sudo cp $(BINARY_DIR)/heimdal /usr/local/bin/heimdal
	@echo "Installation complete"

.PHONY: uninstall
uninstall:
	@echo "Uninstalling binary from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/heimdal
	@echo "Uninstall complete"
