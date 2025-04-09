# ONVIF PTZ Control Client

A command-line tool for controlling PTZ (Pan/Tilt/Zoom) cameras using the ONVIF protocol. This client supports various PTZ operations including continuous movement, relative movement, absolute positioning, and status queries.

## Building

```bash
# From the project root
go build -o ptz-client examples/ptz-control/client.go
```

## Configuration

The client requires a JSON configuration file with your camera details:

```json
{
  "ip": "192.168.1.100:80", 
  "username": "admin",
  "password": "yourpassword"
}
```

Save this as `camera.json` and use it with the `--config` flag. You'll also need to specify a profile token with the `--profile` flag:

```bash
./ptz-client --config camera.json --profile 001 get-status
```

## Available Commands

Get detailed help for all commands:
```bash
./ptz-client --help
```

Get help for a specific command:
```bash
./ptz-client <command> --help
```

### Main Commands:

1. **Get Profiles**
   ```bash
   ./ptz-client --config camera.json get-profiles
   ```
   Retrieves and displays the available media profiles (tokens) available on the ONVIF camera.
   Media profile tokens are necessary as an input to the actuation commands.

2. **Get Status**
   ```bash
   ./ptz-client --config camera.json --profile 001 get-status
   ```
   Shows current PTZ position, movement state, and other status information.

3. **Stop Movement**
   ```bash
   ./ptz-client --config camera.json --profile 001 stop
   ```
   Stops ongoing pan/tilt and zoom movements. Use `--pan-tilt=false` or `--zoom=false` to stop specific movements.

4. **Continuous Move**
   ```bash
   ./ptz-client --config camera.json --profile 001 continuous-move -x 0.5 -y 0.0 -z 0.0 --duration 2s
   ```
   Moves the camera continuously at specified speeds:
   - `-x`: Pan speed (-1.0 to 1.0)
   - `-y`: Tilt speed (-1.0 to 1.0)
   - `-z`: Zoom speed (-1.0 to 1.0)
   - `--duration`: Optional duration to move before stopping (e.g., "2s", "500ms")

5. **Relative Move**
   ```bash
   ./ptz-client --config camera.json --profile 001 relative-move -x 0.1 -y -0.2 -z 0.0
   ```
   Moves the camera by relative amounts:
   - Without `--degrees`: Uses normalized space (-1.0 to 1.0)
   - With `--degrees`: Uses degree space (Pan: -180 to 180, Tilt: -90 to 90)
   - Optional speed control with `--speed-x`, `--speed-y`, `--speed-z`

6. **Absolute Move**
   ```bash
   ./ptz-client --config camera.json --profile 001 absolute-move -x 0.0 -y 0.0 -z 0.5
   ```
   Moves the camera to absolute positions:
   - Without `--degrees`: Uses normalized space (-1.0 to 1.0)
   - With `--degrees`: Uses degree space (Pan: -180 to 180, Tilt: -90 to 90)
   - Zoom always uses normalized space (0.0 to 1.0)
   - Optional speed control with `--speed-x`, `--speed-y`, `--speed-z`

## Examples

1. Move camera right for 2 seconds:
   ```bash
   ./ptz-client --config camera.json --profile 001 continuous-move -x 0.5 -y 0 -z 0 --duration 2s
   ```

2. Move to home position (center):
   ```bash
   ./ptz-client --config camera.json --profile 001 absolute-move -x 0 -y 0 -z 0
   ```

3. Pan 45 degrees right using degree space:
   ```bash
   ./ptz-client --config camera.json --profile 001 relative-move -x 45 -y 0 -z 0 --degrees
   ```

4. Stop all movement:
   ```bash
   ./ptz-client --config camera.json --profile 001 stop
   ```

## Notes

- The profile token should be "000" or "001" typically. Check your camera's documentation or ONVIF device manager tool to find the correct token.
- Speed values range from -1.0 (full speed in negative direction) to 1.0 (full speed in positive direction).
- Default speed values:
  - Continuous move: 0.0 for all axes (no movement)
  - Relative move: 0.5 for all axes (half speed)
  - Absolute move: 0.5 for all axes (half speed)
- For pan/tilt:
  - Positive X moves right, negative X moves left
  - Positive Y moves up, negative Y moves down
- For zoom:
  - Positive values zoom in
  - Negative values zoom out
- Not all cameras support all movement types or spaces (normalized vs. degrees). Test capabilities with `get-status` first. 
