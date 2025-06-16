#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' 

REPO_URL="https://github.com/neg4n/wdmt.git" 
BINARY_NAME="wdmt"
INSTALL_DIR="$HOME/.local/bin"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or higher."
        print_status "Visit: https://golang.org/doc/install"
        exit 1
    fi
    
    GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    REQUIRED_VERSION="1.21"
    
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        print_error "Go version $GO_VERSION found, but $REQUIRED_VERSION or higher is required."
        exit 1
    fi
    
    print_success "Go version $GO_VERSION detected"
}

setup_install_dir() {
    if [ ! -d "$INSTALL_DIR" ]; then
        print_status "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
    
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_warning "Install directory $INSTALL_DIR is not in PATH"
        print_status "Add the following line to your shell profile:"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
}

install_wdmt() {
    print_status "Installing WDMT - Web Developer Maintenance Tool"
    echo
    
    check_go
    setup_install_dir
    
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    print_status "Cloning repository..."
    if ! git clone --depth 1 "$REPO_URL" wdmt; then
        print_error "Failed to clone repository"
        exit 1
    fi
    
    cd wdmt
    
    print_status "Downloading dependencies..."
    if ! go mod tidy; then
        print_error "Failed to download dependencies"
        exit 1
    fi
    
    print_status "Building WDMT..."
    if ! go build -o "$BINARY_NAME" -ldflags="-s -w"; then
        print_error "Build failed"
        exit 1
    fi
    
    print_status "Installing binary to $INSTALL_DIR..."
    if ! mv "$BINARY_NAME" "$INSTALL_DIR/"; then
        print_error "Failed to install binary"
        exit 1
    fi
    
    cd /
    rm -rf "$TEMP_DIR"
    
    print_success "WDMT installed successfully!"
    echo
    print_status "Usage: $BINARY_NAME"
    print_status "Run '$BINARY_NAME --help' for more information"
    
    if command -v "$BINARY_NAME" &> /dev/null; then
        VERSION=$($BINARY_NAME --version 2>/dev/null || echo "unknown")
        print_success "Installation verified - WDMT is ready to use!"
    else
        print_warning "Installation complete, but '$BINARY_NAME' not found in PATH"
        print_status "You may need to restart your terminal or add $INSTALL_DIR to your PATH"
    fi
}

if ! command -v git &> /dev/null; then
    print_error "Git is required but not installed."
    exit 1
fi

install_wdmt 