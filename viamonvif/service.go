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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icholy/digest"
	"github.com/viam-modules/viamrtsp"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

// Model is the model for a onvif discovery service for rtsp cameras.
var (
	Model             = viamrtsp.Family.WithModel("onvif")
	errNoCamerasFound = errors.New("no cameras found, ensure cameras are working or check credentials")
	emptyCred         = device.Credentials{}
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
func (cfg *Config) Validate(_ string) ([]string, error) {
	// check that all creds have usernames set. Note a credential can have both fields empty
	for _, cred := range cfg.Credentials {
		if cred.Pass != "" && cred.User == "" {
			return nil, fmt.Errorf("credential missing username, has password %v", cred.Pass)
		}
	}
	return []string{}, nil
}

type snapshotRequest struct {
	rtspURL string
}

type rtspDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
	Credentials []device.Credentials
	URIs        []URI
	mdnsServer  *mdnsServer
	logger      logging.Logger
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

	return dis, nil
}

// DiscoverResources discovers different rtsp cameras that use onvif.
func (dis *rtspDiscovery) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
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
		return nil, errors.New("no cameras found, ensure cameras are working or check credentials")
	}
	// clear dis.URIs list
	dis.URIs = []URI{}
	for _, camInfo := range list.Cameras {
		dis.logger.Debugf("%s %s %s", camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
		// some cameras return with no urls. explicitly skipping those so the behavior is clear in the service.
		if len(camInfo.URIs) == 0 {
			dis.logger.Errorf("No urls found for camera, skipping. %s %s %s",
				camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
			continue
		}
		dis.URIs = append(dis.URIs, camInfo.URIs...)

		// tryMDNS will attempt to register an mdns entry for the camera. If successfully
		// registered, `tryMDNS` will additionally mutate the `camInfo.RTSPURLs` to use the dns
		// hostname rather than a raw IP. Such that the camera configs we are about to generate will
		// use the dns hostname.
		camInfo.tryMDNS(dis.mdnsServer, dis.logger)

		camConfigs, err := createCamerasFromURLs(camInfo, dis.Name().ShortName(), dis.logger)
		if err != nil {
			return nil, err
		}
		cams = append(cams, camConfigs...)
	}

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
		snapshotReq, err := toSnapshotCommand(command)
		if err != nil {
			return nil, err
		}

		// look up the snapshot uri by the snapshotReq.rtspURL in the list of URIs
		var found bool
		var snapshotURI string
		for _, uri := range dis.URIs {
			if uri.StreamURI == snapshotReq.rtspURL {
				snapshotURI = uri.SnapshotURI
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("snapshot URI not found for %s", snapshotReq.rtspURL)
		}
		dis.logger.Infof("snapshot URI: %s", snapshotURI)

		dataURL, err := downloadPreviewImage(ctx, dis.logger, snapshotURI)
		if err != nil {
			return nil, fmt.Errorf("failed to download preview image: %w", err)
		}
		return map[string]interface{}{
			"preview": dataURL,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

func (dis *rtspDiscovery) Close(_ context.Context) error {
	dis.mdnsServer.Shutdown()
	return nil
}

func toSnapshotCommand(command map[string]interface{}) (*snapshotRequest, error) {
	// First, check if attributes exists and is a map
	attributes, ok := command["attributes"].(map[string]interface{})
	if !ok {
		return nil, errors.New("attributes is missing or not a map")
	}
	rtspURL, ok := attributes["rtsp_address"].(string)
	if !ok {
		return nil, errors.New("invalid snapshot URI")
	}
	return &snapshotRequest{rtspURL: rtspURL}, nil
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
		password, _ = parsedURL.User.Password()
		logger.Debugf("using credentials: username=%s", username)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}
	client := &http.Client{
		// Setting upper bound timeout in case the ctx never times out
		Timeout: 5 * time.Second, //nolint:mnd
		Transport: &digest.Transport{
			Username:  username,
			Password:  password,
			Transport: transport,
		},
	}

	initialReq, err := http.NewRequestWithContext(ctx, http.MethodGet, snapshotURI, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	resp, err := client.Do(initialReq)
	if err != nil {
		return "", fmt.Errorf("failed to execute http request: %w", err)
	}
	defer resp.Body.Close()
	logger.Debugf("snapshot response status: %s", resp.Status)

	// TODO(seanp): Should we log body text in err case?
	if resp.StatusCode != http.StatusOK {
		statusText := http.StatusText(resp.StatusCode)
		return "", fmt.Errorf("failed to get snapshot image, status %d: %s", resp.StatusCode, statusText)
	}

	contentType := resp.Header.Get("Content-Type")
	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data from http response: %w", err)
	}
	logger.Debugf("Retrieved image data: %d bytes and content type: %s", len(imageBytes), contentType)

	dataURL := formatDataURL(contentType, imageBytes)
	logger.Debugf("dataURL: %s", dataURL)

	return dataURL, nil
}

func createCamerasFromURLs(l CameraInfo, discoveryDependencyName string, logger logging.Logger) ([]resource.Config, error) {
	cams := []resource.Config{}
	for index, u := range l.URIs {
		logger.Debugf("camera URL:\t%s", u)

		// Some URLs may contain a hostname that is served by the DiscoveryService's mDNS
		// server. For those that are, we create a config where the dependency is explicitly written
		// down.
		discDep := ""
		if l.urlDependsOnMDNS(index) {
			discDep = discoveryDependencyName
		}

		cfg, err := createCameraConfig(l.Name(index), u.StreamURI, discDep)
		if err != nil {
			return nil, err
		}
		cams = append(cams, cfg)
	}
	return cams, nil
}

func createCameraConfig(name, address, discoveryDependency string) (resource.Config, error) {
	// using the camera's Config struct in case a breaking change occurs
	_true := true
	attributes := viamrtsp.Config{
		Address:        address,
		RTPPassthrough: &_true,
		DiscoveryDep:   discoveryDependency,
	}
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
