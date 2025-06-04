#!/bin/bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

VERSION="3.2"

# Print banner
echo -e "${CYAN}"
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

# Function to print info messages
print_info() {
    echo -e "${BLUE}[i]${NC} $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Go version
check_go_version() {
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local required_version="1.19"
    
    if [ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" != "$required_version" ]; then
        print_error "Go version $required_version or higher is required. Current version: $go_version"
        echo "Please update Go: https://golang.org/doc/install"
        exit 1
    fi
    print_success "Go version $go_version detected"
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
print_status "Checking Go installation..."
if ! command_exists go; then
    print_error "Go is not installed. Please install Go first."
    echo "Visit https://golang.org/doc/install for installation instructions"
    exit 1
fi

# Check Go version
check_go_version

# Create installation directory
INSTALL_DIR="$HOME/.alivehunter"
mkdir -p "$INSTALL_DIR"
print_status "Created installation directory: $INSTALL_DIR"

# Copy source files
print_status "Copying source files..."
if [ -f "main.go" ]; then
    cp main.go "$INSTALL_DIR/"
    print_success "Source files copied"
elif [ -f "AliveHunter.go" ]; then
    cp AliveHunter.go "$INSTALL_DIR/main.go"
    print_success "Source files copied"
else
    print_error "main.go or AliveHunter.go not found in current directory"
    exit 1
fi

# Switch to installation directory
cd "$INSTALL_DIR"

# Initialize Go module
print_status "Initializing Go module..."
go mod init alivehunter
print_success "Go module initialized"

# Download and install dependencies
print_status "Installing dependencies..."
print_info "Downloading github.com/fatih/color..."
go get github.com/fatih/color

print_info "Downloading golang.org/x/net/html..."
go get golang.org/x/net/html

print_info "Downloading golang.org/x/time/rate..."
go get golang.org/x/time/rate

print_status "Cleaning up dependencies..."
go mod tidy
print_success "All dependencies installed"

# Build the binary with optimizations
print_status "Building AliveHunter with optimizations..."
print_info "Compiling for maximum performance..."

# Build with optimizations for speed and size
CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build \
    -o alivehunter \
    -ldflags="-s -w -X main.VERSION=$VERSION" \
    -trimpath \
    main.go

# Verify build
if [ -f alivehunter ]; then
    print_success "Build completed successfully"
    
    # Test the binary
    print_status "Testing binary..."
    if ./alivehunter -h > /dev/null 2>&1; then
        print_success "Binary test passed"
    else
        print_error "Binary test failed"
        exit 1
    fi
    
    # Install the binary
    print_status "Installing binary to /usr/local/bin..."
    sudo mv alivehunter /usr/local/bin/
    sudo chmod +x /usr/local/bin/alivehunter
    
    # Verify installation
    if command_exists alivehunter; then
        print_success "Installation completed successfully!"
    else
        print_error "Installation verification failed"
        exit 1
    fi
else
    print_error "Build failed. Please check the error messages above."
    exit 1
fi

# Clean up installation directory
print_status "Cleaning up temporary files..."
cd "$HOME"
rm -rf "$INSTALL_DIR"
print_success "Cleanup completed"

# Display success message and usage examples
echo
echo -e "${GREEN}╔═══════════════════════════════════════════╗"
echo -e "║       AliveHunter is ready to use! 🎉    ║"
echo -e "╚═══════════════════════════════════════════╝${NC}"
echo
echo -e "${CYAN}🚀 New Features in v$VERSION:${NC}"
echo -e "  ${GREEN}•${NC} Ultra-fast scanning (2-3x faster than httpx)"
echo -e "  ${GREEN}•${NC} Zero false positives with advanced verification"
echo -e "  ${GREEN}•${NC} Pipeline-friendly stdin input"
echo -e "  ${GREEN}•${NC} Multiple operation modes (fast/default/verify)"
echo -e "  ${GREEN}•${NC} Title extraction with robust HTML parsing"
echo -e "  ${GREEN}•${NC} JSON output support"
echo -e "  ${GREEN}•${NC} Configurable TLS versions"
echo
echo -e "${BLUE}📖 Usage Examples:${NC}"
echo -e "  ${CYAN}Basic Usage:${NC}"
echo "    echo 'example.com' | alivehunter"
echo "    cat domains.txt | alivehunter -silent"
echo
echo -e "  ${CYAN}Operation Modes:${NC}"
echo "    cat domains.txt | alivehunter -fast -silent      # Maximum speed"
echo "    cat domains.txt | alivehunter -silent            # Balanced (default)"
echo "    cat domains.txt | alivehunter -verify -silent    # Zero false positives"
echo
echo -e "  ${CYAN}Advanced Features:${NC}"
echo "    cat domains.txt | alivehunter -title             # Extract titles"
echo "    cat domains.txt | alivehunter -json              # JSON output"
echo "    cat domains.txt | alivehunter -mc 200,301        # Match specific codes"
echo
echo -e "  ${CYAN}Pipeline Integration:${NC}"
echo "    subfinder -d target.com | alivehunter -silent"
echo "    cat domains.txt | alivehunter -silent | httpx -title"
echo
echo -e "  ${CYAN}Performance Tuning:${NC}"
echo "    cat domains.txt | alivehunter -t 200 -rate 300   # High performance"
echo "    cat domains.txt | alivehunter -timeout 10s       # Conservative"
echo
echo -e "${BLUE}🔧 Configuration:${NC}"
echo -e "  ${CYAN}Version:${NC} AliveHunter v$VERSION"
echo -e "  ${CYAN}Location:${NC} /usr/local/bin/alivehunter"
echo -e "  ${CYAN}Help:${NC} alivehunter -h"
echo
echo -e "${GREEN}Happy hunting! 🎯${NC}"
echo -e "${YELLOW}Remember: AliveHunter now reads from stdin for pipeline compatibility${NC}"
