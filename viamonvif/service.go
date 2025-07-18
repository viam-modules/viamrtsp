// Package viamonvif provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvif

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/icholy/digest"
	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
	"go.viam.com/utils"
)

// Model is the model for a onvif discovery service for rtsp cameras.
var (
	Model             = viamrtsp.Family.WithModel("onvif")
	errNoCamerasFound = errors.New("no cameras found, ensure cameras are working or check credentials")
	emptyCred         = device.Credentials{}
)

const (
	snapshotClientTimeout = 5 * time.Second
	rtspPollTimeout       = 5 * time.Second
	rtspImageInterval     = 100 * time.Millisecond
	rtspNameSaltLength    = 4
	discoveryInterval     = time.Minute
	imageReqMimeType      = "image/jpeg"
)

func init() {
	resource.RegisterService(
		discovery.API,
		Model,
		resource.Registration[discovery.Service, *Config]{
			Constructor: newDiscovery,
		})
}

// Config is the config for the discovery service.
type Config struct {
	Credentials []device.Credentials `json:"credentials"`
}

// Validate validates the discovery service.
func (cfg *Config) Validate(_ string) ([]string, []string, error) {
	// check that all creds have usernames set. Note a credential can have both fields empty
	for _, cred := range cfg.Credentials {
		if cred.Pass != "" && cred.User == "" {
			return nil, nil, fmt.Errorf("credential missing username, has password %v", cred.Pass)
		}
	}
	return []string{}, nil, nil
}

type previewRequest struct {
	rtspURL string
}

type rtspDiscovery struct {
	resource.Named
	resource.AlwaysRebuild

	rtspToSnapshotURIsMu sync.Mutex
	rtspToSnapshotURIs   map[string]string

	Credentials []device.Credentials
	mdnsServer  *mdnsServer
	logger      logging.Logger

	workers *utils.StoppableWorkers

	discoveredResourcesMu sync.Mutex
	discoveredResources   []resource.Config
}

func newDiscovery(_ context.Context, _ resource.Dependencies,
	conf resource.Config,
	logger logging.Logger,
) (discovery.Service, error) {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}

	dis := &rtspDiscovery{
		Named:       conf.ResourceName().AsNamed(),
		Credentials: append([]device.Credentials{emptyCred}, cfg.Credentials...),
		logger:      logger,
		workers:     utils.NewBackgroundStoppableWorkers(),
	}

	// viam-server sets this environment variable. The contents of this directory is expected to
	// persist across process restarts and module upgrades.
	moduleDataDir := os.Getenv("VIAM_MODULE_DATA")
	if !strings.HasPrefix(moduleDataDir, "/") {
		dis.mdnsServer = newMDNSServer(logger)
	} else {
		dis.mdnsServer = newMDNSServerFromCachedData(
			filepath.Join(moduleDataDir, "mdns_cache.json"), logger.Sublogger("mdns"))
	}

	dis.workers.Add(dis.discoveryBackgroundWorker)

	return dis, nil
}

// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *rtspDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	// If extra is not empty, we assume the user wants to discover resources
	// with provided credentials. We ignore any previously discovered resources
	// and re-run discovery with extra parameters.
	_, hasExtraCreds := getCredFromExtra(extra)
	if hasExtraCreds {
		dis.logger.Debugf("running discovery lookup with extra parameters: %v", extra)
		discovered, err := dis.runDiscoveryLookup(ctx, extra)
		if err != nil {
			return nil, fmt.Errorf("failed to run discovery lookup: %w", err)
		}
		return discovered, nil
	}

	// If we have previously discovered cameras, we will return the cached resources.
	dis.discoveredResourcesMu.Lock()
	hasDiscoveredResources := len(dis.discoveredResources) > 0
	if hasDiscoveredResources {
		result := dis.discoveredResources
		dis.discoveredResourcesMu.Unlock()
		dis.logger.Debug("returning cached discovered resources")
		return result, nil
	}
	dis.discoveredResourcesMu.Unlock()

	// If discovery has not been run before, or no cameras were discovered,
	// we will attempt the discovery lookup again.
	dis.logger.Debug("no cached resources, running discovery lookup with config credentials")
	discovered, err := dis.runDiscoveryLookup(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to run discovery lookup: %w", err)
	}
	dis.discoveredResourcesMu.Lock()
	dis.discoveredResources = discovered
	dis.discoveredResourcesMu.Unlock()

	return discovered, nil
}

