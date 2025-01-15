// Package viamonvif provides ONVIF integration to the viamrtsp module
package viamonvif

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	onvifxsd "github.com/use-go/onvif/xsd/onvif"
	"go.viam.com/rdk/logging"
)

const (
	streamTypeRTPUnicast = "RTP-Unicast"
	streamSetupProtocol  = "RTSP"
)

type Credentials struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

// DiscoverCameras performs WS-Discovery using the use-go/onvif discovery utility,
// then uses ONVIF queries to get available RTSP addresses and supplementary info.
func DiscoverCameras(creds []Credentials, manualXAddrs []*url.URL, logger logging.Logger) (*CameraInfoList, error) {
	var discoveredCameras []CameraInfo
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve network interfaces: %w", err)
	}

	discovered := map[string]*url.URL{}
	for _, xaddr := range manualXAddrs {
		discovered[xaddr.Host] = xaddr
	}
	for _, iface := range ifaces {
		if !ValidWSDiscoveryInterface(iface) {
			logger.Debugf("WS-Discovery skipping interface %s: does not meet WS-Discovery requirements", iface.Name)
			continue
		}
		xaddrs, err := WSDiscovery(ctx, logger, iface)
		if err != nil {
			logger.Debugf("WS-Discovery skipping interface %s: due to error from SendProbe: %w", iface.Name, err)
			continue
		}
		for _, xaddr := range xaddrs {
			discovered[xaddr.Host] = xaddr
		}
	}

	for _, xaddr := range discovered {
		logger.Debugf("Connecting to ONVIF device with URL: %s", xaddr.Host)
		cameraInfo, err := DiscoverCamerasOnXAddr(ctx, xaddr, creds, logger)
		if err != nil {
			logger.Warnf("failed to connect to ONVIF device %v", err)
			continue
		}
		discoveredCameras = append(discoveredCameras, cameraInfo)
	}
	return &CameraInfoList{Cameras: discoveredCameras}, nil
}

// OnvifDevice is an interface to abstract device methods used in the code.
// Used instead of onvif.Device to allow for mocking in tests.
type OnvifDevice interface {
	CallMethod(request interface{}) (*http.Response, error)
}

// GetProfilesResponse is the schema the GetProfiles response is formatted in.
type GetProfilesResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetProfilesResponse struct {
			Profiles []onvifxsd.Profile `xml:"Profiles"`
		} `xml:"GetProfilesResponse"`
	} `xml:"Body"`
}

// getStreamURIResponse is the schema the GetStreamUri response is formatted in.
type getStreamURIResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetStreamURIResponse struct {
			MediaURI onvifxsd.MediaUri `xml:"MediaUri"`
		} `xml:"GetStreamUriResponse"`
	} `xml:"Body"`
}

// DeviceInfo is the schema the GetDeviceInformation response is formatted in.
type DeviceInfo struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetDeviceInformationResponse struct {
			Manufacturer    string `xml:"Manufacturer"`
			Model           string `xml:"Model"`
			FirmwareVersion string `xml:"FirmwareVersion"`
			SerialNumber    string `xml:"SerialNumber"`
			HardwareId      string `xml:"HardwareId"`
		} `xml:"GetDeviceInformationResponse"`
	} `xml:"Body"`
}

// CameraInfo holds both the RTSP URLs and supplementary camera details.
type CameraInfo struct {
	WSDiscoveryXAddr string   `json:"ws_discovery_x_addr"`
	RTSPURLs         []string `json:"rtsp_urls"`
	Manufacturer     string   `json:"manufacturer"`
	Model            string   `json:"model"`
	SerialNumber     string   `json:"serial_number"`
	FirmwareVersion  string   `json:"firmware_version"`
	HardwareId       string   `json:"hardware_id"`
}

// CameraInfoList is a struct containing a list of CameraInfo structs.
type CameraInfoList struct {
	Cameras []CameraInfo `json:"cameras"`
}

func WSDiscovery(ctx context.Context, logger logging.Logger, iface net.Interface) ([]*url.URL, error) {
	logger.Debugf("sending WS-Discovery probe using interface: %s", iface.Name)
	var discoveryResps []string
	attempts := 3 // run ws-discovery probe 3 times due to sync flakiness between announcer and requester
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context canceled for interface: %s", iface.Name)
		}

		resp, err := SendProbe(iface.Name)
		if err != nil {
			logger.Debugf("attempt %d: failed to send WS-Discovery probe on interface %s: %w", i+1, iface.Name, err)
			continue
		}

		discoveryResps = append(discoveryResps, resp...)
	}

	if len(discoveryResps) == 0 {
		return nil, fmt.Errorf("no unique discovery responses received on interface %s after multiple attempts", iface.Name)
	}

	xaddrsSet := make(map[string]*url.URL)
	for _, response := range discoveryResps {
		for _, xaddr := range extractXAddrsFromProbeMatch(response, logger) {
			xaddrsSet[xaddr.Host] = xaddr
		}
	}

	return slices.Collect(maps.Values(xaddrsSet)), nil
}

