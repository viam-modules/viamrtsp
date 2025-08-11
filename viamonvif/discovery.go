// Package viamonvif provides ONVIF integration to the viamrtsp module
package viamonvif

import (
	"context"
	"fmt"
	"maps"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/ptzclient"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
	"go.viam.com/utils"
)

// OnvifDevice is an interface to abstract device methods used in the code.
// Used instead of onvif.Device to allow for mocking in tests.
type OnvifDevice interface {
	GetXaddr() *url.URL
	GetDeviceInformation(ctx context.Context) (device.GetDeviceInformationResponse, error)
	GetProfiles(ctx context.Context) (device.GetProfilesResponse, error)
	GetStreamURI(ctx context.Context, token onvif.ReferenceToken, creds device.Credentials) (*url.URL, error)
	GetSnapshotURI(ctx context.Context, token onvif.ReferenceToken, creds device.Credentials) (*url.URL, error)
}

// DiscoverCameras performs WS-Discovery
// then uses ONVIF queries to get available RTSP addresses and supplementary info.
func DiscoverCameras(
	ctx context.Context,
	creds []device.Credentials,
	manualXAddrs []*url.URL,
	logger logging.Logger,
) (*CameraInfoList, error) {
	var ret []CameraInfo
	discoveredXAddrs, err := discoverOnAllInterfaces(ctx, manualXAddrs, logger)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	ch := make(chan CameraInfo, len(discoveredXAddrs))
	wg.Add(len(discoveredXAddrs))
	for _, xaddr := range discoveredXAddrs {
		utils.ManagedGo(func() {
			cameraInfo, err := DiscoverCameraInfo(ctx, xaddr, creds, logger)
			if err != nil {
				logger.Warnf("failed to connect to ONVIF device %s", err)
				return
			}
			ch <- cameraInfo
		}, wg.Done)
	}
	wg.Wait()
	close(ch)
	for cameraInfo := range ch {
		ret = append(ret, cameraInfo)
	}
	return &CameraInfoList{Cameras: ret}, nil
}

func discoverOnAllInterfaces(ctx context.Context, manualXAddrs []*url.URL, logger logging.Logger) ([]*url.URL, error) {
	logger.Debug("WS-Discovery start")
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve network interfaces: %w", err)
	}

	var wg sync.WaitGroup
	ch := make(chan []*url.URL, len(ifaces))
	wg.Add(len(ifaces))
	for _, iface := range ifaces {
		utils.ManagedGo(func() {
			xaddrs, err := WSDiscovery(ctx, logger, iface)
			if err != nil {
				logger.Debugf("WS-Discovery skipping interface %s: due to error from SendProbe: %w", iface.Name, err)
				return
			}
			ch <- xaddrs
		}, wg.Done)
	}
	logger.Debug("WS-Discovery waiting for all interfaces to return results")
	wg.Wait()

	close(ch)
	logger.Debug("WS-Discovery all interfaces have returned results")

	discovered := map[string]*url.URL{}
	for _, xaddr := range manualXAddrs {
		discovered[xaddr.Host] = xaddr
	}
	for xaddrs := range ch {
		for _, xaddr := range xaddrs {
			discovered[xaddr.Host] = xaddr
		}
	}
	logger.Debug("WS-Discovery complete")
	return slices.Collect(maps.Values(discovered)), nil
}

// MediaInfo holds detailed information about a camera's media capabilities, including
// the stream URI, snapshot URI, frame rate, resolution, and codec.
type MediaInfo struct {
	StreamURI   string              `json:"stream_uri"`
	SnapshotURI string              `json:"snapshot_uri"`
	FrameRate   int                 `json:"frame_rate"`
	Resolution  viamrtsp.Resolution `json:"resolution"`
	Codec       string              `json:"codec"`
}

