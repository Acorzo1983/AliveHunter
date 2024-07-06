#!/bin/bash

# Install Go if not installed
if ! command -v go &> /dev/null
then
    echo "Go could not be found. Installing Go..."
    wget https://golang.org/dl/go1.16.4.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.16.4.linux-amd64.tar.gz
    echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
    source ~/.profile
    rm go1.16.4.linux-amd64.tar.gz
else
    echo "Go is already installed"
fi

# Initialize Go module
go mod init AliveHunter

# Install dependencies
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3
go mod tidy

echo "Installation complete. You can now run AliveHunter using 'go run AliveHunter.go'"