func DiscoverCamerasOnXAddr(
	ctx context.Context,
	xaddr *url.URL,
	creds []Credentials,
	logger logging.Logger,
) (CameraInfo, error) {
	var zero CameraInfo
	for _, cred := range creds {
		if ctx.Err() != nil {
			return zero, fmt.Errorf("context canceled while connecting to ONVIF device: %s", xaddr)
		}
		cameraInfo, err := GetCameraInfo(xaddr, cred, logger)
		if err != nil {
			logger.Warnf("Failed to get camera info from %s: %v", xaddr, err)
			continue
		}
		// once we have addeed a camera info break
		return cameraInfo, nil
	}
	return zero, fmt.Errorf("no credentials matched IP %s", xaddr)
}

// TODO(Nick S): What happens if we don't do this?
func ValidWSDiscoveryInterface(iface net.Interface) bool {
	addrs, err := iface.Addrs()
	if err != nil {
		panic(err)
	}
	addrsNetworksStr := []string{}
	addrsStr := []string{}
	for _, a := range addrs {
		addrsNetworksStr = append(addrsNetworksStr, a.Network())
		addrsStr = append(addrsStr, a.String())
	}

	multiAddrs, err := iface.MulticastAddrs()
	if err != nil {
		panic(err)
	}
	multiAddrsNetworkStr := []string{}
	multiAddrsStr := []string{}
	for _, a := range multiAddrs {
		addrsNetworksStr = append(multiAddrsNetworkStr, a.Network())
		multiAddrsStr = append(multiAddrsStr, a.String())
	}
	// log.Printf("iface: %s, FlagUp: %d, FlagBroadcast: %d, FlagLoopback: %d, FlagPointToPoint: %d, FlagMulticast: %d, FlagRunning: %d, flags: %s, "+
	// 	"addrs: %#v, addrsNetworks: %#v, multicastaddrs: %#v, multicastaddrsNetworks: %#v\n", iface.Name, iface.Flags&net.FlagUp, iface.Flags&net.FlagBroadcast, iface.Flags&net.FlagLoopback, iface.Flags&net.FlagPointToPoint, iface.Flags&net.FlagMulticast, iface.Flags&net.FlagRunning, iface.Flags.String(),
	// 	addrsStr, addrsNetworksStr, multiAddrsStr, multiAddrsNetworkStr)
	return iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 && iface.Flags&net.FlagLoopback == 0
}

// extractXAddrsFromProbeMatch extracts XAddrs from the WS-Discovery ProbeMatch response.
func extractXAddrsFromProbeMatch(response string, logger logging.Logger) []*url.URL {
	type ProbeMatch struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			ProbeMatches struct {
				ProbeMatch []struct {
					XAddrs string `xml:"XAddrs"`
				} `xml:"ProbeMatch"`
			} `xml:"ProbeMatches"`
		} `xml:"Body"`
	}

	var probeMatch ProbeMatch
	err := xml.NewDecoder(strings.NewReader(response)).Decode(&probeMatch)
	if err != nil {
		logger.Warnf("error unmarshalling ONVIF discovery xml response: %w\nFull xml resp: %s", err, response)
	}

	xaddrs := []*url.URL{}
	for _, match := range probeMatch.Body.ProbeMatches.ProbeMatch {
		for _, xaddr := range strings.Split(match.XAddrs, " ") {
			parsedURL, err := url.Parse(xaddr)
			if err != nil {
				logger.Warnf("failed to parse XAddr %s: %w", xaddr, err)
				continue
			}

			xaddrs = append(xaddrs, parsedURL)
		}
	}

	return xaddrs
}

