// Package device allows communication with an onvif device
// inspired by https://github.com/use-go/onvif
// NOTE(Nick S): This code currently isn't cancellable.
// work needs to be done in order to make it cancellable when
// viam resource Close or Reconfigure are called.
package device

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/beevik/etree"
	"github.com/viam-modules/viamrtsp/viamonvif/gosoap"
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
}

// Params configures the device connection.
type Params struct {
	Xaddr      *url.URL
	Username   string
	Password   string
	HTTPClient *http.Client
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

// GetDeviceInformation is a request to the GetDeviceInformation onvif endpoint.
type GetDeviceInformation struct {
	XMLName string `xml:"tds:GetDeviceInformation"`
}

// GetCapabilities is a request to the GetCapabilities onvif endpoint.
type GetCapabilities struct {
	XMLName  string                   `xml:"tds:GetCapabilities"`
	Category onvif.CapabilityCategory `xml:"tds:Category"`
}

// NewDevice function construct a ONVIF Device entity.
func NewDevice(params Params, logger logging.Logger) (*Device, error) {
	dev := &Device{
		xaddr:     params.Xaddr,
		logger:    logger,
		params:    params,
		endpoints: map[string]string{"device": params.Xaddr.String()},
	}

	if dev.params.HTTPClient == nil {
		dev.params.HTTPClient = new(http.Client)
	}

	data, err := dev.callDevice(GetCapabilities{Category: "All"})
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
func (dev *Device) GetDeviceInformation() (GetDeviceInformationResponse, error) {
	var zero GetDeviceInformationResponse
	b, err := dev.callOnvifServiceMethod(dev.endpoints["device"], GetDeviceInformation{})
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
func (dev *Device) GetProfiles() (GetProfilesResponse, error) {
	var zero GetProfilesResponse
	getProfiles := GetProfiles{}
	b, err := dev.callMedia(getProfiles)
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
		dev.logger.Debugf("Profile %d: Token=%s, Name=%s", i, profile.Token, profile.Name)
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

// Credentials contain an onvif device username and password.
type Credentials struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

// GetStreamURI returns a device's stream URI for a given profile.
func (dev *Device) GetStreamURI(profile onvif.Profile, creds Credentials) (*url.URL, error) {
	dev.logger.Debugf("GetStreamUri token: %s, profile: %#v", profile.Token, profile)
	body, err := dev.callMedia(GetStreamURI{
		StreamSetup: onvif.StreamSetup{
			Stream:    onvif.StreamType(streamTypeRTPUnicast),
			Transport: onvif.Transport{Protocol: streamSetupProtocol},
		},
		ProfileToken: profile.Token,
	})
	if err != nil {
		return nil, err
	}
	dev.logger.Debugf("GetStreamUri response: %v", string(body))

	var streamURI getStreamURIResponse
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&streamURI); err != nil {
		return nil, fmt.Errorf("failed to get RTSP URL for profile %s: %w", profile.Token, err)
	}

	dev.logger.Debugf("GetStreamUriResponse decoded %v: ", profile.Token, streamURI)

	uriStr := string(streamURI.Body.GetStreamURIResponse.MediaURI.Uri)
	if uriStr == "" {
		return nil, fmt.Errorf("got empty uri for profile %s", profile.Token)
	}

	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI %s: %w", uriStr, err)
	}

	if creds.User != "" || creds.Pass != "" {
		uri.User = url.UserPassword(creds.User, creds.Pass)
	}
	return uri, nil
}

// GetEndpoint returns specific ONVIF service endpoint address.
func (dev *Device) GetEndpoint(name string) string {
	return dev.endpoints[name]
}

func (dev Device) callMedia(method interface{}) ([]byte, error) {
	return dev.callOnvifServiceMethod(dev.endpoints["media"], method)
}

func (dev Device) callDevice(method interface{}) ([]byte, error) {
	return dev.callOnvifServiceMethod(dev.endpoints["device"], method)
}

func (dev Device) callOnvifServiceMethod(endpoint string, method interface{}) ([]byte, error) {
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

	if dev.params.Username != "" || dev.params.Password != "" {
		if err := soap.AddWSSecurity(dev.params.Username, dev.params.Password); err != nil {
			return nil, err
		}
	}

	return dev.sendSoap(endpoint, soap.String())
}

func (dev *Device) sendSoap(endpoint, message string) ([]byte, error) {
	contentType := "application/soap+xml; charset=utf-8"
	//nolint: noctx
	// TODO(Nick S): This is pretty bad as it can cause the goroutine calling this to hang forever
	// this should be converted to the http interface that takes a context
	resp, err := dev.params.HTTPClient.Post(endpoint, contentType, bytes.NewBufferString(message))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SOAP request to %s failed with status code: %d", endpoint, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
