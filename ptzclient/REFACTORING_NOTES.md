# PTZ Client Refactoring Notes

## Overview

The current `client.go` (~650 lines) can be reduced by ~20% (~135 lines) through extracting repeated patterns into helpers.

---

## 1. Generic `callMethod` Helper (~80 lines saved)

Every handler repeats this pattern:
```go
res, err := s.dev.CallMethod(req, s.logger)
defer res.Body.Close()
bodyBytes, err := io.ReadAll(res.Body)
s.logger.Debugf("... raw response body: %s", ...)
var envelope SomeType
xml.Unmarshal(bodyBytes, &envelope)
```

**Solution** - Add to `do_command_helpers.go`:
```go
func (s *onvifPtzClient) callMethod(req interface{}, result interface{}) ([]byte, error) {
    res, err := s.dev.CallMethod(req, s.logger)
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()
    bodyBytes, err := io.ReadAll(res.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    if result != nil {
        if err := xml.Unmarshal(bodyBytes, result); err != nil {
            s.logger.Warnf("Unmarshal failed. Raw XML:\n%s", string(bodyBytes))
            return nil, fmt.Errorf("failed to unmarshal: %w", err)
        }
    }
    return bodyBytes, nil
}
```

**Before** (repeated 8+ times):
```go
func (s *onvifPtzClient) handleGetStatus() (map[string]interface{}, error) {
    // ... profile token check ...
    req := ptz.GetStatus{ProfileToken: profileToken}
    s.logger.Debugf("Sending GetStatus request for profile: %s", profileToken)

    res, err := s.dev.CallMethod(req, s.logger)
    if err != nil {
        return nil, fmt.Errorf("failed to call GetStatus: %w", err)
    }
    defer res.Body.Close()

    bodyBytes, err := io.ReadAll(res.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read GetStatus response body: %w", err)
    }
    s.logger.Debugf("GetStatus raw response body: %s", string(bodyBytes))

    var statusEnvelope CustomGetStatusEnvelope
    err = xml.Unmarshal(bodyBytes, &statusEnvelope)
    if err != nil {
        s.logger.Warnf("Failed to unmarshal GetStatus response. Raw XML:\n%s", string(bodyBytes))
        return nil, fmt.Errorf("failed to unmarshal GetStatus response: %w", err)
    }
    // ... rest of handler ...
}
```

**After**:
```go
func (s *onvifPtzClient) handleGetStatus() (map[string]interface{}, error) {
    profileToken, err := s.profileToken()
    if err != nil {
        return nil, err
    }

    var envelope CustomGetStatusEnvelope
    if _, err := s.callMethod(ptz.GetStatus{ProfileToken: profileToken}, &envelope); err != nil {
        return nil, fmt.Errorf("GetStatus failed: %w", err)
    }
    // ... rest of handler ...
}
```

---

## 2. Profile Token Helper (~20 lines saved)

Duplicated 6 times:
```go
if s.cfg.ProfileToken == "" {
    return nil, errors.New("profile_token is not configured for this component")
}
profileToken := onvifxsd.ReferenceToken(s.cfg.ProfileToken)
```

**Solution** - Add to `do_command_helpers.go`:
```go
func (s *onvifPtzClient) profileToken() (onvifxsd.ReferenceToken, error) {
    if s.cfg.ProfileToken == "" {
        return "", errors.New("profile_token is not configured")
    }
    return onvifxsd.ReferenceToken(s.cfg.ProfileToken), nil
}
```

---

## 3. Speed Parameter Extraction (~25 lines saved)

Both `handleRelativeMove` and `handleAbsoluteMove` have ~15 lines of identical logic:
```go
_, panSpeedProvided := cmd["pan_speed"]
_, tiltSpeedProvided := cmd["tilt_speed"]
_, zoomSpeedProvided := cmd["zoom_speed"]
sendSpeed := panSpeedProvided || tiltSpeedProvided || zoomSpeedProvided

var panSpeed, tiltSpeed, zoomSpeed float64
if sendSpeed {
    panSpeed = getOptionalFloat64(cmd, "pan_speed", defaultPanSpeed)
    tiltSpeed = getOptionalFloat64(cmd, "tilt_speed", defaultTiltSpeed)
    zoomSpeed = getOptionalFloat64(cmd, "zoom_speed", defaultZoomSpeed)
}
```

**Solution** - Add to `do_command_helpers.go`:
```go
type speeds struct {
    pan, tilt, zoom float64
    provided        bool
}

func extractSpeeds(cmd map[string]interface{}) speeds {
    _, p := cmd["pan_speed"]
    _, t := cmd["tilt_speed"]
    _, z := cmd["zoom_speed"]
    if !p && !t && !z {
        return speeds{}
    }
    return speeds{
        pan:      getOptionalFloat64(cmd, "pan_speed", defaultPanSpeed),
        tilt:     getOptionalFloat64(cmd, "tilt_speed", defaultTiltSpeed),
        zoom:     getOptionalFloat64(cmd, "zoom_speed", defaultZoomSpeed),
        provided: true,
    }
}
```

