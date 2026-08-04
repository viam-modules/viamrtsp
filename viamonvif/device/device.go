// Package device allows communication with an onvif device
// inspired by https://github.com/use-go/onvif
// NOTE(Nick S): This code currently isn't cancellable.
// work needs to be done in order to make it cancellable when
// viam resource Close or Reconfigure are called.
package device

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/viam-modules/viamrtsp/viamonvif/gosoap"
	"github.com/viam-modules/viamrtsp/viamonvif/ptz"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
)

const (
	streamTypeRTPUnicast = "RTP-Unicast"
	streamSetupProtocol  = "RTSP"
)

// Xlmns XML Schema.
var Xlmns = map[string]string{
	"onvif":   "http://www.onvif.org/ver10/schema",
	"tds":     "http://www.onvif.org/ver10/device/wsdl",
	"trt":     "http://www.onvif.org/ver10/media/wsdl",
	"tev":     "http://www.onvif.org/ver10/events/wsdl",
	"tptz":    "http://www.onvif.org/ver20/ptz/wsdl",
	"timg":    "http://www.onvif.org/ver20/imaging/wsdl",
	"tan":     "http://www.onvif.org/ver20/analytics/wsdl",
	"xmime":   "http://www.w3.org/2005/05/xmlmime",
	"wsnt":    "http://docs.oasis-open.org/wsn/b-2",
	"xop":     "http://www.w3.org/2004/08/xop/include",
	"wsa":     "http://www.w3.org/2005/08/addressing",
	"wstop":   "http://docs.oasis-open.org/wsn/t-1",
	"wsntw":   "http://docs.oasis-open.org/wsn/bw-2",
	"wsrf-rw": "http://docs.oasis-open.org/wsrf/rw-2",
	"wsaw":    "http://www.w3.org/2006/05/addressing/wsdl",
}

// Device for a new device of onvif and DeviceInfo
// struct represents an abstract ONVIF device.
// It contains methods, which helps to communicate with ONVIF device.
type Device struct {
	xaddr     *url.URL
	logger    logging.Logger
	params    Params
	endpoints map[string]string
	// clockOffset is the device's clock minus the local clock. WS-Security Created
	// timestamps are generated in the device's time frame so that devices with wrong
	// system dates (e.g. a default of 2000-01-01) do not reject them as stale.
	clockOffset time.Duration
}

// Params configures the device connection.
type Params struct {
	Xaddr      *url.URL
	Username   string
	Password   string
	HTTPClient *http.Client
	// SkipLocalTLSVerification controls whether TLS certificate verification is skipped for local IP addresses.
	// This is necessary for cameras with self-signed certificates.
	SkipLocalTLSVerification bool
}

// GetProfiles is a request to the GetProfiles onvif endpoint.
type GetProfiles struct {
	XMLName string `xml:"trt:GetProfiles"`
}

// GetStreamURI is a request to the GetStreamURI onvif endpoint.
type GetStreamURI struct {
	XMLName      string               `xml:"trt:GetStreamUri"`
	StreamSetup  onvif.StreamSetup    `xml:"trt:StreamSetup"`
	ProfileToken onvif.ReferenceToken `xml:"trt:ProfileToken"`
}

// GetSnapshotURI is a request to the GetSnapshotUri onvif endpoint.
type GetSnapshotURI struct {
	XMLName      string               `xml:"trt:GetSnapshotUri"`
	ProfileToken onvif.ReferenceToken `xml:"trt:ProfileToken"`
}

// GetDeviceInformation is a request to the GetDeviceInformation onvif endpoint.
type GetDeviceInformation struct {
	XMLName string `xml:"tds:GetDeviceInformation"`
}

// GetNetworkInterfaces is a request to the GetNetworkInterfaces onvif endpoint.
type GetNetworkInterfaces struct {
	XMLName string `xml:"tds:GetNetworkInterfaces"`
}

