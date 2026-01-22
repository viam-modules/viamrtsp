# UniFi Protect Discovery Service Setup

This guide explains how to configure your UniFi Protect NVR and cameras to work with the `viam:viamrtsp:unifi` discovery service.

## Prerequisites

- UniFi Protect NVR (any device running UniFi Protect, including UNVR, UNVR Pro, Cloud Key Gen2 Plus, UDM Pro, UDM Pro Max, UDM SE, or UDM)
- UniFi Protect cameras connected to the NVR
- Network access from your Viam machine to the NVR

## Step 1: Enable RTSP on Your Cameras

RTSP streaming must be enabled on each camera you want to discover.

1. Open the UniFi Protect web interface (https://your-nvr-ip)
2. Navigate to **Devices** and select a camera
3. Go to **Settings** > **Advanced**
4. Enable **Share livestream** (also called "Enable RTSP Stream" in some versions)
5. Enable **Enable Secure RTSP Output**
6. Select the quality level you want available (High, Medium, Low)

Repeat for each camera you want to use.

> **Note:** When multiple quality levels are enabled on a camera, the discovery service will return the RTSP URL for the highest available quality stream. For example, if you enable both High and Medium quality streams, the discovery service will return the High quality stream URL.

## Step 2: Generate an API Token

The discovery service uses the UniFi Protect Integration API, which requires an API token.

1. Go to [unifi.ui.com](https://unifi.ui.com) and sign in with your Ubiquiti account
2. Select your site/console from the list
3. Click the **Integrations** button in the bottom-left corner
4. Click **Create New API Key**
5. Give it a descriptive name (e.g., "Viam RTSP Discovery")
6. Copy the generated token immediately - it will only be shown once

> **Note:** API keys are generated through the Cloud Site Manager, not the local NVR interface. Your NVR must be linked to your Ubiquiti account.

For more details, see the [official UniFi API documentation](https://developer.ui.com/site-manager-api/gettingstarted).

## Step 3: Configure the Discovery Service

Add the discovery service to your Viam machine configuration:

```json
{
  "name": "unifi-discovery",
  "api": "rdk:service:discovery",
  "model": "viam:viamrtsp:unifi",
  "attributes": {
    "nvr_address": "10.1.14.106",
    "unifi_token": "your-api-token-here"
  }
}
```

### Attributes

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `nvr_address` | string | Yes | IP address or hostname of your UniFi Protect NVR |
| `unifi_token` | string | Yes | API token generated in Step 2 |

## Step 4: Discover Cameras

Once configured, use the `DiscoverResources` API to find available cameras. The service will return configurations like:

```json
{
  "api": "rdk:component:camera",
  "attributes": {
    "rtsp_address": "rtsp://10.1.14.106:7447/abc123DEF456"
  },
  "model": "viam:viamrtsp:rtsp",
  "name": "front_door_abc123"
}
```

## Network and Ports

The UniFi Protect NVR exposes RTSP streams on the following ports:

| Port | Protocol | Description |
|------|----------|-------------|
| 7441 | RTSPS | Encrypted RTSP (TLS) |
| 7447 | RTSP | Unencrypted RTSP |

The discovery service automatically converts RTSPS URLs (port 7441) to plain RTSP (port 7447) for broader compatibility.

Ensure your Viam machine can reach the NVR on port 7447 for video streaming.

## Troubleshooting

### "Authentication failed: invalid or expired API token"

- Verify your API token is correct and hasn't expired
- Regenerate the token if necessary
- Ensure the token has the required permissions

### "No RTSP stream URL available"

- Check that RTSP is enabled on the camera (Step 1)
- Verify the camera is online and connected to the NVR
- Some camera states (updating, disconnected) may not provide streams

### Cameras not appearing in discovery

- Ensure the camera is adopted by the NVR and not in standalone mode
- Check that the camera firmware is up to date
- Verify network connectivity between your machine and the NVR

### Connection timeouts

- Verify the `nvr_address` is correct and reachable
- Check firewall rules allow access to the NVR on ports 443 (API) and 7447 (RTSP)
- The NVR uses self-signed certificates; this is handled automatically

### Stream quality issues

- The discovery service returns the highest quality stream by default
- You can manually modify the `rtsp_address` to use a different quality endpoint if needed
- Check camera bandwidth settings in UniFi Protect

## Camera Naming

Discovered cameras are named using the format: `{camera_name}_{id_suffix}`

- Camera name is lowercased with spaces replaced by underscores
- A 6-character ID suffix is appended for uniqueness
- Example: "Front Door" camera becomes `front_door_abc123`