type PTZInfo struct {
	Address      string                           `json:"address"`
	Username     string                           `json:"username"`
	Password     string                           `json:"password"`
	ProfileToken string                           `json:"profile_token"`
	PTZNodeToken string                           `json:"ptz_node_token"`
	Capabilities ptzclient.PTZCaps                `json:"capabilities"`
	Movements    map[string]ptzclient.PTZMovement `json:"movements"`
}

// CameraInfo holds both the RTSP URLs and supplementary camera details.
type CameraInfo struct {
	Host            string      `json:"host"`
	MediaEndpoints  []MediaInfo `json:"media_endpoints"`
	PTZEndpoints    []PTZInfo   `json:"ptz_endpoints,omitempty"` // PTZ endpoints are optional
	Manufacturer    string      `json:"manufacturer"`
	Model           string      `json:"model"`
	SerialNumber    string      `json:"serial_number"`
	FirmwareVersion string      `json:"firmware_version"`
	HardwareID      string      `json:"hardware_id"`

	deviceIP net.IP
	mdnsName string
}

// regex to remove non alpha numerics.
var reg = regexp.MustCompile("[^a-zA-Z0-9]+")

// Name creates generates a name for the camera based on discovered information about the camera.
func (cam *CameraInfo) Name(urlNum int) string {
	stripManufacturer := reg.ReplaceAllString(cam.Manufacturer, "")
	stripModel := reg.ReplaceAllString(cam.Model, "")
	stripSerial := reg.ReplaceAllString(cam.SerialNumber, "")
	return fmt.Sprintf("%s-%s-%s-url%v", stripManufacturer, stripModel, stripSerial, urlNum)
}

func (cam *CameraInfo) tryMDNS(mdnsServer *mdnsServer, logger logging.Logger) {
	// Sanity check the input required to make an mdns mapping.
	if cam.deviceIP == nil || cam.SerialNumber == "" {
		logger.Debugf("Not making mdns mapping for device. Host: %v IP: %v SerialNumber: %v",
			cam.Host, cam.deviceIP, cam.SerialNumber)
		return
	}

	// Clean the serial number to be dns compatible.
	cleanedSerialNumber := strToHostName(cam.SerialNumber)
	// Generate full .local hostname for the device.
	cam.mdnsName = fmt.Sprintf("%v.local", cleanedSerialNumber)

	// The mdns server expects a hostname without* the `.local` TLD suffix.
	if err := mdnsServer.Add(cleanedSerialNumber, cam.deviceIP); err != nil {
		logger.Debugf("Unable to make mdns mapping for device. Host: %v IP: %v SerialNumber: %v Err: %v",
			cam.Host, cam.deviceIP, cam.SerialNumber, err)
		return
	}

	wasIPFound := false
	// Replace the URLs in-place such that configs generated from these objects will point to the
	// logical dns hostname rather than a raw IP.
	for idx := range cam.MediaEndpoints {
		if strings.Contains(cam.MediaEndpoints[idx].StreamURI, cam.deviceIP.String()) {
			cam.MediaEndpoints[idx].StreamURI = strings.Replace(cam.MediaEndpoints[idx].StreamURI, cam.deviceIP.String(), cam.mdnsName, 1)
			wasIPFound = true
		} else {
			logger.Debugf("RTSP URL did not contain expected hostname. URL: %v HostName: %v",
				cam.MediaEndpoints[idx].StreamURI, cam.deviceIP.String())
		}
	}

	if !wasIPFound {
		// If for some reason, the `deviceIP`/`xaddr.Host` IP was not found in any of the RTSP urls,
		// stop serving mdns requests for that serial number.
		//
		// We have* observed a device returning RTSP urls with multiple IPs, but do not yet know of
		// a case where none of the URLs contained an IP that matches where the response came from.
		mdnsServer.Remove(cleanedSerialNumber)
	}
}

func (cam *CameraInfo) urlDependsOnMDNS(idx int) bool {
	return strings.Contains(cam.MediaEndpoints[idx].StreamURI, cam.mdnsName)
}