func (dis *rtspDiscovery) runDiscoveryLookup(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	localRTSPToSnapshotURIs := make(map[string]string)
	cams := []resource.Config{}

	discoverCreds := dis.Credentials

	extraCred, ok := getCredFromExtra(extra)
	if ok {
		discoverCreds = append(discoverCreds, extraCred)
	}
	list, err := DiscoverCameras(ctx, discoverCreds, nil, dis.logger)
	if err != nil {
		return nil, err
	}
	if len(list.Cameras) == 0 {
		return nil, errNoCamerasFound
	}

	for _, camInfo := range list.Cameras {
		dis.logger.Debugf("%s %s %s", camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
		// some cameras return with no urls. explicitly skipping those so the behavior is clear in the service.
		if len(camInfo.MediaEndpoints) == 0 {
			dis.logger.Errorf("No urls found for camera, skipping. %s %s %s",
				camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
			continue
		}

		// tryMDNS will attempt to register an mdns entry for the camera. If successfully
		// registered, `tryMDNS` will additionally mutate the `camInfo.RTSPURLs` to use the dns
		// hostname rather than a raw IP. Such that the camera configs we are about to generate will
		// use the dns hostname.
		// mDNS hostname to IP address resolution is not working on Windows so we skip it.
		// TODO(RSDK-10796): Add windows mDNS support to zeroconf fork
		if runtime.GOOS != "windows" {
			camInfo.tryMDNS(dis.mdnsServer, dis.logger)
		}

		camConfigs, err := createCamerasFromURLs(camInfo, dis.Name().ShortName(), dis.logger)
		if err != nil {
			return nil, err
		}
		for _, endpoint := range camInfo.MediaEndpoints {
			// If available, we will use mdns rtsp address as the key instead of the original rtsp address
			localRTSPToSnapshotURIs[endpoint.StreamURI] = endpoint.SnapshotURI
			dis.logger.Debugf("Added snapshot mapping: %s - %s", endpoint.StreamURI, endpoint.SnapshotURI)
		}
		cams = append(cams, camConfigs...)
	}

	// Only lock when updating the shared URI map
	dis.rtspToSnapshotURIsMu.Lock()
	dis.rtspToSnapshotURIs = localRTSPToSnapshotURIs
	dis.rtspToSnapshotURIsMu.Unlock()

	dis.mdnsServer.UpdateCacheFile()

	return cams, nil
}

func (dis *rtspDiscovery) DoCommand(ctx context.Context, command map[string]interface{}) (map[string]interface{}, error) {
	cmd, ok := command["command"].(string)
	if !ok {
		return nil, errors.New("invalid command type")
	}

	switch cmd {
	case "preview":
		dis.logger.Debugf("snapshot command received")
		previewReq, err := toPreviewCommand(command)
		if err != nil {
			return nil, err
		}

		dataURL, err := dis.preview(ctx, previewReq.rtspURL)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"preview": dataURL,
		}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

// preview fetches a camera image from the given RTSP URL.
func (dis *rtspDiscovery) preview(ctx context.Context, rtspURL string) (string, error) {
	var snapshotErr, rtspErr error

	// Attempt to fetch the image from the snapshot URI
	dis.rtspToSnapshotURIsMu.Lock()
	snapshotURI, ok := dis.rtspToSnapshotURIs[rtspURL]
	dis.rtspToSnapshotURIsMu.Unlock()

	if ok {
		dis.logger.Debugf("attempting to fetch image from snapshot URI: %s", snapshotURI)
		dataURL, err := downloadPreviewImage(ctx, dis.logger, snapshotURI)
		if err == nil {
			dis.logger.Debugf("successfully fetched image from snapshot URI: %s", snapshotURI)
			return dataURL, nil
		}
		dis.logger.Debugf("failed to fetch image from snapshot URI: %s, error: %v", snapshotURI, err)
		snapshotErr = fmt.Errorf("snapshot error for snapshot URI %s: %w", snapshotURI, err)
	} else {
		dis.logger.Debugf("snapshot URI not found for RTSP URL: %s", rtspURL)
		snapshotErr = fmt.Errorf("snapshot URI not found for RTSP URL: %s", rtspURL)
	}

	// Fallback to fetching the image via RTSP
	dis.logger.Debugf("attempting to fetch image via RTSP URL: %s", rtspURL)
	dataURL, err := fetchImageFromRTSPURL(ctx, dis.logger, rtspURL)
	if err == nil {
		dis.logger.Debugf("successfully fetched image via RTSP for URL: %s", rtspURL)
		return dataURL, nil
	}
	dis.logger.Warnf("failed to fetch image via RTSP for URL: %s, error: %v", rtspURL, err)
	rtspErr = fmt.Errorf("RTSP error: %w", err)

	return "", fmt.Errorf("both snapshot and RTSP fetch failed: %w", errors.Join(snapshotErr, rtspErr))
}

// discoveryBackgroundWorker loops and runs the discovery service's DiscoverResources method
func (dis *rtspDiscovery) discoveryBackgroundWorker(ctx context.Context) {
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if discovered, err := dis.runDiscoveryLookup(ctx, nil); err != nil {
				dis.logger.Errorf("discovery failed: %v", err)
				dis.discoveredResourcesMu.Lock()
				dis.discoveredResources = discovered
				dis.discoveredResourcesMu.Unlock()
			} else {
				dis.logger.Debug("discovery completed successfully")
			}
		case <-ctx.Done():
			dis.logger.Debug("discovery worker context done, exiting")
			return
		}
	}
}

func (dis *rtspDiscovery) Close(_ context.Context) error {
	dis.mdnsServer.Shutdown()
	dis.logger.Debug("stopping discovery service workers")
	dis.workers.Stop()
	dis.logger.Debug("discovery service closed")

	return nil
}

func toPreviewCommand(command map[string]interface{}) (*previewRequest, error) {
	attributes, ok := command["attributes"].(map[string]interface{})
	if !ok {
		return nil, errors.New("attributes is missing or not a map")
	}
	rtspURL, ok := attributes["rtsp_address"].(string)
	if !ok {
		return nil, errors.New("rtsp_address cannot be empty")
	}
	return &previewRequest{rtspURL: rtspURL}, nil
}

// formatDataURL formats the image data and content type into a data URL.
func formatDataURL(contentType string, imageBytes []byte) string {
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Image)
}

