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

# Initialize a new Go module (if not already initialized)
if [ ! -f go.mod ]; then
    echo "Initializing Go module..."
    go mod init AliveHunter
fi

# Add necessary dependencies
echo "Downloading dependencies..."
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3

# Ensure dependencies are tidy
echo "Tidying up dependencies..."
go mod tidy

# Compile the program (optional step if you want a binary)
echo "Compiling AliveHunter..."
go build -o alivehunter AliveHunter.go

# Move the binary to a directory in the PATH or create a symlink
if [ -f alivehunter ]; then
    sudo mv alivehunter /usr/local/bin/alivehunter
    echo "AliveHunter has been installed globally as 'alivehunter'."
else
    echo "Compilation failed. The binary was not created."
    exit 1
fi

echo "Installation complete. Thanks for installing! Please visit my other hunting tools at: https://github.com/Acorzo1983"
