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

	"github.com/edaniels/zeroconf"
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

	hostNameWithLocal := fmt.Sprintf("%s.local", resp.SerialNumber)
	for idx := range rtspURLs {
		// We expect all of the `rtspURLs` to contain the `xaddr.Host` as a substring. But we'll
		// warn in case that assumption is not always true.
		if strings.Contains(rtspURLs[idx], xaddr.Host) {
			rtspURLs[idx] = strings.Replace(rtspURLs[idx], xaddr.Host, hostNameWithLocal, 1)
		} else {
			logger.Warnf("RTSP URL did not contain expected hostname. URL: %v HostName: %v", rtspURLs[idx], xaddr.Host)
		}
	}

	mdnsServer, err := zeroconf.RegisterProxy(
		resp.SerialNumber,    // Dan: As far as I can tell, just a name.
		"_rtsp._tcp",         // Dan: The mDNS "service" to register. Doesn't make a difference?
		"local",              // the domain
		8080,                 // The service's port is ignored here
		resp.SerialNumber,    // actual mDNS hostname, without the .local domain
		[]string{xaddr.Host}, // ip to use
		[]string{},           // txt fields, not needed
		nil,                  // resolve this name for requests from all network interfaces
		// RSDK-8205: logger.Desugar().Sugar() is necessary to massage a ZapCompatibleLogger into a
		// *zap.SugaredLogger to match zeroconf function signatures.
		logger.Desugar().Sugar(),
	)
	_ = mdnsServer
	if err != nil {
		logger.Warnf("Did not make hostname mapping for camera. Err:", err)
	}

	cameraInfo := CameraInfo{
		Host:            xaddr.Host,
		RTSPURLs:        rtspURLs,
		Manufacturer:    resp.Manufacturer,
		Model:           resp.Model,
		SerialNumber:    resp.SerialNumber,
		FirmwareVersion: resp.FirmwareVersion,
		HardwareID:      resp.HardwareID,
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
