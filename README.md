# [`viamrtsp` module](https://app.viam.com/module/viam/viamrtsp)

This module implements the [`"rdk:component:camera"` API](https://docs.viam.com/components/camera/) for real-time streaming protocol (RTSP) enabled cameras.
Four models are provided:
* `viam:viamrtsp:rtsp` - Codec agnostic. Will auto detect the codec of the `rtsp_address`.
* `viam:viamrtsp:rtsp-h264` - Only supports the H264 codec.
* `viam:viamrtsp:rtsp-h265` - Only supports the H265 codec.
* `viam:viamrtsp:rtsp-mjpeg` - Only supports the M-JPEG codec.
* `viam:viamrtsp:rtsp-mpeg4` - Only supports the MPEG4 codec.
This module also implements the `"rdk:service:discovery"` API, to surface rtsp cameras based on their communication protocol. The following models are implemented:
* `viam:viamrtsp:onvif` - discovers cameras using the [onvif interface](https://www.onvif.org/).
* `viam:viamrtsp:unifi` - discovers cameras connected to a [UniFi Protect](https://ui.com/camera-security) NVR.


Navigate to the [**CONFIGURE** tab](https://docs.viam.com/build/configure/) of your [machine](https://docs.viam.com/fleet/machines/) in the [Viam app](https://app.viam.com/).
[Add the camera component to your machine](https://docs.viam.com/build/configure/#components), searching for `viamrtsp` and selecting your desired model.

## Configure your `viamrtsp` camera

1. Add the `viamrtsp:onvif` discovery service.
1. Use the test tab to find available cameras.
1. Copy (one of) the returned configurations.
1. Add the `viamrtsp` camera.
1. Paste the copied configuration attributes into the JSON configuration:

```json
{
   "rtp_passthrough": true,
   "rtsp_address": "rtsp://tavy16d.viam.local:554/stream"
}
```

Edit the attributes as applicable.

### Attributes

The following attributes are available for all models of `viamrtsp` cameras:

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `rtsp_address` | string | **Required** | The RTSP address where the camera streams. While you can configure a static IP, we recommend using the `viamrtsp:onvif` discovery service to obtain a DNS address. Examples: `"rtsp://foo:bar@192.168.10.10:554/stream"`. |
| `rtp_passthrough` | bool | Optional | RTP passthrough mode (which improves video streaming efficiency) is supported with the H264 codec. It will be on by default. Set to false to disable H264 RTP passthrough. Default: `true`. |
| `lazy_decode` | bool | Optional | The camera only decodes video frames when they're requested via the `Image` API, significantly reducing CPU usage during idle periods. Only compatible with `H264` and `H265` codecs. When disabled (default), the camera continuously decodes the stream to maintain the latest frame. Default: `false`. |
| `i_frame_only_decode` | bool | Optional | Only decodes keyframes (I-frames) from the video stream rather than all incoming frames. This significantly reduces CPU usage at the cost of a lower effective frame rate (typically 1-5 FPS depending on the camera GOP settings). Most suitable for low-motion scenes or when system resources are constrained. Only compatible with `H264` and `H265` codecs. Default: `false`. |
| `transports` | []string | optional | List of transport protocols, in preference order, to use for the RTP stream. Options: `["tcp", "udp", "udp-multicast"]`, Default: `["tcp"]` |

### Example configuration

```json
{
  "rtp_passthrough": true,
  "rtsp_address": "rtsp://tavy16d.viam.local:554/stream"
}
```

**NOTE**
The above is a raw JSON configuration for an `rtsp` model.
To use another provided model, change the "model" string.

## Configure the `viamrtsp:onvif` discovery service

This model is used to locate rtsp cameras on a network that utilize the [onvif interface](https://www.onvif.org/) and surface their configuration.

```json
{
   "credentials": [
    {
      "user": "<USERNAME1>",
      "pass": "<PASSWORD1>"
    }
   ]
}
```

### Attributes

The following attributes are available for all models of `viamrtsp` discovery services:

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `credentials` | struct | Optional | set the username and password for any amount of credentials. |

### Example Configuration

```json
{
   "credentials": [
    {
      "user": "USERNAME1",
      "pass": "PASSWORD1"
    },
    {
      "user": "USERNAME2",
      "pass": "PASSWORD2"
    }
   ]
}
```

### DiscoverResources Extras

The `DiscoverResources` API can also take a credential as `extra`s fields. To discover cameras using this method, add the following to the extras field of the request:

```json
{
  "User": "<username>",
  "Pass": "<password>"
}
```


### Camera Configuration

The `DiscoverResources` API will return a list of cameras discovered by the service and their component configurations. Note that the metadata fields `resolution`, `codec`, and `frame_rate` are descriptive and will not change the behavior of the camera.

```json
{
  "api": "rdk:component:camera",
  "attributes": {
    "discovery_dep": "discovery-1",
    "rtp_passthrough": true,
    "rtsp_address": "rtsp://tavy16d.viam.local:554/stream",
    "resolution": {
      "width": 640,
      "height": 480
    },
    "codec": "H264",
    "frame_rate": 30
  },
  "model": "viam:viamrtsp:rtsp",
  "name": "VIAM-1014255-url1"
}
```

### Preview DoCommand
The `Preview` DoCommand is used to get a preview image of the camera stream. You can copy over the attributes section generated from `DiscoverResources` into the command. The `rtsp_address` is the only required field.

```json
{
  "command": "preview",
  "attributes": {
    "rtsp_address": "rtsp://tavy16d.viam.local:554/stream"
  }
}
```

The response will be preview image in the [DataURL](https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data) format.

```json
{
  "preview": "data:image/jpeg;base64,<base64_encoded_image>"
}
```

### Get Storage State DoCommand

The `get-storage-state` command retrieves the current state of video storage, including available video time ranges and disk usage information.

| Attribute | Type       | Required/Optional | Description          |
|-----------|------------|-------------------|----------------------|
| `command` | string     | required          | The command to be executed. Value must be "get-storage-state". |

#### Get Storage State Request
```json
{
  "command": "get-storage-state"
}
```

#### Get Storage State Response

The response includes a list of `stored_video` time ranges and `disk_usage` statistics.

```json
{
  "command": "get-storage-state",
  "stored_video": [
    {
      "from": "YYYY-MM-DD_HH-MM-SSZ",
      "to": "YYYY-MM-DD_HH-MM-SSZ"
    },
    // ... more ranges
  ],
  "disk_usage": {
    "storage_path": "/path/to/your/storage/directory",
    "storage_used_gb": 99.98,
    "storage_limit_gb": 100,
    "device_storage_remaining_gb": 697.21
  }
}
```

**Response Fields:**

-   `stored_video`: An array of objects, where each object represents a contiguous block of recorded video.
    -   `from`: The start UTC timestamp of the video block.
    -   `to`: The end UTC timestamp of the video block.
-   `disk_usage`:
    -   `storage_path`: The configured path where video segments are stored.
    -   `storage_used_gb`: The amount of disk space (in GB) currently used by the video store in its `storage_path`.
    -   `storage_limit_gb`: The configured maximum disk space (in GB) allocated for the video store.
    -   `device_storage_remaining_gb`: The remaining free disk space (in GB) on the underlying storage device where `storage_path` is located.

#### Example Get Storage State Response

```json
{
  "command": "get-storage-state",
  "stored_video": [
    {
      "to": "2025-05-18_05-59-58Z",
      "from": "2025-05-17_20-23-08Z"
    },
    {
      "from": "2025-05-18_06-00-35Z",
      "to": "2025-05-19_13-56-55Z"
    },
    {
      "from": "2025-05-19_13-57-13Z",
      "to": "2025-05-19_13-58-18Z"
    }
    // ... additional video ranges truncated for brevity
  ],
  "disk_usage": {
    "storage_path": "/root/.viam/video-storage/video_camera-XYZ",
    "storage_used_gb": 99.98937438707799,
    "storage_limit_gb": 100,
    "device_storage_remaining_gb": 697.2153244018555
  }
}
```

> [!NOTE]
> Swapping the `storage_path` config attribute will not delete any data, it will simply cause video store to start persisting video data to the new path and preserve
the old `storage_path` directory with the old videos and the old database, and save new videos in the new path.

### Next steps

Use the `DiscoverResources` API by adding a `viam:viamrtsp:onvif` `discovery` model to retrieve a list of cameras discovered by the service and their configuration. You can then retrieve a preview image from a camera using the `Preview` `DoCommand`. 

Some cameras will output multiple channels, so review the `rtsp_address`, the image preview, and the metadata of the cameras to determine which camera streams you wish to add.

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

## Configure the `viamrtsp:upnp` discovery service

This model is used to locate rtsp cameras on a network that utilize the upnp interface and surface their configuration. Users can define a set of `queries` to detect rtsp cameras with, as well as specify what `endpoints` should be returned with the rtsp address.

```json
{
   "queries": [
    {
      "model_name": "<MODEL_NAME>",
      "manufacturer": "<MANUFACTURER>",
      "serial_number": "<SERIAL_NUMBER>",
      "network": "<NETWORK>",
      "endpoints": ["<ENDPOINT1>","<ENDPOINT2>"]
    }
   ],
   "root_only_search": bool
}
```

### Attributes

The following attributes are available for all models of `viamrtsp` discovery services:

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `queries` | struct | Optional | set any number of device queries to search for cameras with. |
| `root_only_search` | bool | Optional | specify whether the upnp search should search all services, or to only search for root devices. By default is **false**, which will search all services |

#### Query attributes

At least one of `model_name`, `manufacturer`, or `serial_number` must be configured on each query to use the discovery service.

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `model_name` | string | Optional | the model name of the rtsp camera to discover. |
| `manufacturer` | string | Optional | the manufacturer of the rtsp camera to discover. |
| `serial_number` | string | Optional | the serial_number of the rtsp camera to discover. |
| `network` | string | Optional | the network to query for rtsp cameras. Default is `empty`. |
| `endpoints` | []string | Optional | the port and endpoints to configure for the rtsp address if the query is discovered. |

### Example Configuration

this example searches for all devices made by `FLIR` and adds the predefined endpoints to the rtsp address.
```json
{
   "queries": [
    {
      "manufacturer": "FLIR.*",
      "endpoints": ["8554/ir.1","8554/vis.1"]
    }
   ],
   "root_only_search": true
}
```

### DiscoverResources Extras

The DiscoverResources API also can take a credential as `extra`s fields. To discover cameras using this method, add the following to the extras field of the request:

```json
{
  "model_name": "<MODEL_NAME>",
  "manufacturer": "<MANUFACTURER>",
  "serial_number": "<SERIAL_NUMBER>",
  "network": "<NETWORK>",
}
```

Currently specifying endpoints is not supported through the extras field.

## UPnP Host Discovery
If in your rtsp_address your hostname is UPNP_DISCOVER then we will try to find a UPnP host that matches.
You can filter the results by filling out the `query` field in the configuration. See `viamupnp.DeviceQuery` for supported filters.

## Configure the `viamrtsp:unifi` discovery service

This model is used to discover RTSP cameras connected to a [UniFi Protect](https://ui.com/camera-security) NVR. It uses the UniFi Protect Integration API to enumerate cameras and retrieve their RTSP stream URLs.

```json
{
   "nvr_address": "<NVR_IP_OR_HOSTNAME>",
   "unifi_token": "<API_TOKEN>"
}
```

### Attributes

| Name    | Type   | Inclusion    | Description |
| ------- | ------ | ------------ | ----------- |
| `nvr_address` | string | **Required** | The IP address or hostname of the UniFi Protect NVR (e.g., `"10.1.14.106"`). |
| `unifi_token` | string | **Required** | API token for authenticating with the UniFi Protect NVR. Generate this in UniFi Protect under Settings > API Token. |

### Example Configuration

```json
{
   "nvr_address": "10.1.14.106",
   "unifi_token": "aBcDeFgHiJkLmNoPqRsTuVwXyZ123456"
}
```

### Camera Configuration

The `DiscoverResources` API returns a list of cameras discovered from the NVR with their component configurations:

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

**Note:** Camera names are derived from the UniFi Protect camera name (lowercased, spaces replaced with underscores) with a unique ID suffix appended for disambiguation.

### RTSP URL Conversion

The UniFi Protect API returns RTSPS (secure) URLs on port 7441. This discovery service automatically converts them to plain RTSP on port 7447, which is more widely compatible with video clients.

## Configure the `viamrtsp:video-store` generic component for video storage
This model implements the [`"rdk:component:generic"` API](https://docs.viam.com/components/generic/) for storing video data from RTSP cameras. It allows you to save video stream to a local file system. You can later upload clips to cloud storage with `save`, or fetch the video bytes directly with `fetch`.

1. Add a viamrtsp camera component (e.g., `viam:viamrtsp:rtsp`).
2. For cloud upload support, configure a [Data Manager Service](https://docs.viam.com/services/data/cloud-sync/).
3. Configure the `viamrtsp:video-store` component attributes:

```json
{
  "camera": "<rtsp_cam_name>",
  "storage": {
    "size_gb": 1
  }
}
```

### Attributes

| Name                | Type    | Inclusion    | Description |
| ------------------- | ------- | ------------ | ----------- |
| `camera`            | string  | optional     | Name of the camera component to use as the video source |
| `storage`           | object  | required     | Storage configuration settings |
| `storage.size_gb`   | integer | required     | Maximum storage size in gigabytes |
| `storage.upload_path` | string | optional    | Path where uploaded video segments are saved |
| `storage.storage_path` | string | optional   | Path where video segments are stored |
| `video`             | object  | optional     | Video encoding configuration settings (only used when re-encoding is required) |
| `video.bitrate`     | integer | optional     | Bitrate for video encoding (bits per second) - only applies to MPEG4 and MJPEG inputs |
| `video.preset`      | string  | optional     | Encoding preset (e.g., ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow) - only applies to MPEG4 and MJPEG inputs |
| `framerate` | integer | optional | Frame rate to capture video at (frames per second) - only applies to MPEG4 and MJPEG inputs |

### Supported Codecs
The `viamrtsp:video-store` component supports the following codecs:
| Input Codec | Output Codec | Description |
| ----------- | ------------ | ----------- |
| `H264`      | H264         | Direct storage without re-encoding, preserving original quality |
| `H265`      | H265         | Direct storage without re-encoding, preserving original quality |
| `MPEG4`     | H264         | Transcoded to H264 using configured bitrate, preset, and framerate attributes |
| `MJPEG`     | H264         | Re-encoded from frame sequence to H264 video stream using configured bitrate, preset, and framerate attributes |

### DoCommand API

#### From/To

The `From` and `To` timestamps are used to specify the start and end times for video clips. These timestamps must be provided in a specific datetime format to ensure proper parsing and formatting.

##### Datetime Format

The datetime format used is:

- Local Time: `YYYY-MM-DD_HH-MM-SS`
- UTC Time: `YYYY-MM-DD_HH-MM-SSZ`

Where:
- `YYYY`: Year (e.g., 2023)
- `MM`: Month (e.g., 01 for January)
- `DD`: Day (e.g., 15)
- `HH`: Hour in 24-hour format (e.g., 14 for 2 PM)
- `MM`: Minutes (e.g., 30)
- `SS`: Seconds (e.g., 45)
- `Z`: Optional suffix indicating the time is in UTC.

##### Datetime Example

- `2024-01-15_14-30-45` represents January 15, 2024, at 2:30:45 PM **local time**.
- `2024-01-15_14-30-45Z` represents January 15, 2024, at 2:30:45 PM **UTC**.

#### `Save`

The save command retrieves video from local storage, concatenates and trims underlying storage segments based on time range, and writes the clip to a subdirectory of .viam/capture so data manager can upload the clip to the cloud.

| Attribute   | Type                | Required/Optional | Description                      |
|-------------|---------------------|-------------------|----------------------------------|
| `command`   | string              | required          | Command to be executed.          |
| `from`      | timestamp           | required          | Start timestamp.                 |
| `to`        | timestamp           | required          | End timestamp.                   |
| `metadata`  | string              | optional          | Arbitrary metadata string that is appended to filename `<component_name>_<timestamp>_<metadata>.mp4`       |
| `async`     | boolean             | optional          | Whether the operation is async.  |

> [!NOTE]
> Review the [Work with data](https://docs.viam.com/data-ai/data/) documentation for more information on retrieving the saved video file from[Viam Data](https://www.viam.com/product/data).

> [!NOTE]
> If you are requesting video from within the most recent 30 second window, use async save to ensure the current video segment is included in the query.

##### Save Request
```json
{
  "command": "save",
  "from": <start_timestamp>,
  "to": <end_timestamp>,
  "metadata": <arbitrary_metadata_string>
}
```

##### Save Response
```json
{
  "command": "save",
  "filename": <filename_to_be_uploaded>
}
```

> [!NOTE]
> The saved video file will be an MP4 with the video in an encoding format determined by the input codec type. See the [Supported Codecs](#supported-codecs) section for details on how each codec is handled.

##### Async Save Request

The async save command performs the same operation as the save command, but does not wait for the operation to complete. Use this command when you want to save video slices that include the current in-progress video storage segment. It will wait for the current segment to finish recording before saving the video slice.

> [!NOTE]
> The async save command does not support future timestamps. The `from` timestamp must be in the past.
> The `to` timestamp must be the current time or in the past.

```json
{
  "command": "save",
  "from": <start_timestamp>,
  "to": <end_timestamp>,
  "metadata": <arbitrary_metadata_string>,
  "async": true
}
```

##### Async Save Response
```json
{
  "command": "save",
  "filename": <filename_to_be_uploaded>,
  "status": "async"
}
```



#### `Fetch`

The fetch command retrieves video from local storage, and sends the bytes directly back to the client.

| Attribute | Type       | Required/Optional | Description          |
|-----------|------------|-------------------|----------------------|
| `command` | string     | required          | Command to be executed. |
| `from`    | timestamp  | required          | Start timestamp.     |
| `to`      | timestamp  | required          | End timestamp.       |

##### Fetch Request
```json
{
  "command": "fetch",
  "from": <start_timestamp>,
  "to": <end_timestamp>
}
```

##### Fetch Response
```json
{
  "command": "fetch",
  "video": <video_bytes>
}
```
> [!NOTE]
> The returned video bytes will be an MP4 container with video in an encoding format determined by the input codec type. See the [Supported Codecs](#supported-codecs) section for details on how each codec is handled.

## Build for local development

The binary is statically linked with [FFmpeg v6.1](https://github.com/FFmpeg/FFmpeg/tree/release/6.1), eliminating the need to install FFmpeg separately on target machines.

We support building this module using the Makefile for the following host/target combinations:

| Host         | Target       | Supported |
|--------------|--------------|-----------|
| Linux/Arm64  | Linux/Arm64  | ✅        |
| Linux/Arm64  | Android/Arm64| ❌        |
| Linux/Amd64  | Linux/Amd64  | ✅        |
| Linux/Amd64  | Android/Arm64| ✅        |
| Linux/Amd64  | Windows/Amd64| ✅        |
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
* Build for Windows target:
    * Cross-compile from Linux/Amd64 host.
        * Startup canon: `canon -profile viam-rtsp-antique -arch amd64`
        * Build binary: `TARGET_OS=windows TARGET_ARCH=amd64 make`
* Binary will be in `bin/<OS>-<CPU>/viamrtsp`
* Clean up build artifacts: `make clean`
* Clean up all files not tracked in git: `make clean-all`

## Notes

* Non fatal LibAV errors are suppressed unless the module is run in debug mode.
* Heavily cribbed from [gortsplib](https://github.com/bluenviron/gortsplib) examples:
    * [H264 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h264-convert-to-jpeg/main.go)
    * [H265 stream to JPEG](https://github.com/bluenviron/gortsplib/blob/main/examples/client-play-format-h265-convert-to-jpeg/main.go)
    
# Experimental PTZ model

This experimental model implements the [`"rdk:component:generic"` API](https://docs.viam.com/components/generic/) for controlling ONVIF-compliant PTZ (Pan-Tilt-Zoom) cameras. The generic component supports core PTZ operations through the DoCommand method.

## Configure your `onvif-ptz-client`

1. Add the `onvif-ptz-client` generic component
2. Configure connection parameters and profile token:

```json
{
  "address": "192.168.1.100:80",
  "username": "admin",
  "password": "yourpassword",
  "profile_token": "000"
}
```

### Attributes

| Name | Type | Inclusion | Description |
|------|------|-----------|-------------|
| `address` | string | **Required** | Camera IP address with port |
| `username` | string | **Required** | ONVIF authentication username |
| `password` | string | **Required** | ONVIF authentication password |
| `profile_token` | string | Optional | Media profile token for PTZ control (discover with `get-profiles` command) |

### Example Configuration

```json
{
    "name": "ptz-1",
    "api": "rdk:component:generic",
    "model": "viam:viamrtsp:onvif-ptz-client",
    "attributes": {
    "username": "your_username",
      "password": "your_password",
      "profile_token": "your_profile_token",
      "address": "your_camera_ip:port",
  }
}
```

### Supported Commands

#### Get Profiles
```json
{"command": "get-profiles"}
```
Returns list of available media profile tokens.

#### Get Status  
```json
{"command": "get-status"}
```
Returns current PTZ position, movement state, and UTC timestamp.

#### Stop Movement
```json
{
  "command": "stop",
  "pan_tilt": true,
  "zoom": false
}
```
Halts specified movements (default: stop both pan/tilt and zoom).

#### Continuous Move
```json
{
  "command": "continuous-move",
  "pan_speed": 0.5,
  "tilt_speed": -0.2,
  "zoom_speed": 0.1
}
```
Continuous motion at specified speeds (-1.0 to 1.0).

#### Relative Move (Normalized)
```json
{
  "command": "relative-move",
  "pan": 0.1,
  "tilt": -0.05,
  "zoom": 0.1,
  "degrees": false,
  "pan_speed": 0.5,
  "tilt_speed": 0.5,
  "zoom_speed": 0.5
}
```
Relative move using normalized coordinates. Speed parameters are optional.

#### Relative Move (Degrees)
```json
{
  "command": "relative-move",
  "pan": 10,
  "tilt": -5,
  "zoom": 1,
  "degrees": true,
  "pan_speed": 0.2,
  "tilt_speed": 0.2,
  "zoom_speed": 0.5
}
```
Relative move using degree-based coordinates for pan/tilt. Speed parameters are optional.

#### Absolute Move
```json
{
  "command": "absolute-move",
  "pan_position": 0.0,
  "tilt_position": 0.0,
  "zoom_position": 0.5,
  "pan_speed": 1.0,
  "tilt_speed": 1.0,
  "zoom_speed": 1.0
}
```
Absolute position move. Speed parameters are optional.

## Notes

1. **Disclaimer**: This model was made in order to fully integrate with one specific camera. I tried to generalize it to all PTZ cameras, but your mileage may vary.
1. **Profile Discovery**: Use `get-profiles` command to discover valid profile tokens
2. **Coordinate Spaces**:
   - Normalized: -1.0 to 1.0 (pan/tilt), 0.0-1.0 (zoom)
   - Degrees: -180° to 180° (pan), -90° to 90° (tilt)
   - Absolute Moves: Use normalized coordinates (-1.0 to 1.0 for pan/tilt, 0.0 to 1.0 for zoom).
   - Relative Moves:
     - Normalized (`degrees: false`): -1.0 to 1.0 (pan/tilt/zoom).
     - Degrees (`degrees: true`): -180° to 180° (pan), -90° to 90° (tilt). Zoom remains normalized.
3. **Movement Speeds**:
   - Continuous: -1.0 (full reverse) to 1.0 (full forward).
   - Relative/Absolute: Speed parameters (`pan_speed`, `tilt_speed`, `zoom_speed` between 0.0 and 1.0) are optional. If **no** speed parameters are provided, the camera uses its default speed. If **any** speed parameter is provided, the `Speed` element is included in the request (using defaults of 0.5 for Relative or 1.0 for Absolute for any *unspecified* speed components).

## Troubleshooting

**ONVIF Compliance**: Ensure camera supports ONVIF Profile S with PTZ services. Test with ONVIF Device Manager first.

**Authentication Issues**: Some cameras require ONVIF authentication separate from web interface credentials.

**Profile Configuration**: If commands fail with empty profile token:
1. Run `get-profiles` command
2. Copy valid token to configuration
3. Restart component
