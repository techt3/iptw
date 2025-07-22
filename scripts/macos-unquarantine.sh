#!/bin/bash

# macOS Unquarantine Script for IPTW
# This script removes the quarantine attribute from the IPTW binary
# Use this only if you encounter "Apple could not verify..." warnings

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_PATH=""

# Function to show usage
show_usage() {
    echo "Usage: $0 [path-to-iptw-binary]"
    echo ""
    echo "Examples:"
    echo "  $0                           # Auto-detect iptw in current directory"
    echo "  $0 /usr/local/bin/iptw       # Specify full path"
    echo "  $0 ./build/iptw              # Relative path"
    echo ""
    echo "This script removes the macOS quarantine attribute from the IPTW binary,"
    echo "which allows it to run without Gatekeeper warnings."
}

# Function to find iptw binary
find_binary() {
    # Check if binary path was provided
    if [[ $# -gt 0 ]]; then
        BINARY_PATH="$1"
    else
        # Try to find iptw in common locations
        if [[ -f "./iptw" ]]; then
            BINARY_PATH="./iptw"
        elif [[ -f "../iptw" ]]; then
            BINARY_PATH="../iptw"
        elif [[ -f "iptw" ]]; then
            BINARY_PATH="iptw"
        else
            echo "❌ Could not find iptw binary automatically."
            echo "   Please specify the path to the binary."
            echo ""
            show_usage
            exit 1
        fi
    fi
}

# Function to check if file exists and is executable
check_binary() {
    if [[ ! -f "$BINARY_PATH" ]]; then
        echo "❌ Binary not found: $BINARY_PATH"
        exit 1
    fi
    
    # Make executable if it isn't already
    if [[ ! -x "$BINARY_PATH" ]]; then
        echo "🔧 Making binary executable..."
        chmod +x "$BINARY_PATH"
    fi
}

# Function to remove quarantine
remove_quarantine() {
    echo "🧹 Removing quarantine attribute from: $BINARY_PATH"
    
    if xattr -d com.apple.quarantine "$BINARY_PATH" 2>/dev/null; then
        echo "✅ Quarantine attribute removed successfully!"
    else
        echo "ℹ️  No quarantine attribute found (binary may already be trusted)"
    fi
    
    # Also remove any other extended attributes that might cause issues
    if xattr -c "$BINARY_PATH" 2>/dev/null; then
        echo "🧹 Cleared all extended attributes"
    fi
}

# Function to verify binary
verify_binary() {
    echo "🔍 Verifying binary..."
    
    # Check if it's a valid executable
    if file "$BINARY_PATH" | grep -q "executable"; then
        echo "✅ Binary appears to be a valid executable"
    else
        echo "⚠️  Warning: Binary may not be a valid executable"
    fi
    
    # Try to run version command
    if "$BINARY_PATH" --version 2>/dev/null | head -n 1; then
        echo "✅ Binary responds to version command correctly"
    else
        echo "⚠️  Warning: Binary doesn't respond to --version command"
    fi
}

# Main execution
main() {
    echo "🍎 IPTW macOS Unquarantine Script"
    echo "=================================="
    echo ""
    
    # Handle help
    if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
        show_usage
        exit 0
    fi
    
    # Find and check binary
    find_binary "$@"
    check_binary
    
    echo "Found binary: $BINARY_PATH"
    echo ""
    
    # Remove quarantine
    remove_quarantine
    echo ""
    
    # Verify binary works
    verify_binary
    echo ""
    
    echo "✅ Setup complete! You should now be able to run:"
    echo "   $BINARY_PATH"
    echo ""
    echo "If you still see Gatekeeper warnings, try:"
    echo "   1. Right-click the binary → Open → Open"
    echo "   2. Or go to System Preferences → Security & Privacy → General → Open Anyway"
}

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ This script is only for macOS systems"
    exit 1
fi

# Run main function
main "$@"
