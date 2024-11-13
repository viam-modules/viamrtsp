# [`viamrtsp` module](https://app.viam.com/module/viam/viamrtsp)

This module implements the [`"rdk:component:camera"` API](https://docs.viam.com/components/camera/) for real-time streaming protocol (RTSP) enabled cameras.
Four models are provided:
* `viam:viamrtsp:rtsp` - Codec agnostic. Will auto detect the codec of the `rtsp_address`.
* `viam:viamrtsp:rtsp-h264` - Only supports the H264 codec.
* `viam:viamrtsp:rtsp-h265` - Only supports the H265 codec.
* `viam:viamrtsp:rtsp-mjpeg` - Only supports the M-JPEG codec.

## Configure your `viamrtsp` camera

Navigate to the [**CONFIGURE** tab](https://docs.viam.com/build/configure/) of your [machine](https://docs.viam.com/fleet/machines/) in the [Viam app](https://app.viam.com/).
[Add the camera component to your machine](https://docs.viam.com/build/configure/#components), searching for `viamrtsp` and selecting your desired model.

Copy and paste the following attributes template into the resulting component's attribute panel:

```
{
   "rtp_passthrough": true,
   "rtsp_address": "rtsp://foo:bar@192.168.10.10:554/stream"
}
```

Edit the attributes as applicable.

### Attributes

The following attributes are available for all models of `viamrtsp` cameras:

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `rtsp_address` | string | **Required** | The RTSP address where the camera streams. |
| `rtp_passthrough` | bool | Optional | RTP passthrough mode (which improves video streaming efficiency) is supported with the H264 codec. It will be on by default. Set to false to disable H264 RTP passthrough. <br> Default: `true` |

### Example configuration

```
{
  "components": [
    {
      "name": "your-rtsp-cam",
      "namespace": "rdk",
      "type": "camera",
      "model": "viam:viamrtsp:rtsp",
      "attributes": {
        "rtp_passthrough": true,
        "rtsp_address": "rtsp://foo:bar@192.168.10.10:554/stream"
      }
    }
  ],
  "modules": [
    {
      "type": "registry",
      "name": "viam_viamrtsp",
      "module_id": "viam:viamrtsp",
      "version": "latest"
    }
  ]
}
```

> [!NOTE]
> The above is a raw JSON configuration for an `rtsp` model.
> To use another provided model, change the "model" string.

### Next steps

To test your camera, go to the [**CONTROL** tab](https://docs.viam.com/fleet/control/) of your machine in the [Viam app](https://app.viam.com) and expand the camera's panel.

## RTSP stream discovery
### Use `DiscoverComponents`
In addition to being able to manually specify RTSP addresses to stream from, `viamrtsp` also offers a discovery utility to search for IP cameras connected to your LAN and return their respective RTSP addresses.

Each model in `viamrtsp` registers a `Discover` method that can be invoked via a `DiscoverComponents` robot API call. See [examples/discovery-client](examples/discovery-client/client.go) for an example usage of discovery and [`DiscoverComponents`](https://docs.viam.com/appendix/apis/robot/#discovercomponents) for documentation.

### Common RTSP discovery pitfalls
#### DHCP
IP camera does not support DHCP, and does not have an assigned IP after connecting to your LAN. In this case, you'll have to assign the camera's IP manually. This can be done through your router's web-based management interface.
To find the IP address of your router's management interface, you can use the following command on Darwin systems:
```
netstat -nr | grep default
```
And the following command on Linux systems:
```
ip route | grep default
```
This will display the IP address of your default gateway, which is usually the IP address of your router. You can then access the router's management interface by typing this IP address in a web browser. Some router interfaces also allow you to find a camera using its MAC address or the specific Ethernet port it's connected to, and manually assign an IP address from there.

#### ONVIF adherence
Discovery relies on the IP camera adhering to the ONVIF Profile S standard, which includes methods such as getting device metadata, media profiles, and stream URIs. It will not work with non-existent or incompatible ONVIF camera integrations that do not meet this profile level.

#### ONVIF authentication
For some IP cameras, ONVIF authentication may be flaky or broken. A workaround is to disable the camera's ONVIF authentication temporarily to discover the RTSP address, then (optionally) re-enable the setting.

## UPnP Host Discovery
If in your rtsp_address your hostname is UPNP_DISCOVER then we will try to find a UPnP host that matches.
You can filter the results by fillong out the `query` field in the configuration. See `viamupnp.DeviceQuery` for supported filters.

## Build for local development

The binary is statically linked with [FFmpeg v6.1](https://github.com/FFmpeg/FFmpeg/tree/release/6.1), eliminating the need to install FFmpeg separately on target machines.

We support building this module using the Makefile for the following host/target combinations:

| Host         | Target       | Supported |
|--------------|--------------|-----------|
| Linux/Arm64  | Linux/Arm64  | ✅        |
| Linux/Arm64  | Android/Arm64| ❌        |
| Linux/Amd64  | Linux/Amd64  | ✅        |
| Linux/Amd64  | Android/Arm64| ✅        |
| Darwin/Arm64 | Darwin/Arm64 | ✅        |
| Darwin/Arm64 | Android/Arm64| ✅        |
| Darwin/Amd64 | Darwin/Amd64 | ❌        |
| Darwin/Amd64 | Android/Arm64| ❌        |

* Build for Linux targets:
    * Install canon: `go install github.com/viamrobotics/canon@latest`
    * Startup canon dev container.
        * Linux/Arm64: `canon -profile viam-rtsp-antique -arch arm64`
        * Linux/Amd64: `canon -profile viam-rtsp-antique -arch amd64`
    * Build binary: `make`
* Build for MacOS target:
    * Build binary: `make`
* Build for Android target:
    * Cross-compile from Linux/Amd64 or Darwin/Arm64 host.
    * To build from Linux/Amd64 host:
        * Startup canon: `canon -profile viam-rtsp-antique -arch amd64`
        * Build binary: `TARGET_OS=android TARGET_ARCH=arm64 make`
    * To build from Darwin/Arm64 host:
        * Build binary: `TARGET_OS=android TARGET_ARCH=arm64 make`
* Binary will be in `bin/<OS>-<CPU>/viamrtsp`
* Clean up build artifacts: `make clean`
* Clean up all files not tracked in git: `make clean-all`

## Notes

* Non fatal LibAV errors are suppressed unless the module is run in debug mode.
* Heavily cribbed from [gortsplib](https://github.com/bluenviron/gortsplib) examples:
    * [H264 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h264-convert-to-jpeg/main.go)
    * [H265 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h265-convert-to-jpeg/main.go)