// downloadPreviewImage downloads the preview image from the snapshot uri and returns it as a data URL.
func downloadPreviewImage(ctx context.Context, logger logging.Logger, snapshotURI string) (string, error) {
	parsedURL, err := url.Parse(snapshotURI)
	if err != nil {
		return "", fmt.Errorf("found an invalid snapshot URI: %w", err)
	}

	var username, password string
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		if pwd, hasPassword := parsedURL.User.Password(); hasPassword {
			password = pwd
		}
		if password == "" {
			logger.Warnf("found a snapshot URI with no password: %s", snapshotURI)
		}
		logger.Debugf("creating snapshot request using credentials: username=%s", username)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}
	client := &http.Client{
		// Setting upper bound timeout in case the ctx never times out
		Timeout: snapshotClientTimeout,
		Transport: &digest.Transport{
			Username:  username,
			Password:  password,
			Transport: transport,
		},
	}

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, snapshotURI, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	resp, err := client.Do(getReq)
	if err != nil {
		return "", fmt.Errorf("failed to execute http request: %w", err)
	}
	defer resp.Body.Close()
	logger.Debugf("snapshot response status: %s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		statusText := http.StatusText(resp.StatusCode)
		bodyText := "<could not read response body>"
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			bodyText = string(bodyBytes)
		} else {
			logger.Warnf("failed to read error response body: %v", err)
		}
		return "", fmt.Errorf("failed to get snapshot image, status %d: %s, body: %s", resp.StatusCode, statusText, bodyText)
	}

	contentType := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", fmt.Errorf("failed to parse content type %s: %w", contentType, err)
	}
	isImage := strings.HasPrefix(mediaType, "image/")
	if !isImage {
		return "", fmt.Errorf("snapshot URI returned non-image mime type: %s", mediaType)
	}
	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data from http response: %w", err)
	}
	logger.Debugf("retrieved image data: %d bytes and content type: %s", len(imageBytes), contentType)

	dataURL := formatDataURL(contentType, imageBytes)

	return dataURL, nil
}

