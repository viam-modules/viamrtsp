#!/bin/bash
# Generate viam-server config for integration tests
# Usage: ./generate-config.sh <config_name>

set -e

CONFIG_NAME="${1:-test}"

# Find viamrtsp binary
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if [[ "$OS" == "mingw"* ]] || [[ "$OS" == "msys"* ]] || [[ "$OS" == "cygwin"* ]] || [[ -n "$WINDIR" ]]; then
    VIAMRTSP_PATH=$(find . -name "viamrtsp.exe" -type f | head -1)
else
    VIAMRTSP_PATH=$(find . -name "viamrtsp" -type f | head -1)
fi

if [ -z "$VIAMRTSP_PATH" ]; then
    echo "ERROR: viamrtsp binary not found"
    exit 1
fi

# Convert to absolute path
VIAMRTSP_PATH=$(cd "$(dirname "$VIAMRTSP_PATH")" && pwd)/$(basename "$VIAMRTSP_PATH")

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
