#!/bin/bash
# Run viam-server in background and check it started
# Usage: ./run-viam-server.sh <config_file>

set -e

CONFIG_FILE="${1:-integration-test-config.json}"
WAIT_TIME="${2:-10}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if [[ "$OS" == "mingw"* ]] || [[ "$OS" == "msys"* ]] || [[ "$OS" == "cygwin"* ]] || [[ -n "$WINDIR" ]]; then
    # Windows
    ./viam-server.exe -debug -config "$CONFIG_FILE" &
else
    # Linux/macOS
    viam-server -debug -config "$CONFIG_FILE" &
fi

sleep "$WAIT_TIME"

# Check if viam-server is running
if [[ "$OS" == "mingw"* ]] || [[ "$OS" == "msys"* ]] || [[ "$OS" == "cygwin"* ]] || [[ -n "$WINDIR" ]]; then
    # Windows - use tasklist
    if tasklist | grep -i "viam-server" > /dev/null; then
        echo "viam-server is running"
    else
        echo "ERROR: viam-server is NOT running!"
        exit 1
    fi
else
    # Linux/macOS - use pgrep
    if pgrep -x "viam-server" > /dev/null; then
        echo "viam-server is running"
    else
        echo "ERROR: viam-server is NOT running!"
        exit 1
    fi
fi

sleep "$WAIT_TIME"