// fetchImageFromRTSPURL fetches the image from the rtsp URL and returns it as a data URL.
func fetchImageFromRTSPURL(ctx context.Context, logger logging.Logger, rtspURL string) (string, error) {
	// Wrap viamrtsp.Config in a resource.Config
	rtspConfig := viamrtsp.Config{
		Address: rtspURL,
	}
	uniqueName := generateUniqueName("tmp-camera")
	resourceConfig := resource.Config{
		Name:                uniqueName,
		API:                 camera.API,
		Model:               viamrtsp.ModelAgnostic,
		ConvertedAttributes: &rtspConfig,
	}

	camera, err := viamrtsp.NewRTSPCamera(ctx, nil, resourceConfig, logger)
	if err != nil {
		return "", fmt.Errorf("failed to create RTSP camera: %w", err)
	}
	defer func() {
		if closeErr := camera.Close(ctx); closeErr != nil {
			logger.Warnf("failed to close camera: %v", closeErr)
		}
	}()

	retryInterval := rtspImageInterval
	timeout := rtspPollTimeout
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()
	timeoutChan := time.After(timeout)
	var imageErr error
	for {
		select {
		case <-ticker.C:
			// Attempt to get an image from the RTSP camera
			img, metadata, err := camera.Image(ctx, imageReqMimeType, nil)
			if err == nil {
				logger.Debugf("Received image with metadata: %v, size: %d bytes", metadata, len(img))
				dataURL := formatDataURL(metadata.MimeType, img)
				return dataURL, nil
			}
			imageErr = err
			logger.Debugf("Failed to get image from RTSP camera: %v", imageErr)
		case <-timeoutChan:
			return "", fmt.Errorf("timeout while trying to get image from RTSP camera: %w", imageErr)
		case <-ctx.Done():
			return "", fmt.Errorf("context canceled while fetching image from RTSP camera: %w", ctx.Err())
		}
	}
}

// generateUniqueName creates a unique name by adding timestamp and random bytes
func generateUniqueName(prefix string) string {
	timestamp := time.Now().UnixNano()
	uniqueName := fmt.Sprintf("%s-%d", prefix, timestamp)
	return uniqueName
}

func createCamerasFromURLs(l CameraInfo, discoveryDependencyName string, logger logging.Logger) ([]resource.Config, error) {
	cams := []resource.Config{}
	for index, u := range l.MediaEndpoints {
		logger.Debugf("camera URL:\t%s", u)

		// Some URLs may contain a hostname that is served by the DiscoveryService's mDNS
		// server. For those that are, we create a config where the dependency is explicitly written
		// down.
		discDep := ""
		if l.urlDependsOnMDNS(index) {
			discDep = discoveryDependencyName
		}

		// Using the camera's Config struct in case a breaking change occurs
		_true := true
		config := viamrtsp.Config{
			Address:        u.StreamURI,
			RTPPassthrough: &_true,
			DiscoveryDep:   discDep,
			Codec:          u.Codec,
			FrameRate:      u.FrameRate,
		}
		if u.Resolution.Width != 0 && u.Resolution.Height != 0 {
			config.Resolution = &viamrtsp.Resolution{
				Width:  u.Resolution.Width,
				Height: u.Resolution.Height,
			}
		}

		cfg, err := createCameraConfig(l.Name(index), config)
		if err != nil {
			return nil, err
		}
		cams = append(cams, cfg)
	}
	return cams, nil
}

func createCameraConfig(name string, attributes viamrtsp.Config) (resource.Config, error) {
	var result map[string]interface{}

	// marshal to bytes
	jsonBytes, err := json.Marshal(attributes)
	if err != nil {
		return resource.Config{}, err
	}

	// convert to map to be used as attributes in resource.Config
	if err = json.Unmarshal(jsonBytes, &result); err != nil {
		return resource.Config{}, err
	}

	return resource.Config{
		Name: name, API: camera.API, Model: viamrtsp.ModelAgnostic,
		Attributes: result, ConvertedAttributes: &attributes,
	}, nil
}

func getCredFromExtra(extra map[string]any) (device.Credentials, bool) {
	// check for a username from extras
	extraUser, ok := extra["User"].(string)
	if !ok {
		return device.Credentials{}, false
	}
	// not requiring a password to match config
	extraPass, ok := extra["Pass"].(string)
	if !ok {
		extraPass = ""
	}

	return device.Credentials{User: extraUser, Pass: extraPass}, true
}