// GetCapabilities is a request to the GetCapabilities onvif endpoint.
type GetCapabilities struct {
	XMLName  string                   `xml:"tds:GetCapabilities"`
	Category onvif.CapabilityCategory `xml:"tds:Category"`
}

// GetSystemDateAndTime is a request to the GetSystemDateAndTime onvif endpoint.
// Per the ONVIF Core Spec this endpoint must be available without authentication.
type GetSystemDateAndTime struct {
	XMLName string `xml:"tds:GetSystemDateAndTime"`
}

// NewDevice construct an ONVIF Device entity.
func NewDevice(ctx context.Context, params Params, logger logging.Logger) (*Device, error) {
	dev := &Device{
		xaddr:     params.Xaddr,
		logger:    logger,
		params:    params,
		endpoints: map[string]string{"device": params.Xaddr.String()},
	}

	if dev.params.HTTPClient == nil {
		var skipVerify bool
		if params.SkipLocalTLSVerification {
			hostname := strings.ToLower(params.Xaddr.Hostname())
			if strings.HasSuffix(hostname, ".local") {
				skipVerify = true
			} else if ip, err := netip.ParseAddr(hostname); err == nil {
				skipVerify = ip.IsPrivate() || ip.IsLoopback()
			}
		}
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipVerify, //nolint:gosec
			},
		}
		dev.params.HTTPClient = &http.Client{
			Transport: transport,
		}

		if skipVerify {
			logger.Debugf("TLS certificate verification disabled for local IP address: %s.",
				params.Xaddr.Hostname())
		}
	}

	// Sync with the device's clock before making any authenticated calls. The
	// WS-Security password digest embeds a Created timestamp that devices validate
	// against their own clock, so a device with a wrong system date (e.g. a default
	// of 2000-01-01) rejects tokens generated from the local clock. Only needed
	// when credentials will be sent.
	if params.Username != "" || params.Password != "" {
		dev.syncClock(ctx)
	}

	data, err := dev.callDevice(ctx, GetCapabilities{Category: "All"})
	if err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, err
	}
	dev.logger.Debugf("GetCapabilitiesResponse: %s", string(data))
	services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/*/XAddr")
	for i, s := range services {
		if i == 0 {
			dev.logger.Debug("GetCapabilities services:")
		}
		dev.logger.Debugf("%s: %s", s.Parent().Tag, s.Text())
		dev.endpoints[strings.ToLower(s.Parent().Tag)] = s.Text()
	}
	extensionServices := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/Extension/*/XAddr")
	for i, s := range extensionServices {
		if i == 0 {
			dev.logger.Debug("GetCapabilities extension services:")
		}
		dev.logger.Debugf("%s: %s", s.Parent().Tag, s.Text())
		dev.endpoints[strings.ToLower(s.Parent().Tag)] = s.Text()
	}

	return dev, nil
}

// GetDeviceInformationResponse is the response to GetDeviceInformation.
type GetDeviceInformationResponse struct {
	Manufacturer    string `xml:"Manufacturer"`
	Model           string `xml:"Model"`
	FirmwareVersion string `xml:"FirmwareVersion"`
	SerialNumber    string `xml:"SerialNumber"`
	HardwareID      string `xml:"HardwareId"`
}

// GetDeviceInformationResponseEnvelope is the envelope of the GetDeviceInformationResponse.
type GetDeviceInformationResponseEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetDeviceInformationResponse GetDeviceInformationResponse `xml:"GetDeviceInformationResponse"`
	} `xml:"Body"`
}

// GetDeviceInformation returns device information.
func (dev *Device) GetDeviceInformation(ctx context.Context) (GetDeviceInformationResponse, error) {
	var zero GetDeviceInformationResponse
	b, err := dev.callOnvifServiceMethod(ctx, dev.endpoints["device"], GetDeviceInformation{})
	if err != nil {
		return zero, fmt.Errorf("failed to get device information: %w", err)
	}
	dev.logger.Debugf("GetDeviceInformation response body: %s", string(b))

	var resp GetDeviceInformationResponseEnvelope
	err = xml.NewDecoder(bytes.NewReader(b)).Decode(&resp)
	if err != nil {
		return zero, fmt.Errorf("failed to decode device information response: %w", err)
	}
	dev.logger.Debugf("GetDeviceInformation decoded: %#v", resp.Body.GetDeviceInformationResponse)
	return resp.Body.GetDeviceInformationResponse, nil
}

