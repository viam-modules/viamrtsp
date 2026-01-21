# UniFi Protect Discovery Service Setup

This guide explains how to configure your UniFi Protect NVR and cameras to work with the `viam:viamrtsp:unifi` discovery service.

## Prerequisites

- UniFi Protect NVR (UNVR, UNVR Pro, Cloud Key Gen2 Plus, or UDM Pro)
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

> **Note:** The discovery service will prefer the highest quality stream available (High > Medium > Low).

## Step 2: Generate an API Token

The discovery service uses the UniFi Protect Integration API, which requires an API token.

1. Log in to your UniFi OS console (https://your-nvr-ip)
2. Click on your profile icon in the bottom-left corner
3. Select **Control Plane** (or navigate to unifi.ui.com and sign in)
4. Go to **API** > **API Keys**
5. Click **Create API Key**
6. Give it a descriptive name (e.g., "Viam RTSP Discovery")
7. Copy the generated token - you won't be able to see it again

For more details, see the [official UniFi API documentation](https://developer.ui.com/site-manager-api/gettingstarted#obtaining-an-api-key).

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
