#!/bin/bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

VERSION="1.4"

# Print banner
echo -e "${GREEN}"
echo "╔═══════════════════════════════╗"
echo "║     AliveHunter Installer     ║"
echo "║     Version $VERSION              ║"
echo "║     Created by Albert.C       ║"
echo "╚═══════════════════════════════╝"
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

# Check for previous installation
check_previous_installation() {
    if [ -d "$HOME/.alivehunter" ] || [ -f "/usr/local/bin/alivehunter" ]; then
        print_status "Previous installation detected"
        read -p "Would you like to remove it and install the new version? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_status "Removing previous installation..."
            sudo rm -rf "$HOME/.alivehunter" 2>/dev/null || true
            sudo rm -f "/usr/local/bin/alivehunter" 2>/dev/null || true
            print_success "Previous installation removed"
        else
            print_error "Installation cancelled"
            exit 1
        fi
    fi
}

# Check for root privileges
if [ "$EUID" -eq 0 ]; then 
    print_error "Please do not run this script as root"
    exit 1
fi

# Check for previous installation
check_previous_installation

# Ensure Go is installed
if ! command_exists go; then
    print_error "Go is not installed. Please install Go first."
    echo "Visit https://golang.org/doc/install for installation instructions"
    exit 1
fi

# Create installation directory
INSTALL_DIR="$HOME/.alivehunter"
mkdir -p "$INSTALL_DIR"
print_status "Created installation directory: $INSTALL_DIR"

# Copy source files
print_status "Copying source files..."
if [ -f "AliveHunter.go" ]; then
    cp AliveHunter.go "$INSTALL_DIR/"
    print_success "Source files copied"
else
    print_error "AliveHunter.go not found in current directory"
    exit 1
fi

# Initialize Go module
cd "$INSTALL_DIR"
print_status "Initializing Go module..."
go mod init alivehunter
print_success "Go module initialized"

# Download and install dependencies
print_status "Installing dependencies..."
go get github.com/fatih/color
go get golang.org/x/time/rate
go mod tidy
print_success "Dependencies installed"

# Build the binary
print_status "Building AliveHunter..."
CGO_ENABLED=0 go build -o alivehunter -ldflags="-s -w" AliveHunter.go

# Install the binary
if [ -f alivehunter ]; then
    sudo mv alivehunter /usr/local/bin/
    sudo chmod +x /usr/local/bin/alivehunter
    print_success "Installation completed successfully!"
    echo
    echo -e "${GREEN}╔═══════════════════════════════════════════╗"
    echo -e "║       AliveHunter is ready to use! 🎉    ║"
    echo -e "╚═══════════════════════════════════════════╝${NC}"
    echo
    echo -e "${BLUE}Usage Examples:${NC}"
    echo "  alivehunter -l urls.txt                    # Basic scan"
    echo "  alivehunter -l urls.txt -o results.txt     # Custom output"
    echo "  alivehunter -l urls.txt -w 20              # 20 workers"
    echo "  alivehunter -l urls.txt -rate 50           # 50 requests/sec"
    echo "  alivehunter -l urls.txt -p proxies.txt     # Use proxies"
    echo "  alivehunter -l urls.txt --https            # HTTPS only"
    echo
    echo -e "${BLUE}Version:${NC} AliveHunter v$VERSION"
    echo -e "${BLUE}Location:${NC} /usr/local/bin/alivehunter"
    echo
    echo -e "${GREEN}Happy hunting! 🎯${NC}"
else
    print_error "Build failed. Please check the error messages above."
    exit 1
fi
