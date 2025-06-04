# AliveHunter Makefile
VERSION = 3.2
BINARY_NAME = alivehunter
BUILD_DIR = build
LDFLAGS = -s -w -X main.VERSION=$(VERSION)

.PHONY: all build install uninstall clean test help

all: build

# Build for current platform
build:
	@echo "Building AliveHunter v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags="$(LDFLAGS)" -trimpath .
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all: build-linux build-windows build-mac

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -ldflags="$(LDFLAGS)" -trimpath .

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe -ldflags="$(LDFLAGS)" -trimpath .

build-mac:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -ldflags="$(LDFLAGS)" -trimpath .

# Install to system
install: build
	@echo "Installing AliveHunter..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "AliveHunter installed to /usr/local/bin/$(BINARY_NAME)"

# Uninstall from system
uninstall:
	@echo "Removing AliveHunter..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "AliveHunter removed"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Show help
help:
	@echo "AliveHunter v$(VERSION) - Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build      - Build for current platform"
	@echo "  make build-all  - Build for all platforms"
	@echo "  make install    - Build and install to system"
	@echo "  make uninstall  - Remove from system"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make help       - Show this help"
