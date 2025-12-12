// Package unifi provides Unifi discovery for RTSP cameras
package unifi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/viam-modules/viamrtsp"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

const (
	httpClientTimeout = 30 * time.Second
	idSuffixLength    = 6
)

// Model is the model for the Unifi discovery service.
var Model = viamrtsp.Family.WithModel("unifi")

// Config is the configuration for the Unifi discovery service.
type Config struct {
	NVRAddress string `json:"nvr_address"`
	UnifiToken string `json:"unifi_token"`
}

// unifiCamera represents a camera from the UniFi Protect API.
type unifiCamera struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// rtspStreamResponse represents the RTSPS stream response from the API.
type rtspStreamResponse struct {
	High    string `json:"high"`
	Medium  string `json:"medium"`
	Low     string `json:"low"`
	Package string `json:"package"`
}

type unifiDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable
	logger     logging.Logger
	unifToken  string
	nvrAddr    string
	httpClient *http.Client
}

// Validate validates the Unifi discovery service configuration.
func (cfg *Config) Validate(_ string) ([]string, []string, error) {
	if cfg.NVRAddress == "" {
		return nil, nil, errors.New("nvr_address is required")
	}
	if cfg.UnifiToken == "" {
		return nil, nil, errors.New("unifi_token is required")
	}
	return nil, nil, nil
}

func init() {
	resource.RegisterService(
		discovery.API,
		Model,
		resource.Registration[discovery.Service, *Config]{
			Constructor: newUnifiDiscovery,
		})
}

// NewUnifiDiscovery creates a new Unifi discovery service (exported for testing).
func NewUnifiDiscovery(ctx context.Context, deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	return newUnifiDiscovery(ctx, deps, conf, logger)
}

func newUnifiDiscovery(_ context.Context, _ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}

	dis := &unifiDiscovery{
		Named:      conf.ResourceName().AsNamed(),
		unifToken:  cfg.UnifiToken,
		nvrAddr:    cfg.NVRAddress,
		logger:     logger,
		httpClient: newHTTPClient(),
	}

	return dis, nil
}

func (dis *unifiDiscovery) DiscoverResources(ctx context.Context, _ map[string]any) ([]resource.Config, error) {
	// Get list of cameras from NVR
	cameras, err := dis.getCameras(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cameras: %w", err)
	}

	dis.logger.Infof("Found %d cameras on NVR %s", len(cameras), dis.nvrAddr)

	var configs []resource.Config

	for _, cam := range cameras {
		rtspURL, err := dis.getRTSPStream(ctx, cam.ID)
		if err != nil {
			dis.logger.Warnf("Failed to get RTSP stream for camera %s (%s): %v", cam.Name, cam.ID, err)
			continue
		}

		if rtspURL == "" {
			dis.logger.Warnf("No RTSP stream available for camera %s (%s)", cam.Name, cam.ID)
			continue
		}

		dis.logger.Infof("Camera %s: %s", cam.Name, rtspURL)

		// Create a camera config for this discovered camera
		cfg := resource.Config{
			API:   camera.API,
			Model: viamrtsp.ModelAgnostic,
			Name:  sanitizeName(cam.Name, cam.ID),
			Attributes: map[string]any{
				"rtsp_address": rtspURL,
			},
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

func (dis *unifiDiscovery) getCameras(ctx context.Context) ([]unifiCamera, error) {
	url := fmt.Sprintf("https://%s/proxy/protect/integration/v1/cameras", dis.nvrAddr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", dis.unifToken)
	req.Header.Set("Accept", "application/json")

	resp, err := dis.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var cameras []unifiCamera
	if err := json.NewDecoder(resp.Body).Decode(&cameras); err != nil {
		return nil, fmt.Errorf("failed to decode cameras response: %w", err)
	}

	return cameras, nil
}

func (dis *unifiDiscovery) getRTSPStream(ctx context.Context, cameraID string) (string, error) {
	url := fmt.Sprintf("https://%s/proxy/protect/integration/v1/cameras/%s/rtsps-stream", dis.nvrAddr, cameraID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Api-Key", dis.unifToken)
	req.Header.Set("Accept", "application/json")

	resp, err := dis.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	var streamResp rtspStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&streamResp); err != nil {
		return "", fmt.Errorf("failed to decode stream response: %w", err)
	}

	// Use high quality stream, fall back to medium/low if not available
	rtspsURL := streamResp.High
	if rtspsURL == "" {
		rtspsURL = streamResp.Medium
	}
	if rtspsURL == "" {
		rtspsURL = streamResp.Low
	}
	if rtspsURL == "" {
		return "", nil
	}

	// Convert RTSPS to RTSP: change port 7441 to 7447 and remove ?enableSrtp
	rtspURL := convertRTSPStoRTSP(rtspsURL)

	return rtspURL, nil
}

// convertRTSPStoRTSP converts an RTSPS URL to plain RTSP.
// Example: rtsps://10.1.14.106:7441/6uVHv61ad7NDfMCS?enableSrtp -> rtsp://10.1.14.106:7447/6uVHv61ad7NDfMCS
func convertRTSPStoRTSP(rtspsURL string) string {
	rtspURL := strings.Replace(rtspsURL, "rtsps://", "rtsp://", 1)
	rtspURL = strings.Replace(rtspURL, ":7441/", ":7447/", 1)
	if idx := strings.Index(rtspURL, "?enableSrtp"); idx != -1 {
		rtspURL = rtspURL[:idx]
	}

	return rtspURL
}

// checkResponse validates the HTTP response status and content type.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("authentication failed: invalid or expired API token")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Check content type to catch HTML error pages
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unexpected content type %q (expected application/json), check API token", contentType)
	}

	return nil
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: httpClientTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // UniFi NVRs use self-signed certs
			},
		},
	}
}

// sanitizeName converts a camera name to a valid resource name with ID suffix for uniqueness.
func sanitizeName(name, id string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	// Add last 6 chars of ID as suffix to ensure uniqueness
	if len(id) >= idSuffixLength {
		name = name + "_" + id[len(id)-idSuffixLength:]
	} else if id != "" {
		name = name + "_" + id
	}
	return name
}