**Usage**:
```go
spd := extractSpeeds(cmd)
if spd.provided {
    // use spd.pan, spd.tilt, spd.zoom
}
```

---

## 4. Speed Validation Helper (~10 lines saved)

Duplicated validation logic:
```go
if panSpeed < 0.0 || panSpeed > 1.0 || tiltSpeed < 0.0 || tiltSpeed > 1.0 || zoomSpeed < 0.0 || zoomSpeed > 1.0 {
    return nil, errors.New("speed values must be between 0.0 and 1.0")
}
```

**Solution** - Add to `do_command_helpers.go`:
```go
func validateSpeeds(p, t, z float64, allowNegative bool) error {
    min := 0.0
    if allowNegative {
        min = -1.0
    }
    if p < min || p > 1.0 || t < min || t > 1.0 || z < min || z > 1.0 {
        return fmt.Errorf("speeds must be between %.1f and 1.0", min)
    }
    return nil
}
```

---

## Summary

| Refactoring | Lines Saved | Files Affected |
|-------------|-------------|----------------|
| Generic `callMethod` helper | ~80 | client.go, do_command_helpers.go |
| `profileToken()` helper | ~20 | client.go, do_command_helpers.go |
| Speed extraction helper | ~25 | client.go, do_command_helpers.go |
| Speed validation helper | ~10 | client.go, do_command_helpers.go |
| **Total** | **~135 (~20%)** | |

Final line count: ~650 → ~500 lines

---

## 5. Potential Type Consolidation with `viamonvif`

### Current State

There's overlap between `ptzclient/onvif_types.go` and `viamonvif/xsd/onvif/onvif.go`:

| ptzclient/onvif_types.go | viamonvif/xsd/onvif/onvif.go |
|--------------------------|------------------------------|
| `CustomVector2D` | `Vector2D` |
| `CustomVector1D` | `Vector1D` |
| `CustomPTZStatus` | `PTZStatus` |
| Space URI constants | (none - could be shared) |

### Why Custom Types Exist

The "Custom" types in ptzclient have **specific XML namespace bindings** for SOAP response parsing:
```go
type CustomGetStatusEnvelope struct {
    XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
    Body    CustomGetStatusBody `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
}
```

The viamonvif types use different namespace prefixes (e.g., `xml:"onvif:PanTilt"`).

### What Could Be Consolidated

**Space URI constants** (currently in ptzclient) could move to a shared location:
```go
// These could live in viamonvif/xsd/onvif/ or a shared constants package
const (
    AbsolutePanTiltPositionGenericSpace     = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/PositionGenericSpace"
    AbsoluteZoomPositionGenericSpace        = "http://www.onvif.org/ver10/tptz/ZoomSpaces/PositionGenericSpace"
    RelativePanTiltTranslationGenericSpace  = "http://www.onvif.org/ver10/tptz/PanTiltSpaces/TranslationGenericSpace"
    // ... etc
)
```

### Recommendation

- **Keep Custom types separate** - they serve a specific parsing purpose
- **Move Space URI constants** to `viamonvif/xsd/onvif/` for shared use
- Low priority - the current duplication is minimal and serves different purposes

---

## 6. Removing `sean-onvif` External Dependency

### Current State

The repo uses `github.com/hexbabe/sean-onvif` in two places:

**`ptzclient/client.go`** - Heavy usage:
```go
import (
    onvif "github.com/hexbabe/sean-onvif"
    "github.com/hexbabe/sean-onvif/media"
    "github.com/hexbabe/sean-onvif/ptz"
    "github.com/hexbabe/sean-onvif/xsd"
    onvifxsd "github.com/hexbabe/sean-onvif/xsd/onvif"
)
```

**`viamonvif/device/device.go`** - Minimal usage:
```go
import (
    "github.com/hexbabe/sean-onvif/ptz"  // only for ptz.GetNodes{}
)
```

### What's Used from `sean-onvif`

| Category | Types Used |
|----------|------------|
| Device | `onvif.Device`, `onvif.DeviceParams`, `onvif.NewDevice()` |
| Device Method | `dev.CallMethod()` → returns `*http.Response` |
| PTZ Requests | `ptz.Stop`, `ptz.ContinuousMove`, `ptz.RelativeMove`, `ptz.AbsoluteMove`, `ptz.GetStatus`, `ptz.GetConfiguration`, `ptz.GetConfigurations`, `ptz.GetServiceCapabilities`, `ptz.GetNodes` |
| PTZ Responses | `ptz.GetConfigurationResponse`, `ptz.GetConfigurationsResponse`, `ptz.GetServiceCapabilitiesResponse` |
| Media Requests | `media.GetProfiles` |
| XSD Primitives | `xsd.Boolean`, `xsd.Duration` |
| ONVIF Types | `onvifxsd.ReferenceToken`, `onvifxsd.PTZSpeed`, `onvifxsd.Vector2D`, `onvifxsd.Vector1D`, `onvifxsd.PTZVector` |

### What `viamonvif` Already Has

- `device.Device` with `callOnvifServiceMethod()` (returns `[]byte`, not `*http.Response`)
- `device.GetProfiles` request type (local version)
- Types in `xsd/onvif/onvif.go`: `Vector2D`, `Vector1D`, `PTZSpeed`, `PTZStatus`, `PTZVector`, `ReferenceToken`
- Types in `xsd/built_in.go`: likely has `Boolean`, `Duration` equivalents

### Work Required to Remove `sean-onvif`

#### 1. Add PTZ Request Types to `viamonvif`

Create `viamonvif/ptz/requests.go`:
```go
package ptz

