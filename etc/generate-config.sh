#!/bin/bash
# Generate viam-server config for integration tests
# Usage: ./generate-config.sh <config_name>

set -e

CONFIG_NAME="${1:-test}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# On Windows, use the module.tar.gz directly (viam-server expects tarball format)
# On Linux/macOS, use the extracted binary
if [[ "$OS" == "mingw"* ]] || [[ "$OS" == "msys"* ]] || [[ "$OS" == "cygwin"* ]] || [[ -n "$WINDIR" ]]; then
    # Windows: use module.tar.gz
    VIAMRTSP_PATH=$(find . -name "module.tar.gz" -type f | head -1)
    if [ -z "$VIAMRTSP_PATH" ]; then
        echo "ERROR: module.tar.gz not found"
        exit 1
    fi
    # Convert to absolute path
    VIAMRTSP_PATH=$(cd "$(dirname "$VIAMRTSP_PATH")" && pwd)/$(basename "$VIAMRTSP_PATH")
    # Convert Unix-style path to Windows-style path (use -m for forward slashes)
    VIAMRTSP_PATH=$(cygpath -m "$VIAMRTSP_PATH")
else
    # Linux/macOS: use extracted binary
    VIAMRTSP_PATH=$(find . -name "viamrtsp" -type f | head -1)
    if [ -z "$VIAMRTSP_PATH" ]; then
        echo "ERROR: viamrtsp binary not found"
        exit 1
    fi
    # Convert to absolute path
    VIAMRTSP_PATH=$(cd "$(dirname "$VIAMRTSP_PATH")" && pwd)/$(basename "$VIAMRTSP_PATH")
fi

echo "Found viamrtsp at: $VIAMRTSP_PATH"

cat > "integration-test-config-${CONFIG_NAME}.json" << EOF
{
  "components": [
    {
      "name": "ip-cam",
      "namespace": "rdk",
      "type": "camera",
      "model": "viam:viamrtsp:rtsp",
      "attributes": {
        "rtsp_address": "rtsp://localhost:8554/live.stream"
      },
      "depends_on": []
    }
  ],
  "modules": [
    {
      "type": "local",
      "name": "viamrtsp",
      "executable_path": "$VIAMRTSP_PATH"
    }
  ]
}
EOF

echo "Generated config: integration-test-config-${CONFIG_NAME}.json"
