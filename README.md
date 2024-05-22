# [`viamrtsp` module](https://app.viam.com/module/erh/viamrtsp)

This module implements the [`"rdk:component:camera"` API](https://docs.viam.com/components/camera/) for real-time streaming protocol (RTSP) enabled cameras.
Four models are provided:
* `erh:viamrtsp:rtsp` - Codec agnostic. Will auto detect the codec of the `rtsp_address`.
* `erh:viamrtsp:rtsp-h264` - Only supports the H264 codec.
* `erh:viamrtsp:rtsp-h265` - Only supports the H265 codec.
* `erh:viamrtsp:rtsp-mjpeg` - Only supports the M-JPEG codec.

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
| `rtp_passthrough` | bool | **Required** | RTP passthrough mode (which improves video streaming efficiency) is supported with the H264 codec if this attribute is set to `true`. |
| `rtsp_address` | string | **Required** | The RTSP address where the camera streams. |
| `intrinsic_parameters` | object | Optional | The intrinsic parameters of the camera used to do 2D <-> 3D projections: <ul> <li> `width_px`: The expected width of the aligned image in pixels. </li> <li> `height_px`: The expected height of the aligned image in pixels. </li> <li> `fx`: The image center x point. </li> <li> `fy`: The image center y point. </li> <li> `ppx`: The image focal x. </li> <li> `ppy`: The image focal y. </li> </ul> |
| `distortion_parameters` | object | Optional | Modified Brown-Conrady parameters used to correct for distortions caused by the shape of the camera lens: <ul> <li> `rk1`: The radial distortion x. </li> <li> `rk2`: The radial distortion y. </li> <li> `rk3`: The radial distortion z. </li> <li> `tp1`: The tangential distortion x. </li> <li> `tp2`: The tangential distortion y. </li> </ul> |

### Example configuration

```
{
  "components": [
    {
      "name": "your-rtsp-cam",
      "namespace": "rdk",
      "type": "camera",
      "model": "erh:viamrtsp:rtsp",
      "attributes": {
        "rtp_passthrough": true,
        "rtsp_address": "rtsp://foo:bar@192.168.10.10:554/stream"
      }
    }
  ],
  "modules": [
    {
      "type": "registry",
      "name": "erh_viamrtsp",
      "module_id": "erh:viamrtsp",
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

## Build for local development

The binary is statically linked with [FFmpeg v6.1](https://github.com/FFmpeg/FFmpeg/tree/release/6.1), eliminating the need to install FFmpeg separately on target machines.
* Build for Linux targets:
    * Install canon: `go install github.com/viamrobotics/canon@latest`
    * Startup canon dev container.
        * Linux/Arm64: `canon -profile viam-rtsp-antique -arch arm64`
        * Linux/Amd64: `canon -profile viam-rtsp-antique -arch amd64`
    * Build binary: `make`
* Build for MacOS target:
    * Build binary: `make`
* Binary will be in `bin/<OS>-<CPU>/viamrtsp`
* Clean up build artifacts: `make clean`
* Clean up all files not tracked in git: `make clean-all`

## Notes

* Non fatal LibAV errors are suppressed unless the module is run in debug mode.
* Heavily cribbed from [gortsplib](https://github.com/bluenviron/gortsplib) examples:
    * [H264 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h264-convert-to-jpeg/main.go)
    * [H265 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h265-convert-to-jpeg/main.go)
