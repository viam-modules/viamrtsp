# PTZ model

This module exposes two PTZ models:
- `viam:viamrtsp:onvif-ptz` implements the PTZ gRPC API (`rdk:component:ptz`) for standardized PTZ control.
- `viam:viamrtsp:onvif-ptz-client` implements the legacy [`"rdk:component:generic"` API](https://docs.viam.com/components/generic/) and supports PTZ control through DoCommand.

Use `onvif-ptz` for new integrations and keep `onvif-ptz-client` for backwards compatibility.

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

## Configure your `onvif-ptz`

`onvif-ptz` uses the same configuration fields as `onvif-ptz-client`, but exposes the PTZ gRPC API.

```json
{
  "name": "ptz-1",
  "api": "rdk:component:ptz",
  "model": "viam:viamrtsp:onvif-ptz",
  "attributes": {
    "address": "http://192.168.1.10/onvif/device_service",
    "username": "admin",
    "password": "password",
    "profile_token": "MainStream"
  }
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

#### Get Configuration
```json
{"command": "get-configuration"}
```
Returns PTZ configuration for the current profile.

#### Get Configurations
```json
{"command": "get-configurations"}
```
Returns all available PTZ configurations.

#### Get Service Capabilities
```json
{"command": "get-service-capabilities"}
```
Returns PTZ service capabilities (EFlip, Reverse, MoveStatus, StatusPosition, etc.).

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
  "pan_tilt_space": "http://www.onvif.org/ver10/tptz/PanTiltSpaces/TranslationGenericSpace",
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
  "pan_tilt_space": "http://www.onvif.org/ver10/tptz/PanTiltSpaces/SphericalTranslationSpaceDegrees",
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
     - Normalized (`pan_tilt_space` generic space or omitted): -1.0 to 1.0 (pan/tilt/zoom).
     - Degrees (`pan_tilt_space` spherical degrees): -180° to 180° (pan), -90° to 90° (tilt). Zoom remains normalized.
   - Deprecated: `degrees` is still accepted by DoCommand for backwards compatibility, but prefer `pan_tilt_space`.
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