// onvifDate is a calendar date in a GetSystemDateAndTime response.
type onvifDate struct {
	Year  int `xml:"Year"`
	Month int `xml:"Month"`
	Day   int `xml:"Day"`
}

// onvifTime is a wall-clock time in a GetSystemDateAndTime response.
type onvifTime struct {
	Hour   int `xml:"Hour"`
	Minute int `xml:"Minute"`
	Second int `xml:"Second"`
}

// onvifDateTime is a date and time in a GetSystemDateAndTime response.
type onvifDateTime struct {
	Time onvifTime `xml:"Time"`
	Date onvifDate `xml:"Date"`
}

// toTime converts an onvifDateTime to a time.Time in UTC. The second return
// value is false when the element was absent from the response (zero-valued).
func (dt onvifDateTime) toTime() (time.Time, bool) {
	if dt.Date.Year == 0 {
		return time.Time{}, false
	}
	return time.Date(dt.Date.Year, time.Month(dt.Date.Month), dt.Date.Day,
		dt.Time.Hour, dt.Time.Minute, dt.Time.Second, 0, time.UTC), true
}

// getSystemDateAndTimeResponseEnvelope is the envelope of the response to the
// GetSystemDateAndTime endpoint.
type getSystemDateAndTimeResponseEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetSystemDateAndTimeResponse struct {
			SystemDateAndTime struct {
				UTCDateTime   onvifDateTime `xml:"UTCDateTime"`
				LocalDateTime onvifDateTime `xml:"LocalDateTime"`
			} `xml:"SystemDateAndTime"`
		} `xml:"GetSystemDateAndTimeResponse"`
	} `xml:"Body"`
}

// parseSystemDateAndTimeResponse extracts the device's clock reading from a
// GetSystemDateAndTime response body. UTCDateTime is preferred; some devices
// omit it, in which case LocalDateTime is treated as UTC — an approximate
// offset still beats none for clock-skew compensation.
func parseSystemDateAndTimeResponse(b []byte) (time.Time, error) {
	var zero time.Time
	var resp getSystemDateAndTimeResponseEnvelope
	if err := xml.NewDecoder(bytes.NewReader(b)).Decode(&resp); err != nil {
		return zero, fmt.Errorf("failed to decode system date and time response: %w", err)
	}
	sdt := resp.Body.GetSystemDateAndTimeResponse.SystemDateAndTime
	if t, ok := sdt.UTCDateTime.toTime(); ok {
		return t, nil
	}
	if t, ok := sdt.LocalDateTime.toTime(); ok {
		return t, nil
	}
	return zero, errors.New("no date and time found in GetSystemDateAndTime response")
}

// getSystemDateAndTime returns the device's current clock reading. It is sent
// without a WS-Security header: the ONVIF Core Spec requires this endpoint to
// work unauthenticated, and on a clock-skewed device an authenticated request
// would be rejected for exactly the skew this call exists to measure.
func (dev *Device) getSystemDateAndTime(ctx context.Context) (time.Time, error) {
	var zero time.Time
	b, err := dev.callOnvifServiceMethodNoAuth(ctx, dev.endpoints["device"], GetSystemDateAndTime{})
	if err != nil {
		return zero, fmt.Errorf("failed to get system date and time: %w", err)
	}
	dev.logger.Debugf("GetSystemDateAndTime response body: %s", string(b))
	return parseSystemDateAndTimeResponse(b)
}

