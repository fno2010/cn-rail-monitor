# =============================================================================
# Makefile for cn-rail-monitor
# =============================================================================

# Binary name
BINARY_NAME=cn-rail-monitor
BINARY_DIR=bin

# Go settings
GO=go
GOFLAGS=-v

# Installation directory (default: /usr/local/bin)
PREFIX?=/usr/local
INSTALL_DIR=$(PREFIX)/bin

# Systemd user service directory
SYSTEMD_USER_DIR=$(HOME)/.config/systemd/user

# =============================================================================
# Build Targets
# =============================================================================

.PHONY: all build clean install test run

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd

# Build for different platforms
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd

build-darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd

build-all: clean build build-linux build-darwin
	@echo "Build complete!"
	@ls -lh $(BINARY_DIR)/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)/
	$(GO) clean

# =============================================================================
# Installation Targets
# =============================================================================

# Install binary to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	sudo install -m 755 $(BINARY_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

# Install binary to user directory (no sudo required)
install-user: build
	@echo "Installing $(BINARY_NAME) to $(HOME)/.local/bin..."
	mkdir -p $(HOME)/.local/bin
	install -m 755 $(BINARY_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@echo "Binary installed to $(HOME)/.local/bin. Add '$(HOME)/.local/bin' to your PATH if needed."

# =============================================================================
# Development Targets
# =============================================================================

.PHONY: test test-api test-config test-output test-cover test-race

# Run all tests
test:
	$(GO) test -v ./...

# Run API tests only
test-api:
	$(GO) test -v ./internal/api/...

# Run config tests only
test-config:
	$(GO) test -v ./internal/config/...

# Run output tests only
test-output:
	$(GO) test -v ./internal/output/...

# Run tests with coverage report
test-cover:
	$(GO) test -v -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	$(GO) tool cover -func=coverage.out
	@rm -f coverage.out

# Run tests with race detector
test-race:
	$(GO) test -race -v ./...

# Run with custom config
run:
	$(GO) run ./cmd

# Run with specific config file
run-config:
	$(GO) run ./cmd -config config.yaml

# Development: auto-rebuild on file changes (requiresentr)
dev: build
	@echo "Running in development mode..."
	entr -s 'make build && ./bin/cn-rail-monitor -config config.yaml' ./cmd/**/*.go ./config.yaml

# =============================================================================
# Deployment Targets
# =============================================================================

# Install systemd user service
install-systemd-user:
	@echo "Installing systemd user service..."
	@# Create config directory and copy default config
	mkdir -p $(HOME)/.config/cn-rail-monitor
	@test -f $(HOME)/.config/cn-rail-monitor/config.yaml || cp config.yaml.example $(HOME)/.config/cn-rail-monitor/config.yaml
	@# Install service file with correct binary and config paths
	mkdir -p $(SYSTEMD_USER_DIR)
	sed 's|ExecStart=.*|ExecStart=%h/.local/bin/cn-rail-monitor -config %h/.config/cn-rail-monitor/config.yaml|' systemd/cn-rail-monitor.service > $(SYSTEMD_USER_DIR)/cn-rail-monitor.service
	@echo "Service installed. Config file: $(HOME)/.config/cn-rail-monitor/config.yaml"
	@echo "Run: systemctl --user daemon-reload && systemctl --user start cn-rail-monitor"

# Uninstall systemd user service
uninstall-systemd-user:
	@echo "Uninstalling systemd user service..."
	systemctl --user stop cn-rail-monitor 2>/dev/null || true
	systemctl --user disable cn-rail-monitor 2>/dev/null || true
	rm -f $(SYSTEMD_USER_DIR)/cn-rail-monitor.service
	@echo "Service uninstalled. Config file preserved at $(HOME)/.config/cn-rail-monitor/config.yaml"

# Enable and start systemd user service
start-systemd-user:
	@echo "Starting cn-rail-monitor service..."
	systemctl --user daemon-reload
	systemctl --user enable cn-rail-monitor
	systemctl --user start cn-rail-monitor
	@echo "Service started. Check status with: systemctl --user status cn-rail-monitor"

# Stop systemd user service
stop-systemd-user:
	@echo "Stopping cn-rail-monitor service..."
	systemctl --user stop cn-rail-monitor

# Restart systemd user service
restart-systemd-user:
	@echo "Restarting cn-rail-monitor service..."
	systemctl --user restart cn-rail-monitor

# View service logs
logs-systemd-user:
	@echo "Showing cn-rail-monitor logs..."
	systemctl --user status cn-rail-monitor
	journalctl --user-unit=cn-rail-monitor -f

# =============================================================================
# Docker Targets
# =============================================================================

# Build Docker image
docker-build:
	docker build -t cn-rail-monitor:latest .

# Build with docker-compose
docker-compose-build:
	docker-compose build

# Run with Docker
docker-run:
	docker run -d \
		--name cn-rail-monitor \
		-p 8080:8080 \
		-v $(PWD)/config.yaml:/app/config.yaml:ro \
		cn-rail-monitor:latest

# Run with docker-compose
docker-compose-up:
	docker-compose up -d

# Stop Docker container
docker-stop:
	docker stop cn-rail-monitor 2>/dev/null || true
	docker rm cn-rail-monitor 2>/dev/null || true

# Stop docker-compose
docker-compose-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker logs -f cn-rail-monitor

# View docker-compose logs
docker-compose-logs:
	docker-compose logs -f

# =============================================================================
# Help
# =============================================================================

help:
	@echo "cn-rail-monitor Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build                  - Build the binary"
	@echo "  build-linux            - Build for Linux"
	@echo "  build-darwin           - Build for macOS"
	@echo "  build-all              - Build for all platforms"
	@echo "  clean                  - Clean build artifacts"
	@echo "  install                - Install binary to system (requires sudo)"
	@echo "  install-user           - Install binary to user directory"
	@echo ""
	@echo "  test                   - Run all tests"
	@echo "  test-api               - Run API tests only"
	@echo "  test-config            - Run config tests only"
	@echo "  test-output            - Run output tests only"
	@echo "  test-cover             - Run tests with coverage report"
	@echo "  test-race              - Run tests with race detector"
	@echo ""
	@echo "  run                    - Run the application"
	@echo "  run-config             - Run with config.yaml"
	@echo "  dev                    - Development mode with auto-rebuild"
	@echo ""
	@echo "  install-systemd-user   - Install systemd user service"
	@echo "  uninstall-systemd-user  - Uninstall systemd user service"
	@echo "  start-systemd-user     - Start systemd user service"
	@echo "  stop-systemd-user      - Stop systemd user service"
	@echo "  restart-systemd-user   - Restart systemd user service"
	@echo "  logs-systemd-user      - View service logs"
	@echo ""
	@echo "  docker-build            - Build Docker image"
	@echo "  docker-compose-build  - Build with docker-compose"
	@echo "  docker-run            - Run Docker container"
	@echo "  docker-compose-up     - Run with docker-compose"
	@echo "  docker-stop          - Stop Docker container"
	@echo "  docker-compose-down  - Stop docker-compose"
	@echo "  docker-logs          - View Docker logs"
	@echo "  docker-compose-logs  - View docker-compose logs"
	@echo ""
	@echo "  help                   - Show this help message"
