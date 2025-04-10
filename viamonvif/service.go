// Package viamonvif provides the discovery service that wraps ONVIF integration for the viamrtsp module
package viamonvif

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
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

type rtspDiscovery struct {
	resource.Named
	resource.AlwaysRebuild
	Credentials []device.Credentials
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

	for _, camInfo := range list.Cameras {
		dis.logger.Debugf("%s %s %s", camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
		// some cameras return with no urls. explicitly skipping those so the behavior is clear in the service.
		if len(camInfo.RTSPURLs) == 0 {
			dis.logger.Errorf("No urls found for camera, skipping. %s %s %s",
				camInfo.Manufacturer, camInfo.Model, camInfo.SerialNumber)
			continue
		}

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

func (rc *rtspDiscovery) DoCommand(ctx context.Context, command map[string]interface{}) (map[string]interface{}, error) {
	cmd, ok := command["command"].(string)
	if !ok {
		return nil, errors.New("invalid command type")
	}

	switch cmd {
	case "snapshot":
		rc.logger.Debugf("snapshot command received")
		snapshotReq, err := toSnapshotCommand(command)
		if err != nil {
			return nil, err
		}

		// Parse the URL to extract credentials
		parsedURL, err := url.Parse(snapshotReq.snapshotURI)
		if err != nil {
			return nil, fmt.Errorf("invalid snapshot URI: %w", err)
		}

		// Extract credentials
		var username, password string
		if parsedURL.User != nil {
			username = parsedURL.User.Username()
			password, _ = parsedURL.User.Password()
			rc.logger.Debugf("Using credentials: username=%s", username)
		}

		// Create HTTP client
		client := &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		// Make initial request to get auth challenge
		initialReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, snapshotReq.snapshotURI, nil)
		resp, err := client.Do(initialReq)
		if err != nil {
			return nil, fmt.Errorf("failed to make initial request: %w", err)
		}

		// If we get 401 with WWW-Authenticate header, handle digest auth
		if resp.StatusCode == http.StatusUnauthorized {
			rc.logger.Debugf("Got 401, handling digest authentication")
			authHeader := resp.Header.Get("WWW-Authenticate")
			resp.Body.Close()

			digestParts := digestAuthParams(authHeader)

			// Calculate digest response
			ha1 := md5hex(username + ":" + digestParts["realm"] + ":" + password)
			ha2 := md5hex("GET" + ":" + parsedURL.RequestURI())
			nonceCount := "00000001"
			cnonce := randHex(16)

			response := md5hex(ha1 + ":" + digestParts["nonce"] + ":" +
				nonceCount + ":" + cnonce + ":" + digestParts["qop"] + ":" + ha2)

			// Build authorization header
			authValue := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", algorithm=%s, qop=%s, nc=%s, cnonce="%s", response="%s"`,
				username, digestParts["realm"], digestParts["nonce"], parsedURL.RequestURI(),
				digestParts["algorithm"], digestParts["qop"], nonceCount, cnonce, response)

			authReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, snapshotReq.snapshotURI, nil)
			authReq.Header.Set("Authorization", authValue)

			resp, err = client.Do(authReq)
			if err != nil {
				return nil, fmt.Errorf("failed to make authenticated request: %w", err)
			}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read image data: %w", err)
		}

		base64Image := base64.StdEncoding.EncodeToString(imageBytes)

		return map[string]interface{}{
			"snapshot":     base64Image,
			"content_type": resp.Header.Get("Content-Type"),
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

func digestAuthParams(header string) map[string]string {
	parts := strings.Split(header, " ")
	if len(parts) < 2 || !strings.EqualFold(parts[0], "digest") {
		return nil
	}

	result := make(map[string]string)
	headerVal := strings.Join(parts[1:], " ")

	for _, part := range strings.Split(headerVal, ",") {
		part = strings.TrimSpace(part)
		subparts := strings.SplitN(part, "=", 2)
		if len(subparts) != 2 {
			continue
		}

		key := subparts[0]
		value := strings.Trim(subparts[1], "\"")
		result[key] = value
	}

	if _, ok := result["algorithm"]; !ok {
		result["algorithm"] = "MD5" // Default algorithm
	}

	return result
}

func md5hex(data string) string {
	h := md5.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func randHex(n int) string {
	bytes := make([]byte, n/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type snapshotRequest struct {
	snapshotURI string
}

func toSnapshotCommand(command map[string]interface{}) (*snapshotRequest, error) {
	snapshotURI, ok := command["snapshot_uri"].(string)
	if !ok {
		return nil, errors.New("invalid snapshot URI")
	}
	return &snapshotRequest{snapshotURI: snapshotURI}, nil
}

func (dis *rtspDiscovery) Close(_ context.Context) error {
	dis.mdnsServer.Shutdown()
	return nil
}

func createCamerasFromURLs(l CameraInfo, discoveryDependencyName string, logger logging.Logger) ([]resource.Config, error) {
	if len(l.RTSPURLs) != len(l.SnapshotURIs) {
		logger.Warnf("mismatched lengths: %d RTSP URLs and %d Snapshot URIs, some cameras may not have snapshot URIs",
			len(l.RTSPURLs), len(l.SnapshotURIs))
	}

	cams := []resource.Config{}
	for index, u := range l.RTSPURLs {
		logger.Debugf("camera URL:\t%s", u)

		// Some URLs may contain a hostname that is served by the DiscoveryService's mDNS
		// server. For those that are, we create a config where the dependency is explicitly written
		// down.
		discDep := ""
		if l.urlDependsOnMDNS(index) {
			discDep = discoveryDependencyName
		}

		snapshotURI := ""
		if index < len(l.SnapshotURIs) {
			snapshotURI = l.SnapshotURIs[index]
		}

		cfg, err := createCameraConfig(l.Name(index), u, snapshotURI, discDep)
		if err != nil {
			return nil, err
		}
		cams = append(cams, cfg)
	}
	return cams, nil
}

func createCameraConfig(name, address, uri, discoveryDependency string) (resource.Config, error) {
	// using the camera's Config struct in case a breaking change occurs
	_true := true
	attributes := viamrtsp.Config{
		Address:        address,
		RTPPassthrough: &_true,
		SnapshotURI:    uri,
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