// CameraInfoList is a struct containing a list of CameraInfo structs.
type CameraInfoList struct {
	Cameras []CameraInfo `json:"cameras"`
	PTZs    []PTZInfo    `json:"ptzs,omitempty"` // PTZs are optional
}

// DiscoverCameraInfo discovers camera information on a given uri
// using onvif
// the xaddr should be the URL of the onvif camera's device service URI.
func DiscoverCameraInfo(
	ctx context.Context,
	xaddr *url.URL,
	creds []device.Credentials,
	logger logging.Logger,
) (CameraInfo, error) {
	logger.Debugf("Connecting to ONVIF device with URL: %s", xaddr)
	var zero CameraInfo
	for _, cred := range creds {
		if ctx.Err() != nil {
			return zero, fmt.Errorf("context canceled while connecting to ONVIF device: %s", xaddr)
		}
		// This calls GetCapabilities
		dev, err := device.NewDevice(ctx, device.Params{
			Xaddr:                    xaddr,
			Username:                 cred.User,
			Password:                 cred.Pass,
			SkipLocalTLSVerification: true,
		}, logger)
		if err != nil {
			logger.Debugf("Failed to get camera capabilities from %s: %v", xaddr, err)
			continue
		}

		cameraInfo, err := GetCameraInfo(ctx, dev, xaddr, cred, logger)
		if err != nil {
			logger.Warnf("Failed to get camera info from %s: %v", xaddr, err)
			continue
		}
		// once we have added a camera info break
		return cameraInfo, nil
	}
	return zero, fmt.Errorf("no credentials matched IP %s", xaddr)
}

var consecutiveDashesRegexp = regexp.MustCompile(`\-\-+`)

func strToHostName(inp string) string {
	// Turn strings into valid hostnames. We perform the following steps:
	// - For each character:
	//   - Any alphanumeric is copied over. Casing is preserved.
	//   - Any non-alphanumeric is turned into a dash.
	// - As a final pass, replace any string of dashes with a single dash.
	sb := strings.Builder{}
	for _, ch := range inp {
		if unicode.IsLetter(ch) || unicode.IsNumber(ch) {
			sb.WriteRune(ch)
		} else {
			sb.WriteRune('-')
		}
	}

	return consecutiveDashesRegexp.ReplaceAllLiteralString(sb.String(), "-")
}

// GetCameraInfo uses the ONVIF Media service to get the RTSP stream URLs and camera details.
func GetCameraInfo(
	ctx context.Context,
	dev OnvifDevice,
	xaddr *url.URL,
	creds device.Credentials,
	logger logging.Logger,
) (CameraInfo, error) {
	var zero CameraInfo
	// Fetch device information (manufacturer, serial number, etc.)
	resp, err := dev.GetDeviceInformation(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to read device information response body: %w", err)
	}
	logger.Debugf("ip: %s GetCapabilities: DeviceInfo: %#v", xaddr, dev)

	// Call the ONVIF Media service to get the available media profiles using the same device instance
	mes, pes, err := GetMediaInfoFromProfiles(ctx, dev, creds, logger)
	if err != nil {
		return zero, fmt.Errorf("failed to get stream info: %w", err)
	}

	cameraInfo := CameraInfo{
		Host:            xaddr.Host,
		MediaEndpoints:  mes,
		Manufacturer:    resp.Manufacturer,
		Model:           resp.Model,
		SerialNumber:    resp.SerialNumber,
		FirmwareVersion: resp.FirmwareVersion,
		HardwareID:      resp.HardwareID,
		PTZEndpoints:    pes,

		// Will be nil if there's an error.
		deviceIP: net.ParseIP(xaddr.Host),
	}

	return cameraInfo, nil
}

func ExtractInfoFromProfiles(
	ctx context.Context,
	dev OnvifDevice,
	profileToken onvif.ReferenceToken,
	logger logging.Logger,
) ([]MediaInfo, []PTZInfo, error) {
	// Get the media info from the profiles
	return nil, nil, nil
}

