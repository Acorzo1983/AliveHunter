#!/bin/bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print banner
echo -e "${GREEN}"
echo "AliveHunter Installer v1.0"
echo "Created by Albert.C"
echo -e "${NC}"

# Function to print status messages
print_status() {
    echo -e "${YELLOW}[*]${NC} $1"
}

# Function to print success messages
print_success() {
    echo -e "${GREEN}[+]${NC} $1"
}

# Function to print error messages
print_error() {
    echo -e "${RED}[!]${NC} $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check for root privileges
if [ "$EUID" -eq 0 ]; then 
    print_error "Please do not run this script as root"
    exit 1
fi

# Ensure Go is installed
if ! command_exists go; then
    print_error "Go is not installed. Please install Go first."
    echo "Visit https://golang.org/doc/install for installation instructions"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
MIN_VERSION="1.16"
if [ "$(printf '%s\n' "$MIN_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$MIN_VERSION" ]; then
    print_error "Go version $GO_VERSION is too old. Please upgrade to at least version $MIN_VERSION"
    exit 1
fi

# Create installation directory
INSTALL_DIR="$HOME/.alivehunter"
mkdir -p "$INSTALL_DIR"
print_status "Created installation directory: $INSTALL_DIR"

# Initialize Go module
cd "$INSTALL_DIR"
if [ ! -f go.mod ]; then
    print_status "Initializing Go module..."
    go mod init alivehunter
    print_success "Go module initialized"
fi

# Copy source files
print_status "Copying source files..."
cp "$PWD/AliveHunter.go" "$INSTALL_DIR/"
print_success "Source files copied"

# Download and install dependencies
print_status "Installing dependencies..."
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3
go get golang.org/x/time/rate
go mod tidy
print_success "Dependencies installed"

# Build the binary
print_status "Building AliveHunter..."
go build -o alivehunter AliveHunter.go
print_success "Build completed"

# Create bin directory if it doesn't exist
mkdir -p "$HOME/.local/bin"

# Add ~/.local/bin to PATH if not already present
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc" 2>/dev/null || true
    print_status "Added ~/.local/bin to PATH"
fi

# Install the binary
if [ -f alivehunter ]; then
    mv alivehunter "$HOME/.local/bin/"
    chmod +x "$HOME/.local/bin/alivehunter"
    print_success "AliveHunter installed successfully!"
    echo
    echo -e "${GREEN}Usage:${NC}"
    echo "  alivehunter -l domains.txt               # Basic usage"
    echo "  alivehunter -l domains.txt -p proxy.txt  # With proxies"
    echo "  alivehunter -h                           # Show help"
    echo
    echo -e "${YELLOW}Note:${NC} You may need to restart your terminal or run 'source ~/.bashrc'"
    echo "      for the command to be available."
else
    print_error "Build failed. Please check the error messages above."
    exit 1
fi