// syncClock measures the offset between the device's clock and the local clock
// so authenticated requests can be timestamped in the device's time frame. Any
// failure leaves the offset at zero, i.e. assumes synchronized clocks.
func (dev *Device) syncClock(ctx context.Context) {
	deviceTime, err := dev.getSystemDateAndTime(ctx)
	if err != nil {
		dev.logger.Debugf("GetSystemDateAndTime failed, assuming device clock is synchronized with local clock: %v", err)
		return
	}
	offset := deviceTime.Sub(time.Now().UTC())
	if offset.Abs() > time.Minute {
		dev.logger.Infof("device %s clock differs from local clock by %v; using device time for authentication",
			dev.xaddr.Host, offset)
	}
	dev.clockOffset = offset
}

// GetNetworkInterfacesResponse is the body of the response to the GetNetworkInterfaces endpoint.
type GetNetworkInterfacesResponse struct {
	NetworkInterfaces []onvif.NetworkInterface `xml:"NetworkInterfaces"`
}

// GetNetworkInterfacesResponseEnvelope is the envelope of the GetNetworkInterfacesResponse.
type GetNetworkInterfacesResponseEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetNetworkInterfacesResponse GetNetworkInterfacesResponse `xml:"GetNetworkInterfacesResponse"`
	} `xml:"Body"`
}

// GetNetworkInterfaces returns the device's network interfaces, including MAC addresses.
func (dev *Device) GetNetworkInterfaces(ctx context.Context) (GetNetworkInterfacesResponse, error) {
	var zero GetNetworkInterfacesResponse
	b, err := dev.callOnvifServiceMethod(ctx, dev.endpoints["device"], GetNetworkInterfaces{})
	if err != nil {
		return zero, fmt.Errorf("failed to get network interfaces: %w", err)
	}
	dev.logger.Debugf("GetNetworkInterfaces response body: %s", string(b))

	var resp GetNetworkInterfacesResponseEnvelope
	err = xml.NewDecoder(bytes.NewReader(b)).Decode(&resp)
	if err != nil {
		return zero, fmt.Errorf("failed to decode network interfaces response: %w", err)
	}
	dev.logger.Debugf("GetNetworkInterfaces decoded: %#v", resp.Body.GetNetworkInterfacesResponse)
	return resp.Body.GetNetworkInterfacesResponse, nil
}

// GetProfilesResponse is the body of the response to the GetProfiles endpoint.
type GetProfilesResponse struct {
	Profiles []onvif.Profile `xml:"Profiles"`
}

// GetProfilesResponseEnvelope is the envelope of the response to the GetProfiles endpoint.
type GetProfilesResponseEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetProfilesResponse GetProfilesResponse `xml:"GetProfilesResponse"`
	} `xml:"Body"`
}

// GetProfiles returns the device's profiles.
func (dev *Device) GetProfiles(ctx context.Context) (GetProfilesResponse, error) {
	var zero GetProfilesResponse
	getProfiles := GetProfiles{}
	b, err := dev.callMedia(ctx, getProfiles)
	if err != nil {
		return zero, fmt.Errorf("failed to get media profiles: %w", err)
	}

	dev.logger.Debugf("GetProfiles response body: %s", b)
	var resp GetProfilesResponseEnvelope
	err = xml.NewDecoder(bytes.NewReader(b)).Decode(&resp)
	if err != nil {
		return zero, fmt.Errorf("failed to decode media profiles response: %w", err)
	}

	if len(resp.Body.GetProfilesResponse.Profiles) == 0 {
		dev.logger.Warn("No media profiles found in the response")
		return zero, errors.New("no media profiles found")
	}

	dev.logger.Debugf("Found %d media profiles", len(resp.Body.GetProfilesResponse.Profiles))
	for i, profile := range resp.Body.GetProfilesResponse.Profiles {
		dev.logger.Debugf("Profile %d: Token=%s, Name=%s, FrameRate=%d, Resolution=%dx%d, Codec=%s",
			i,
			profile.Token,
			profile.Name,
			int(profile.VideoEncoderConfiguration.RateControl.FrameRateLimit),
			profile.VideoEncoderConfiguration.Resolution.Width,
			profile.VideoEncoderConfiguration.Resolution.Height,
			string(profile.VideoEncoderConfiguration.Encoding))
	}

	dev.logger.Debugf("GetProfiles decoded: %#v", resp.Body.GetProfilesResponse)
	return resp.Body.GetProfilesResponse, nil
}

type getStreamURIResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetStreamURIResponse struct {
			MediaURI onvif.MediaUri `xml:"MediaUri"`
		} `xml:"GetStreamUriResponse"`
	} `xml:"Body"`
}

type getSnapshotURIResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetSnapshotURIResponse struct {
			MediaURI onvif.MediaUri `xml:"MediaUri"`
		} `xml:"GetSnapshotUriResponse"`
	} `xml:"Body"`
}

// Credentials contain an onvif device username and password.
type Credentials struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

// GetStreamURI returns a device's stream URI for a given profile token.
func (dev *Device) GetStreamURI(ctx context.Context, token onvif.ReferenceToken, creds Credentials) (*url.URL, error) {
	dev.logger.Debugf("GetStreamUri token: %s", token)
	body, err := dev.callMedia(ctx, GetStreamURI{
		StreamSetup: onvif.StreamSetup{
			Stream:    onvif.StreamType(streamTypeRTPUnicast),
			Transport: onvif.Transport{Protocol: streamSetupProtocol},
		},
		ProfileToken: token,
	})
	if err != nil {
		return nil, err
	}
	dev.logger.Debugf("GetStreamUri response: %s", string(body))

	var streamURI getStreamURIResponse
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&streamURI); err != nil {
		return nil, fmt.Errorf("failed to get RTSP URL for token %s: %w", token, err)
	}
	dev.logger.Debugf("GetStreamUriResponse decoded for token %s, streamURI: %v ", token, streamURI)

	uriStr := string(streamURI.Body.GetStreamURIResponse.MediaURI.Uri)
	if uriStr == "" {
		return nil, fmt.Errorf("got empty stream uri for token %s", token)
	}

	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI %s: %w", uriStr, err)
	}

	if creds.User != "" || creds.Pass != "" {
		uri.User = url.UserPassword(creds.User, creds.Pass)
	}
	dev.logger.Debugf("GetStreamUriResponse parsed for token %s: %s", token, uri.String())

	return uri, nil
}

// GetSnapshotURI returns a device's snapshot URI for a given profile token.
func (dev *Device) GetSnapshotURI(ctx context.Context, token onvif.ReferenceToken, creds Credentials) (*url.URL, error) {
	dev.logger.Debugf("GetSnapshotUri token: %s", token)
	body, err := dev.callMedia(ctx, GetSnapshotURI{
		ProfileToken: token,
	})
	if err != nil {
		return nil, err
	}
	dev.logger.Debugf("GetSnapshotUri response: %v", string(body))
	var snapshotURI getSnapshotURIResponse
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&snapshotURI); err != nil {
		return nil, fmt.Errorf("failed to get snapshot URL for token %s: %w", token, err)
	}
	dev.logger.Debugf("getSnapshotUriResponse decoded for token %s, snapshotURI: %v", token, snapshotURI)

	uriStr := string(snapshotURI.Body.GetSnapshotURIResponse.MediaURI.Uri)
	if uriStr == "" {
		return nil, fmt.Errorf("got empty snapshot uri for token %s", token)
	}
	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI %s: %w", uriStr, err)
	}
	if creds.User != "" || creds.Pass != "" {
		uri.User = url.UserPassword(creds.User, creds.Pass)
	}
	dev.logger.Debugf("GetSnapshotUriResponse parsed for token %s: %s", token, uri.String())

	return uri, nil
}

// GetEndpoint returns specific ONVIF service endpoint address.
func (dev *Device) GetEndpoint(name string) string {
	return dev.endpoints[name]
}