import "github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"

type Stop struct {
    XMLName      string               `xml:"tptz:Stop"`
    ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
    PanTilt      bool                 `xml:"tptz:PanTilt,omitempty"`
    Zoom         bool                 `xml:"tptz:Zoom,omitempty"`
}

type ContinuousMove struct {
    XMLName      string               `xml:"tptz:ContinuousMove"`
    ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
    Velocity     onvif.PTZSpeed       `xml:"tptz:Velocity"`
    Timeout      string               `xml:"tptz:Timeout,omitempty"`
}

type RelativeMove struct {
    XMLName      string               `xml:"tptz:RelativeMove"`
    ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
    Translation  onvif.PTZVector      `xml:"tptz:Translation"`
    Speed        onvif.PTZSpeed       `xml:"tptz:Speed,omitempty"`
}

type AbsoluteMove struct {
    XMLName      string               `xml:"tptz:AbsoluteMove"`
    ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
    Position     onvif.PTZVector      `xml:"tptz:Position"`
    Speed        onvif.PTZSpeed       `xml:"tptz:Speed,omitempty"`
}

type GetStatus struct {
    XMLName      string               `xml:"tptz:GetStatus"`
    ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type GetConfiguration struct {
    XMLName      string               `xml:"tptz:GetConfiguration"`
    ProfileToken onvif.ReferenceToken `xml:"tptz:ProfileToken"`
}

type GetConfigurations struct {
    XMLName string `xml:"tptz:GetConfigurations"`
}

type GetServiceCapabilities struct {
    XMLName string `xml:"tptz:GetServiceCapabilities"`
}

type GetNodes struct {
    XMLName string `xml:"tptz:GetNodes"`
}
```

#### 2. Add PTZ Response Types

Create `viamonvif/ptz/responses.go`:
```go
package ptz

type GetConfigurationResponse struct {
    PTZConfiguration PTZConfiguration `xml:"PTZConfiguration"`
}

type GetConfigurationsResponse struct {
    PTZConfiguration []PTZConfiguration `xml:"PTZConfiguration"`
}

type GetServiceCapabilitiesResponse struct {
    Capabilities Capabilities `xml:"Capabilities"`
}

// ... define PTZConfiguration, Capabilities structs
```

#### 3. Add XSD Primitives (if missing)

Check `viamonvif/xsd/built_in.go` for `Boolean` and `Duration` types.

#### 4. Adapt `ptzclient` to Use Local Device

**Option A**: Add `CallMethod` to `viamonvif/device.Device` that returns `*http.Response`

**Option B**: Refactor `ptzclient` to use `callOnvifServiceMethod()` which returns `[]byte`
- This is cleaner and avoids the response body handling boilerplate
- Aligns with the refactoring in Section 1 (generic `callMethod` helper)

#### 5. Update Imports in `ptzclient/client.go`

```go
// Before
import (
    onvif "github.com/hexbabe/sean-onvif"
    "github.com/hexbabe/sean-onvif/ptz"
    onvifxsd "github.com/hexbabe/sean-onvif/xsd/onvif"
)

// After
import (
    "github.com/viam-modules/viamrtsp/viamonvif/device"
    "github.com/viam-modules/viamrtsp/viamonvif/ptz"
    "github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
)
```

### Effort Estimate

| Task | Complexity |
|------|------------|
| Add PTZ request types | Low - ~50 lines |
| Add PTZ response types | Medium - ~100 lines |
| Verify/add XSD primitives | Low - ~20 lines |
| Adapt ptzclient to local device | Medium - touch ~15 call sites |
| Testing | Medium - ensure all PTZ commands work |

**Total**: Moderate effort, ~1-2 days of work

### Benefits

- Remove external dependency
- Single source of truth for ONVIF types
- Easier to maintain and debug
- Better control over SOAP request/response handling
