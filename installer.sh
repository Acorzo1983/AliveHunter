#!/bin/bash

# Installer script for AliveHunter

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Ensure Go is installed
if ! command_exists go; then
    echo "Go is not installed. Please install Go and run this script again."
    exit 1
fi

# Set up the directory
INSTALL_DIR="$HOME/Tools/AliveHunter"
mkdir -p "$INSTALL_DIR"

# Download the AliveHunter.go file
echo "Downloading AliveHunter.go..."
curl -s -o "$INSTALL_DIR/AliveHunter.go" "https://your-download-link/AliveHunter.go"

# Navigate to the installation directory
cd "$INSTALL_DIR" || exit

# Initialize a new Go module
echo "Initializing Go module..."
go mod init AliveHunter

# Add necessary dependencies
echo "Downloading dependencies..."
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3

echo "Installation complete. You can now run AliveHunter using the following command:"
echo "go run $INSTALL_DIR/AliveHunter.go -l <input_file> -o <output_file>"