func (dev *Device) callMedia(ctx context.Context, method interface{}) ([]byte, error) {
	return dev.callOnvifServiceMethod(ctx, dev.endpoints["media"], method)
}

func (dev *Device) callDevice(ctx context.Context, method interface{}) ([]byte, error) {
	return dev.callOnvifServiceMethod(ctx, dev.endpoints["device"], method)
}

func (dev *Device) callOnvifServiceMethod(ctx context.Context, endpoint string, method interface{}) ([]byte, error) {
	return dev.call(ctx, endpoint, method, true)
}

// callOnvifServiceMethodNoAuth calls an ONVIF service method without a WS-Security
// header, for endpoints the spec defines as pre-auth (e.g. GetSystemDateAndTime).
func (dev *Device) callOnvifServiceMethodNoAuth(ctx context.Context, endpoint string, method interface{}) ([]byte, error) {
	return dev.call(ctx, endpoint, method, false)
}

func (dev *Device) call(ctx context.Context, endpoint string, method interface{}, authenticate bool) ([]byte, error) {
	output, err := xml.MarshalIndent(method, "  ", "    ")
	if err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(output); err != nil {
		return nil, err
	}

	soap, err := gosoap.NewEmptySOAP()
	if err != nil {
		return nil, err
	}

	if err := soap.AddBodyContent(doc.Root()); err != nil {
		return nil, err
	}
	for key, value := range Xlmns {
		if err := soap.AddRootNamespace(key, value); err != nil {
			return nil, err
		}
	}
	if err := soap.AddAction(); err != nil {
		return nil, err
	}

	if authenticate && (dev.params.Username != "" || dev.params.Password != "") {
		if err := soap.AddWSSecurity(dev.params.Username, dev.params.Password, dev.clockOffset); err != nil {
			return nil, err
		}
	}

	return dev.sendSoap(ctx, endpoint, soap.String())
}

func (dev *Device) sendSoap(ctx context.Context, endpoint, message string) ([]byte, error) {
	contentType := "application/soap+xml; charset=utf-8"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(message))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	// Using Do instead of POST to support context cancellation and timeout.
	resp, err := dev.params.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Include the (truncated) response body: devices report SOAP faults such as
		// ter:NotAuthorized there, which is often the only clue to auth failures.
		const maxErrBodyLen = 4096
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodyLen))
		if len(errBody) > 0 {
			return nil, fmt.Errorf("SOAP request to %s failed with status code: %d, body: %s", endpoint, resp.StatusCode, errBody)
		}
		return nil, fmt.Errorf("SOAP request to %s failed with status code: %d", endpoint, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// GetXaddr returns the URL of the Onvif web service.
func (dev *Device) GetXaddr() *url.URL {
	if dev.xaddr == nil {
		return nil
	}
	return &url.URL{
		Scheme: dev.xaddr.Scheme,
		Host:   dev.xaddr.Host,
		Path:   dev.xaddr.Path,
	}
}

// GetPTZNodes returns a list of PTZ nodes supported by the device.
// Includes complete information about each node's movement capabilities.
func (dev *Device) GetPTZNodes(ctx context.Context) ([]onvif.PTZNode, error) {
	req := ptz.GetNodes{}
	data, err := dev.callOnvifServiceMethod(ctx, dev.endpoints["ptz"], req)
	if err != nil {
		return nil, fmt.Errorf("GetNodes failed: %w", err)
	}
	dev.logger.Debugf("GetPTZNodes response body: %s", string(data))

	var env onvif.GetNodesEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal GetNodesResponse: %w", err)
	}

	return env.Body.GetNodesResponse.Nodes, nil
}

// CallPTZMethod calls a PTZ service method and returns the raw response bytes.
func (dev *Device) CallPTZMethod(ctx context.Context, method interface{}) ([]byte, error) {
	return dev.callOnvifServiceMethod(ctx, dev.endpoints["ptz"], method)
}