// GetMediaInfoFromProfiles uses the ONVIF Media service to get the RTSP stream URLs
// and Snapshot URIs for all available profiles.
func GetMediaInfoFromProfiles(
	ctx context.Context,
	dev OnvifDevice,
	creds device.Credentials,
	logger logging.Logger,
) ([]MediaInfo, []PTZInfo, error) {
	resp, err := dev.GetProfiles(ctx)
	if err != nil {
		return nil, nil, err
	}

	var mes []MediaInfo
	var ptzInfos []PTZInfo
	// Iterate over all profiles and get the RTSP stream and snapshot URI for each one
	for _, profile := range resp.Profiles {
		logger.Debugf("Looking up media info for profile: %s (token: %s)", profile.Name, profile.Token)
		streamURI, err := dev.GetStreamURI(ctx, profile.Token, creds)
		if err != nil {
			logger.Warn(err.Error())
			continue
		}

		snapshotURIString := ""
		snapshotURI, err := dev.GetSnapshotURI(ctx, profile.Token, creds)
		if err != nil {
			logger.Warnf("Unable to determine onvif snapshot URI  profile %s: %s, err: %s", profile.Name, streamURI.String(), err.Error())
		} else {
			snapshotURIString = snapshotURI.String()
		}

		// Always add the MediaInfo if we get a stream URI, even if the snapshot URI fails.
		mes = append(mes, MediaInfo{
			StreamURI:   streamURI.String(),
			SnapshotURI: snapshotURIString,
			FrameRate:   int(profile.VideoEncoderConfiguration.RateControl.FrameRateLimit),
			Resolution: viamrtsp.Resolution{
				Width:  int(profile.VideoEncoderConfiguration.Resolution.Width),
				Height: int(profile.VideoEncoderConfiguration.Resolution.Height),
			},
			Codec: string(profile.VideoEncoderConfiguration.Encoding),
		})

		// Here we will also fech ptzInfo if available.
		// Check if the profile has PTZ capabilities
		if profile.PTZConfiguration.Token == "" {
			logger.Debugf("Profile %s does not have PTZ capabilities, %s", profile.Name, streamURI.String())
			continue
		}

		logger.Debugf("Profile %s has PTZ capabilities, %s", profile.Name, streamURI.String())
		ptzCfg := profile.PTZConfiguration
		nodeToken := string(ptzCfg.NodeToken)
		movements := map[string]ptzclient.PTZMovement{}
		absPanTilt := ptzCfg.PanTiltLimits.Range
		absZoom := ptzCfg.ZoomLimits.Range
		if absPanTilt.URI != "" || absZoom.URI != "" {
			movements["continuous"] = ptzclient.PTZMovement{
				PanTilt: ptzclient.PanTiltSpace{
					XMin:  absPanTilt.XRange.Min,
					XMax:  absPanTilt.XRange.Max,
					YMin:  absPanTilt.YRange.Min,
					YMax:  absPanTilt.YRange.Max,
					Space: lastURISegment(string(absPanTilt.URI)),
				},
				Zoom: ptzclient.ZoomSpace{
					XMin:  absZoom.XRange.Min,
					XMax:  absZoom.XRange.Max,
					Space: lastURISegment(string(absZoom.URI)),
				},
			}
		}

		ptzInfos = append(ptzInfos, PTZInfo{
			Address:      dev.GetXaddr().Host,
			Username:     creds.User,
			Password:     creds.Pass,
			ProfileToken: string(profile.Token),
			PTZNodeToken: nodeToken,
			Capabilities: ptzclient.PTZCaps{}, // TODO(seanp): Do we want to fill caps in?
			Movements:    movements,
		})

		logger.Debugf("PTZ Info for profile %s: %+v", profile.Name, ptzInfos[len(ptzInfos)-1])
	}

	return mes, ptzInfos, nil
}

func lastURISegment(uri string) string {
	parts := strings.Split(uri, "/")
	return parts[len(parts)-1]
}
