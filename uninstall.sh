#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${YELLOW}"
echo "╔═════════════════════════════════╗"
echo "║     AliveHunter Uninstaller     ║"
echo "║       by Albert.C               ║"
echo "╚═════════════════════════════════╝"
echo -e "${NC}"

echo -e "${YELLOW}[*]${NC} Removing AliveHunter..."

# Remove binary
if [ -f "/usr/local/bin/alivehunter" ]; then
    sudo rm -f /usr/local/bin/alivehunter
    echo -e "${GREEN}[+]${NC} Binary removed from /usr/local/bin/"
fi

# Remove any leftover directories
rm -rf "$HOME/.alivehunter" 2>/dev/null

echo -e "${GREEN}[+]${NC} AliveHunter completely removed"
echo -e "${BLUE}[i]${NC} Thank you for using AliveHunter!"
