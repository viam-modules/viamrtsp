# PTZ model

This model implements the [`"rdk:component:generic"` API](https://docs.viam.com/components/generic/) for controlling ONVIF-compliant PTZ (Pan-Tilt-Zoom) cameras. The generic component supports core PTZ operations through the DoCommand method.

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
  "model": "viam:viamrtsp:onvif-ptz-client",
  "attributes": {
    "address": "192.168.1.100:80",
    "username": "admin",
    "password": "yourpassword",
    "profile_token": "001"
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
  "zoom_speed": 0.0
}
```
Continuous motion at specified speeds (-1.0 to 1.0).

#### Relative Move
```json
{
  "command": "relative-move",
  "pan": 0.1,
  "tilt": -0.05,
  "zoom": 0.0,
  "degrees": true
}
```
Relative position move with optional degree space conversion.

#### Absolute Move
```json
{
  "command": "absolute-move", 
  "pan": 0.0,
  "tilt": 0.0,
  "zoom": 0.5,
  "degrees": false
}
```
Absolute position move in normalized or degree space.

## Notes

1. **Disclaimer**: This model was made in order to fully integrate with one specific camera. I tried to generalize it to all PTZ cameras, but your mileage may vary.
1. **Profile Discovery**: Use `get-profiles` command to discover valid profile tokens
2. **Coordinate Spaces**:
   - Normalized: -1.0 to 1.0 (pan/tilt), 0.0-1.0 (zoom)
   - Degrees: -180° to 180° (pan), -90° to 90° (tilt)
3. **Movement Speeds**: 
   - Continuous: -1.0 (full reverse) to 1.0 (full forward)
   - Relative/Absolute: 0.5 default speed (override with `speed_*` params)

## Troubleshooting

**ONVIF Compliance**: Ensure camera supports ONVIF Profile S with PTZ services. Test with ONVIF Device Manager first.

**Authentication Issues**: Some cameras require ONVIF authentication separate from web interface credentials.

**Profile Configuration**: If commands fail with empty profile token:
1. Run `get-profiles` command
2. Copy valid token to configuration
3. Restart component