func GetDeviceInformation(onvifDevice OnvifDevice, logger logging.Logger) (DeviceInfo, error) {
	var zero DeviceInfo
	resp, err := onvifDevice.CallMethod(device.GetDeviceInformation{})
	if err != nil {
		return zero, fmt.Errorf("failed to get device information: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	logger.Debugf("GetDeviceInformation response body: %s", string(b))

	var deviceInfo DeviceInfo
	err = xml.NewDecoder(bytes.NewReader(b)).Decode(&deviceInfo)
	if err != nil {
		return zero, fmt.Errorf("failed to decode device information response: %w", err)
	}
	return deviceInfo, nil
}

// GetCameraInfo uses the ONVIF Media service to get the RTSP stream URLs and camera details.
func GetCameraInfo(xaddr *url.URL, creds Credentials, logger logging.Logger) (CameraInfo, error) {
	var zero CameraInfo
	// This calls GetCapabilities
	onvifDevice, err := onvif.NewDevice(onvif.DeviceParams{
		Xaddr:    xaddr.Host,
		Username: creds.User,
		Password: creds.Pass,
	})
	if err != nil {
		return zero, fmt.Errorf("failed to connect to ONVIF device: %v", err)
	}

	logger.Debugf("ip: %s GetCapabilities: DeviceInfo: %#v, Endpoints: %#v", xaddr, onvifDevice.GetDeviceInfo(), onvifDevice.GetServices())
	// Fetch device information (manufacturer, serial number, etc.)
	deviceInfo, err := GetDeviceInformation(onvifDevice, logger)
	if err != nil {
		return zero, fmt.Errorf("failed to read device information response body: %w", err)
	}

	// Call the ONVIF Media service to get the available media profiles using the same device instance
	rtspURLs, err := GetRTSPStreamURIsFromProfiles(onvifDevice, creds, logger)
	if err != nil {
		return zero, fmt.Errorf("failed to get RTSP URLs: %w", err)
	}

	cameraInfo := CameraInfo{
		WSDiscoveryXAddr: xaddr.Host,
		RTSPURLs:         rtspURLs,
		Manufacturer:     deviceInfo.Body.GetDeviceInformationResponse.Manufacturer,
		Model:            deviceInfo.Body.GetDeviceInformationResponse.Model,
		SerialNumber:     deviceInfo.Body.GetDeviceInformationResponse.SerialNumber,
		FirmwareVersion:  deviceInfo.Body.GetDeviceInformationResponse.FirmwareVersion,
		HardwareId:       deviceInfo.Body.GetDeviceInformationResponse.HardwareId,
	}

	return cameraInfo, nil
}

func GetProfiles(onvifDevice OnvifDevice, logger logging.Logger) (GetProfilesResponse, error) {
	var zero GetProfilesResponse
	getProfiles := media.GetProfiles{}
	profilesResponse, err := onvifDevice.CallMethod(getProfiles)
	if err != nil {
		return zero, fmt.Errorf("failed to get media profiles: %w", err)
	}
	defer profilesResponse.Body.Close()

	profilesBody, err := io.ReadAll(profilesResponse.Body)
	if err != nil {
		return zero, fmt.Errorf("failed to read profiles response body: %w", err)
	}
	logger.Debugf("GetProfiles response body: %s", profilesBody)
	// Reset the response body reader after logging
	profilesResponse.Body = io.NopCloser(bytes.NewReader(profilesBody))

	var envelope GetProfilesResponse
	err = xml.NewDecoder(profilesResponse.Body).Decode(&envelope)
	if err != nil {
		return zero, fmt.Errorf("failed to decode media profiles response: %w", err)
	}

	if len(envelope.Body.GetProfilesResponse.Profiles) == 0 {
		logger.Warn("No media profiles found in the response")
		return zero, errors.New("no media profiles found")
	}

	logger.Debugf("Found %d media profiles", len(envelope.Body.GetProfilesResponse.Profiles))
	for i, profile := range envelope.Body.GetProfilesResponse.Profiles {
		logger.Debugf("Profile %d: Token=%s, Name=%s", i, profile.Token, profile.Name)
	}

	return envelope, nil
}

func GetStreamURI(onvifDevice OnvifDevice, profile onvifxsd.Profile, creds Credentials, logger logging.Logger) (*url.URL, error) {
	logger.Debugf("Using profile token and profile: %s %#v", profile.Token, profile)
	resp, err := onvifDevice.CallMethod(media.GetStreamUri{
		StreamSetup: onvifxsd.StreamSetup{
			Stream:    onvifxsd.StreamType(streamTypeRTPUnicast),
			Transport: onvifxsd.Transport{Protocol: streamSetupProtocol},
		},
		ProfileToken: profile.Token,
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logger.Debugf("GetStreamUri response: %v", string(body))

	// Reset the response body reader after logging
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var streamURI getStreamURIResponse
	if err := xml.NewDecoder(resp.Body).Decode(&streamURI); err != nil {
		return nil, fmt.Errorf("Failed to get RTSP URL for profile %s: %v", profile.Token, err)
	}

	logger.Debugf("stream uri response for profile %s: %v: ", profile.Token, streamURI)

	uriStr := string(streamURI.Body.GetStreamURIResponse.MediaURI.Uri)
	if uriStr == "" {
		return nil, fmt.Errorf("got empty uri for profile %s", profile.Token)
	}

	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI %s: %v", uriStr, err)
	}

	if creds.User != "" || creds.Pass != "" {
		uri.User = url.UserPassword(creds.User, creds.Pass)
	}
	return uri, nil
}

// GetRTSPStreamURIsFromProfiles uses the ONVIF Media service to get the RTSP stream URLs for all available profiles.
func GetRTSPStreamURIsFromProfiles(onvifDevice OnvifDevice, creds Credentials, logger logging.Logger) ([]string, error) {
	resp, err := GetProfiles(onvifDevice, logger)
	if err != nil {
		return nil, err
	}

	// Resultant slice of RTSP URIs
	var rtspUris []string
	// Iterate over all profiles and get the RTSP stream URI for each one
	for _, profile := range resp.Body.GetProfilesResponse.Profiles {
		uri, err := GetStreamURI(onvifDevice, profile, creds, logger)
		if err != nil {
			logger.Warnf(err.Error())
			continue
		}

		rtspUris = append(rtspUris, uri.String())
	}

	return rtspUris, nil
}
