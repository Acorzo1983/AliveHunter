#!/bin/bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

VERSION="3.2"

# Print banner
echo -e "${CYAN}"
echo "╔═══════════════════════════════════════════╗"
echo "║          AliveHunter Installer            ║"
echo "║            Version $VERSION                   ║"
echo "║      Ultra-fast Bug Bounty Tool           ║"
echo "║         Created by Albert.C               ║"
echo "╚═══════════════════════════════════════════╝"
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

# Function to print feature messages
print_feature() {
    echo -e "${PURPLE}[✓]${NC} $1"
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
    print_success "Go version $go_version detected ✓"
}

# Check system resources
check_system() {
    print_info "Checking system compatibility..."
    
    # Check CPU cores for performance recommendations
    local cores=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "unknown")
    if [ "$cores" != "unknown" ]; then
        print_info "CPU cores detected: $cores (recommended max workers: $((cores * 50)))"
    fi
    
    # Check available memory
    if command_exists free; then
        local mem_gb=$(free -g | awk '/^Mem:/{print $2}')
        print_info "Available RAM: ${mem_gb}GB"
        if [ "$mem_gb" -lt 2 ]; then
            print_error "Warning: Less than 2GB RAM detected. Consider using conservative settings."
        fi
    fi
}

# Check for previous installation
check_previous_installation() {
    if [ -d "$HOME/.alivehunter" ] || [ -f "/usr/local/bin/alivehunter" ]; then
        print_status "Previous AliveHunter installation detected"
        read -p "Would you like to remove it and install v$VERSION? (y/n) " -n 1 -r
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

# Display features
display_features() {
    echo -e "${CYAN}🚀 AliveHunter v$VERSION Features:${NC}"
    print_feature "Ultra-fast web discovery (2-3x faster than httpx)"
    print_feature "Zero false positives with smart verification"
    print_feature "Multiple output formats (clean, detailed, JSON)"
    print_feature "File input (-l) and pipe support for bug bounty workflows"
    print_feature "Advanced filtering with status code matching"
    print_feature "Title extraction with robust HTML parsing"
    print_feature "High-performance scanning (500+ req/s in fast mode)"
    print_feature "Perfect integration with nuclei, httpx, subfinder"
    echo
}

# Check for root privileges
if [ "$EUID" -eq 0 ]; then 
    print_error "Please do not run this script as root"
    exit 1
fi

# Display features
display_features

# Check for previous installation
check_previous_installation

# System compatibility check
check_system

# Ensure Go is installed
print_status "Checking Go installation..."
if ! command_exists go; then
    print_error "Go is not installed. Please install Go first."
    echo "Visit https://golang.org/doc/install for installation instructions"
    echo "Minimum required version: 1.19"
    exit 1
fi

# Check Go version
check_go_version

# Create installation directory
INSTALL_DIR="$HOME/.alivehunter"
mkdir -p "$INSTALL_DIR"
print_status "Created installation directory: $INSTALL_DIR"

# Copy source files with better detection
print_status "Copying source files..."
if [ -f "main.go" ]; then
    cp main.go "$INSTALL_DIR/"
    print_success "Source files copied (main.go)"
elif [ -f "AliveHunter.go" ]; then
    cp AliveHunter.go "$INSTALL_DIR/main.go"
    print_success "Source files copied (AliveHunter.go → main.go)"
elif [ -f "alivehunter.go" ]; then
    cp alivehunter.go "$INSTALL_DIR/main.go"
    print_success "Source files copied (alivehunter.go → main.go)"
else
    print_error "Go source file not found. Expected: main.go, AliveHunter.go, or alivehunter.go"
    exit 1
fi

# Switch to installation directory
cd "$INSTALL_DIR"

# Initialize Go module
print_status "Initializing Go module..."
go mod init alivehunter > /dev/null 2>&1
print_success "Go module initialized"

# Download and install dependencies with progress
print_status "Installing dependencies..."
print_info "Downloading github.com/fatih/color (terminal colors)..."
go get github.com/fatih/color

print_info "Downloading golang.org/x/net/html (robust HTML parsing)..."
go get golang.org/x/net/html

print_info "Downloading golang.org/x/time/rate (rate limiting)..."
go get golang.org/x/time/rate

print_status "Optimizing dependencies..."
go mod tidy
print_success "All dependencies installed successfully"

# Build the binary with optimizations
print_status "Building AliveHunter with performance optimizations..."
print_info "Compiling for maximum speed and minimal binary size..."

# Build with comprehensive optimizations
CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build \
    -o alivehunter \
    -ldflags="-s -w -X main.VERSION=$VERSION" \
    -trimpath \
    -buildmode=exe \
    main.go

# Verify build
if [ -f alivehunter ]; then
    # Get binary size
    binary_size=$(du -h alivehunter | cut -f1)
    print_success "Build completed successfully (${binary_size})"
    
    # Test the binary
    print_status "Testing binary functionality..."
    if ./alivehunter -h > /dev/null 2>&1; then
        print_success "Binary test passed ✓"
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

# Display success message and comprehensive usage examples
echo
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗"
echo -e "║                AliveHunter v$VERSION is ready! 🎉                 ║"
echo -e "╚═══════════════════════════════════════════════════════════════╝${NC}"

echo
echo -e "${CYAN}🎯 Quick Start for Bug Bounty:${NC}"
echo -e "  ${GREEN}▸${NC} alivehunter -l scope.txt -fast -silent > live.txt"
echo -e "  ${GREEN}▸${NC} cat scope.txt | alivehunter -silent | nuclei -t cves/"

echo
echo -e "${CYAN}📖 Common Usage Patterns:${NC}"

echo -e "\n  ${YELLOW}🔹 File Input (Direct):${NC}"
echo "    alivehunter -l domains.txt                     # Basic scan with details"
echo "    alivehunter -l domains.txt -fast -silent       # Fast scan, clean output"
echo "    alivehunter -l domains.txt -verify -title      # Zero false positives"

echo -e "\n  ${YELLOW}🔹 Pipeline Integration:${NC}"
echo "    subfinder -d target.com | alivehunter -silent  # With subfinder"
echo "    cat domains.txt | alivehunter -fast -silent    # From file via pipe"

echo -e "\n  ${YELLOW}🔹 Output Formats:${NC}"
echo "    alivehunter -l scope.txt -silent               # Clean URLs for tools"
echo "    alivehunter -l scope.txt -title                # Detailed with titles"
echo "    alivehunter -l scope.txt -json                 # JSON for processing"

echo -e "\n  ${YELLOW}🔹 Performance Tuning:${NC}"
echo "    alivehunter -l big_scope.txt -fast -t 300 -rate 500    # High speed"
echo "    alivehunter -l scope.txt -t 50 -rate 25                # Conservative"

echo -e "\n  ${YELLOW}🔹 Advanced Filtering:${NC}"
echo "    alivehunter -l scope.txt -mc 200,301                   # Specific codes"
echo "    alivehunter -l scope.txt -mc 401,403 -silent           # Auth endpoints"

echo
echo -e "${CYAN}🚀 Complete Bug Bounty Workflow:${NC}"
echo "  # 1. Fast initial scope validation"
echo "  alivehunter -l scope.txt -fast -silent > live.txt"
echo
echo "  # 2. Vulnerability scanning with nuclei"
echo "  nuclei -l live.txt -t cves/ -o vulnerabilities.txt"
echo
echo "  # 3. Technology detection"
echo "  cat live.txt | httpx -title -tech -probe > detailed.txt"
echo
echo "  # 4. Critical target verification"
echo "  alivehunter -l priority.txt -verify -json > verified.json"

echo
echo -e "${CYAN}⚡ Performance Modes:${NC}"
echo -e "  ${GREEN}Fast Mode:${NC}    ~500+ req/s (minimal verification)"
echo -e "  ${YELLOW}Default:${NC}      ~300 req/s (balanced accuracy)"
echo -e "  ${BLUE}Verify Mode:${NC}  ~150 req/s (zero false positives)"

echo
echo -e "${CYAN}🔧 Installation Details:${NC}"
echo -e "  ${GREEN}Version:${NC}      AliveHunter v$VERSION"
echo -e "  ${GREEN}Location:${NC}     /usr/local/bin/alivehunter"
echo -e "  ${GREEN}Config:${NC}       Run 'alivehunter -h' for full options"
echo -e "  ${GREEN}Integration:${NC}  Works with nuclei, httpx, subfinder, amass"

echo
echo -e "${GREEN}📚 Documentation:${NC}"
echo "  Full help:     alivehunter -h"
echo "  Repository:    github.com/Acorzo1983/AliveHunter"
echo "  Issues:        Report bugs and feature requests on GitHub"

echo
echo -e "${GREEN}💡 Pro Tips:${NC}"
echo "  • Use -fast for large scope files (10k+ domains)"
echo "  • Use -silent for clean output perfect for pipelines"
echo "  • Use -verify for critical targets requiring 100% accuracy"
echo "  • Combine with nuclei: alivehunter -l scope.txt -silent | nuclei"
echo "  • JSON output works great with jq for filtering"

echo
color.New(color.FgHiGreen).Println("Made with ❤️ by Albert.C")
echo -e "${YELLOW}Happy Bug Bounty Hunting! 🎯${NC}"
echo
