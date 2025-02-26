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

	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
	"go.viam.com/utils"
)

// OnvifDevice is an interface to abstract device methods used in the code.
// Used instead of onvif.Device to allow for mocking in tests.
type OnvifDevice interface {
	GetDeviceInformation() (device.GetDeviceInformationResponse, error)
	GetProfiles() (device.GetProfilesResponse, error)
	GetStreamURI(profile onvif.Profile, creds device.Credentials) (*url.URL, error)
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
				logger.Warnf("failed to connect to ONVIF device %w", err)
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

// CameraInfo holds both the RTSP URLs and supplementary camera details.
type CameraInfo struct {
	Host            string   `json:"host"`
	RTSPURLs        []string `json:"rtsp_urls"`
	Manufacturer    string   `json:"manufacturer"`
	Model           string   `json:"model"`
	SerialNumber    string   `json:"serial_number"`
	FirmwareVersion string   `json:"firmware_version"`
	HardwareID      string   `json:"hardware_id"`

	deviceIP net.IP
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
	derivedHostname := fmt.Sprintf("%v.local", cleanedSerialNumber)

	// The mdns server expects a hostname without* the `.local` TLD suffix.
	if err := mdnsServer.Add(cleanedSerialNumber, cam.deviceIP); err != nil {
		logger.Debugf("Unable to make mdns mapping for device. Host: %v IP: %v SerialNumber: %v Err: %v",
			cam.Host, cam.deviceIP, cam.SerialNumber, err)
		return
	}

	wasIPFound := false
	// Replace the URLs in-place such that configs generated from these objects will point to the
	// logical dns hostname rather than a raw IP.
	for idx := range cam.RTSPURLs {
		if strings.Contains(cam.RTSPURLs[idx], cam.deviceIP.String()) {
			cam.RTSPURLs[idx] = strings.Replace(cam.RTSPURLs[idx], cam.deviceIP.String(), derivedHostname, 1)
			wasIPFound = true
		} else {
			logger.Debugf("RTSP URL did not contain expected hostname. URL: %v HostName: %v",
				cam.RTSPURLs[idx], cam.deviceIP.String())
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

// CameraInfoList is a struct containing a list of CameraInfo structs.
type CameraInfoList struct {
	Cameras []CameraInfo `json:"cameras"`
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
		dev, err := device.NewDevice(device.Params{
			Xaddr:    xaddr,
			Username: cred.User,
			Password: cred.Pass,
		}, logger)
		if err != nil {
			logger.Debugf("Failed to get camera capabilities from %s: %v", xaddr, err)
			continue
		}

		cameraInfo, err := GetCameraInfo(dev, xaddr, cred, logger)
		if err != nil {
			logger.Warnf("Failed to get camera info from %s: %v", xaddr, err)
			continue
		}
		// once we have addeed a camera info break
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
func GetCameraInfo(dev OnvifDevice, xaddr *url.URL, creds device.Credentials, logger logging.Logger) (CameraInfo, error) {
	var zero CameraInfo
	// Fetch device information (manufacturer, serial number, etc.)
	resp, err := dev.GetDeviceInformation()
	if err != nil {
		return zero, fmt.Errorf("failed to read device information response body: %w", err)
	}
	logger.Debugf("ip: %s GetCapabilities: DeviceInfo: %#v", xaddr, dev)

	// Call the ONVIF Media service to get the available media profiles using the same device instance
	rtspURLs, err := GetRTSPStreamURIsFromProfiles(dev, creds, logger)
	if err != nil {
		return zero, fmt.Errorf("failed to get RTSP URLs: %w", err)
	}

	cameraInfo := CameraInfo{
		Host:            xaddr.Host,
		RTSPURLs:        rtspURLs,
		Manufacturer:    resp.Manufacturer,
		Model:           resp.Model,
		SerialNumber:    resp.SerialNumber,
		FirmwareVersion: resp.FirmwareVersion,
		HardwareID:      resp.HardwareID,

		// Will be nil if there's an error.
		deviceIP: net.ParseIP(xaddr.Host),
	}

	return cameraInfo, nil
}

// GetRTSPStreamURIsFromProfiles uses the ONVIF Media service to get the RTSP stream URLs for all available profiles.
func GetRTSPStreamURIsFromProfiles(dev OnvifDevice, creds device.Credentials, logger logging.Logger) ([]string, error) {
	resp, err := dev.GetProfiles()
	if err != nil {
		return nil, err
	}

	// Resultant slice of RTSP URIs
	var rtspUris []string

	// Iterate over all profiles and get the RTSP stream URI for each one
	for _, profile := range resp.Profiles {
		uri, err := dev.GetStreamURI(profile, creds)
		if err != nil {
			logger.Warn(err.Error())
			continue
		}

		rtspUris = append(rtspUris, uri.String())
	}

	return rtspUris, nil
}
