// Package viamonvif provides ONVIF integration to the viamrtsp module
package viamonvif

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	wsdiscovery "github.com/use-go/onvif/ws-discovery"
	onvifxsd "github.com/use-go/onvif/xsd/onvif"
	"go.viam.com/rdk/logging"
	"go.viam.com/utils"
)

const (
	streamTypeRTPUnicast = "RTP-Unicast"
)

// OnvifDevice is an interface to abstract device methods used in the code.
// Used instead of onvif.Device to allow for mocking in tests.
type OnvifDevice interface {
	CallMethod(request interface{}) (*http.Response, error)
}

// getProfilesResponse is the schema the GetProfiles response is formatted in.
type getProfilesResponse struct {
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

// getDeviceInformationResponse is the schema the GetDeviceInformation response is formatted in.
type getDeviceInformationResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetDeviceInformationResponse struct {
			Manufacturer string `xml:"Manufacturer"`
			Model        string `xml:"Model"`
			SerialNumber string `xml:"SerialNumber"`
		} `xml:"GetDeviceInformationResponse"`
	} `xml:"Body"`
}

// CameraInfo holds both the RTSP URLs and supplementary camera details.
type CameraInfo struct {
	RTSPURLs     []string `json:"rtsp_urls"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	SerialNumber string   `json:"serial_number"`
}

// CameraInfoList is a struct containing a list of CameraInfo structs.
type CameraInfoList struct {
	Cameras []CameraInfo `json:"cameras"`
}

// DiscoverCameras performs WS-Discovery using the use-go/onvif discovery utility,
// then uses ONVIF queries to get available RTSP addresses and supplementary info.
func DiscoverCameras(username, password string, logger logging.Logger) (*CameraInfoList, error) {
	var discoveredCameras []CameraInfo
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve network interfaces: %w", err)
	}

	// Filter out unsuitable interfaces
	var validInterfaces []net.Interface
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 && iface.Flags&net.FlagLoopback == 0 {
			validInterfaces = append(validInterfaces, iface)
		} else {
			logger.Debugf("skipping interface %s: does not meet WS-Discovery requirements", iface.Name)
		}
	}
	// If no valid interfaces, return an error
	if len(validInterfaces) == 0 {
		return nil, errors.New("no valid net interfaces for WS-Discovery")
	}

	resultsChan := make(chan []CameraInfo, len(validInterfaces))
	activeWorkers := sync.WaitGroup{}
	for _, iface := range validInterfaces {
		activeWorkers.Add(1)
		utils.ManagedGo(func() {
			logger.Debugf("sending WS-Discovery probe using interface: %s", iface.Name)
			var discoveryResps []string
			iterations := 3 // run ws-discovery probe 3 times due to sync flakiness between announcer and requester
			for i := range make([]int, iterations) {
				if ctx.Err() != nil {
					logger.Infof("context canceled for interface: %s", iface.Name)
					return
				}

				// other ws-discovery args
				scopes := []string{}
				types := []string{"dn:NetworkVideoTransmitter"}
				namespaces := map[string]string{}
				resp, err := wsdiscovery.SendProbe(iface.Name, scopes, types, namespaces)
				if err != nil {
					logger.Warnf("attempt %d: failed to send WS-Discovery probe on interface %s: %v", i+1, iface.Name, err)
					break
				}
				discoveryResps = append(discoveryResps, resp...)
			}

			if len(discoveryResps) == 0 {
				logger.Warnf("no unique discovery responses received on interface %s after multiple attempts", iface.Name)
				return
			}

			xaddrsSet := make(map[string]struct{})
			for _, response := range discoveryResps {
				xaddrs := extractXAddrsFromProbeMatch([]byte(response), logger)
				for _, xaddr := range xaddrs {
					xaddrsSet[xaddr] = struct{}{}
				}
			}

			var cameraInfos []CameraInfo
			for xaddr := range xaddrsSet {
				if ctx.Err() != nil {
					logger.Infof("context canceled while connecting to ONVIF device: %s", xaddr)
					return
				}

				logger.Infof("Connecting to ONVIF device with URL: %s", xaddr)
				deviceInstance, err := onvif.NewDevice(onvif.DeviceParams{
					Xaddr:    xaddr,
					Username: username,
					Password: password,
				})
				if err != nil {
					logger.Warnf("failed to connect to ONVIF device: %w", err)
					continue
				}
				cameraInfo, err := getCameraInfo(deviceInstance, username, password, logger)
				if err != nil {
					logger.Warnf("Failed to get camera info from %s: %v\n", xaddr, err)
					continue
				}
				cameraInfos = append(cameraInfos, cameraInfo)
			}

			if len(cameraInfos) > 0 {
				resultsChan <- cameraInfos
			}
		}, func() {
			defer activeWorkers.Done()
		})
	}

	go func() {
		activeWorkers.Wait()
		close(resultsChan)
	}()

	for cameraInfos := range resultsChan {
		discoveredCameras = append(discoveredCameras, cameraInfos...)
	}

	return &CameraInfoList{Cameras: discoveredCameras}, nil
}

// extractXAddrsFromProbeMatch extracts XAddrs from the WS-Discovery ProbeMatch response.
func extractXAddrsFromProbeMatch(response []byte, logger logging.Logger) []string {
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
	err := xml.NewDecoder(bytes.NewReader(response)).Decode(&probeMatch)
	if err != nil {
		logger.Warnf("error unmarshalling ONVIF discovery xml response: %w\nFull xml resp: %s", err, response)
	}

	var xaddrs []string
	for _, match := range probeMatch.Body.ProbeMatches.ProbeMatch {
		for _, xaddr := range strings.Split(match.XAddrs, " ") {
			parsedURL, err := url.Parse(xaddr)
			if err != nil {
				logger.Warnf("failed to parse XAddr %s: %w", xaddr, err)
				continue
			}

			// Ensure only base address (hostname or IP) is used
			baseAddress := parsedURL.Host
			if baseAddress == "" {
				continue
			}

			xaddrs = append(xaddrs, baseAddress)
		}
	}

	return xaddrs
}

// getCameraInfo uses the ONVIF Media service to get the RTSP stream URLs and camera details.
func getCameraInfo(deviceInstance OnvifDevice, username, password string, logger logging.Logger) (CameraInfo, error) {
	// Fetch device information (manufacturer, serial number, etc.)
	getDeviceInfo := device.GetDeviceInformation{}
	deviceInfoResponse, err := deviceInstance.CallMethod(getDeviceInfo)
	if err != nil {
		return CameraInfo{}, fmt.Errorf("failed to get device information: %w", err)
	}
	defer deviceInfoResponse.Body.Close()

	deviceInfoBody, err := io.ReadAll(deviceInfoResponse.Body)
	if err != nil {
		return CameraInfo{}, fmt.Errorf("failed to read device information response body: %w", err)
	}
	logger.Debugf("GetDeviceInformation response body: %s", deviceInfoBody)
	deviceInfoResponse.Body = io.NopCloser(bytes.NewReader(deviceInfoBody))

	var deviceInfoEnvelope getDeviceInformationResponse
	err = xml.NewDecoder(deviceInfoResponse.Body).Decode(&deviceInfoEnvelope)
	if err != nil {
		return CameraInfo{}, fmt.Errorf("failed to decode device information response: %w", err)
	}

	// Call the ONVIF Media service to get the available media profiles using the same device instance
	rtspURLs, err := getRTSPStreamURLs(deviceInstance, username, password, logger)
	if err != nil {
		return CameraInfo{}, fmt.Errorf("failed to get RTSP URLs: %w", err)
	}

	cameraInfo := CameraInfo{
		RTSPURLs:     rtspURLs,
		Manufacturer: deviceInfoEnvelope.Body.GetDeviceInformationResponse.Manufacturer,
		Model:        deviceInfoEnvelope.Body.GetDeviceInformationResponse.Model,
		SerialNumber: deviceInfoEnvelope.Body.GetDeviceInformationResponse.SerialNumber,
	}

	return cameraInfo, nil
}

// getRTSPStreamURLs uses the ONVIF Media service to get the RTSP stream URLs for all available profiles.
func getRTSPStreamURLs(deviceInstance OnvifDevice, username, password string, logger logging.Logger) ([]string, error) {
	getProfiles := media.GetProfiles{}
	profilesResponse, err := deviceInstance.CallMethod(getProfiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get media profiles: %w", err)
	}
	defer profilesResponse.Body.Close()

	profilesBody, err := io.ReadAll(profilesResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles response body: %w", err)
	}
	logger.Debugf("GetProfiles response body: %s", profilesBody)
	// Reset the response body reader after logging
	profilesResponse.Body = io.NopCloser(bytes.NewReader(profilesBody))

	var envelope getProfilesResponse
	err = xml.NewDecoder(profilesResponse.Body).Decode(&envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to decode media profiles response: %w", err)
	}

	if len(envelope.Body.GetProfilesResponse.Profiles) == 0 {
		logger.Warn("No media profiles found in the response")
		return nil, errors.New("no media profiles found")
	}

	logger.Debugf("Found %d media profiles", len(envelope.Body.GetProfilesResponse.Profiles))
	for i, profile := range envelope.Body.GetProfilesResponse.Profiles {
		logger.Debugf("Profile %d: Token=%s, Name=%s", i, profile.Token, profile.Name)
	}

	// Resultant slice of RTSP URIs
	var rtspUris []string
	// Iterate over all profiles and get the RTSP stream URI for each one
	for _, profile := range envelope.Body.GetProfilesResponse.Profiles {
		logger.Debugf("Using profile token: %s", profile.Token)

		getStreamURI := media.GetStreamUri{
			StreamSetup: onvifxsd.StreamSetup{
				Stream: onvifxsd.StreamType(streamTypeRTPUnicast),
			},
			ProfileToken: profile.Token,
		}

		streamURIResponse, err := deviceInstance.CallMethod(getStreamURI)
		if err != nil {
			logger.Warnf("Failed to get RTSP URL for profile %s: %v", profile.Token, err)
			continue
		}
		defer streamURIResponse.Body.Close()

		streamURIBody, err := io.ReadAll(streamURIResponse.Body)
		if err != nil {
			logger.Warnf("Failed to read stream URI response body for profile %s: %v", profile.Token, err)
			continue
		}
		logger.Debugf("GetStreamUri response body for profile %s: %s", profile.Token, streamURIBody)
		// Reset the response body reader after logging
		streamURIResponse.Body = io.NopCloser(bytes.NewReader(streamURIBody))

		var streamURI getStreamURIResponse
		err = xml.NewDecoder(streamURIResponse.Body).Decode(&streamURI)
		if err != nil {
			logger.Warnf("Failed to decode stream URI response for profile %s: %v", profile.Token, err)
			continue
		}

		uri := string(streamURI.Body.GetStreamURIResponse.MediaURI.Uri)
		if parsedURI, err := url.Parse(uri); err == nil {
			if username != "" && password != "" {
				parsedURI.User = url.UserPassword(username, password)
				uri = parsedURI.String()
			}
		} else {
			logger.Warnf("Failed to parse URI %s: %v", uri, err)
			continue
		}
		rtspUris = append(rtspUris, uri)
	}

	return rtspUris, nil
}
