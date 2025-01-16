// inspired by https://github.com/use-go/onvif
package device

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/viam-modules/viamrtsp/viamonvif/gosoap"
	"github.com/viam-modules/viamrtsp/viamonvif/media"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
)

const (
	streamTypeRTPUnicast = "RTP-Unicast"
	streamSetupProtocol  = "RTSP"
)

// Xlmns XML Scheam
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

// DeviceType alias for int
type DeviceType int

// Onvif Device Tyoe
const (
	NVD DeviceType = iota
	NVS
	NVA
	NVT
)

func (devType DeviceType) String() string {
	stringRepresentation := []string{
		"NetworkVideoDisplay",
		"NetworkVideoStorage",
		"NetworkVideoAnalytics",
		"NetworkVideoTransmitter",
	}
	i := uint8(devType)
	switch {
	case i <= uint8(NVT):
		return stringRepresentation[i]
	default:
		return strconv.Itoa(int(i))
	}
}

// Device for a new device of onvif and DeviceInfo
// struct represents an abstract ONVIF device.
// It contains methods, which helps to communicate with ONVIF device
type Device struct {
	xaddr     *url.URL
	logger    logging.Logger
	params    DeviceParams
	endpoints map[string]string
}

type DeviceParams struct {
	Xaddr      *url.URL
	Username   string
	Password   string
	HttpClient *http.Client
}

// NewDevice function construct a ONVIF Device entity
func NewDevice(params DeviceParams, logger logging.Logger) (*Device, error) {
	dev := &Device{
		xaddr:     params.Xaddr,
		logger:    logger,
		params:    params,
		endpoints: map[string]string{"device": params.Xaddr.String()},
	}

	if dev.params.HttpClient == nil {
		dev.params.HttpClient = new(http.Client)
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
	extension_services := doc.FindElements("./Envelope/Body/GetCapabilitiesResponse/Capabilities/Extension/*/XAddr")
	for i, s := range extension_services {
		if i == 0 {
			dev.logger.Debug("GetCapabilities extension services:")
		}
		dev.logger.Debugf("%s: %s", s.Parent().Tag, s.Text())
		dev.endpoints[strings.ToLower(s.Parent().Tag)] = s.Text()
	}

	return dev, nil
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

func (dev *Device) GetDeviceInformation() (DeviceInfo, error) {
	var zero DeviceInfo
	b, err := dev.callMethodDo(dev.endpoints["device"], GetDeviceInformation{})
	if err != nil {
		return zero, fmt.Errorf("failed to get device information: %w", err)
	}
	dev.logger.Debugf("GetDeviceInformation response body: %s", string(b))

	var resp DeviceInfo
	err = xml.NewDecoder(bytes.NewReader(b)).Decode(&resp)
	if err != nil {
		return zero, fmt.Errorf("failed to decode device information response: %w", err)
	}
	return resp, nil
}

// GetProfilesResponse is the schema the GetProfiles response is formatted in.
type GetProfilesResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetProfilesResponse struct {
			Profiles []onvif.Profile `xml:"Profiles"`
		} `xml:"GetProfilesResponse"`
	} `xml:"Body"`
}

func (dev *Device) GetProfiles() (GetProfilesResponse, error) {
	var zero GetProfilesResponse
	getProfiles := media.GetProfiles{}
	b, err := dev.callMedia(getProfiles)
	if err != nil {
		return zero, fmt.Errorf("failed to get media profiles: %w", err)
	}

	dev.logger.Debugf("GetProfiles response body: %s", b)
	var resp GetProfilesResponse
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

	return resp, nil
}

// getStreamURIResponse is the schema the GetStreamUri response is formatted in.
type getStreamURIResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		GetStreamURIResponse struct {
			MediaURI onvif.MediaUri `xml:"MediaUri"`
		} `xml:"GetStreamUriResponse"`
	} `xml:"Body"`
}

type Credentials struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

func (dev *Device) GetStreamUri(profile onvif.Profile, creds Credentials) (*url.URL, error) {
	dev.logger.Debugf("Using profile token and profile: %s %#v", profile.Token, profile)
	body, err := dev.callMedia(media.GetStreamUri{
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
		return nil, fmt.Errorf("Failed to get RTSP URL for profile %s: %v", profile.Token, err)
	}

	dev.logger.Debugf("stream uri response for profile %s: %v: ", profile.Token, streamURI)

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

// func (dev *Device) addEndpoint(Key, Value string) {
// 	// Replace host with host from device params.
// 	// if u, err := url.Parse(Value); err == nil {
// 	// 	u.Host = dev.host
// 	// 	Value = u.String()
// 	// }

// 	dev.endpoints[strings.ToLower(Key)] = Value
// }

// GetEndpoint returns specific ONVIF service endpoint address
func (dev *Device) GetEndpoint(name string) string {
	return dev.endpoints[name]
}

// getEndpoint functions get the target service endpoint in a better way
func (dev Device) getEndpoint(endpoint string) (string, error) {
	// common condition, endpointMark in map we use this.
	if endpointURL, ok := dev.endpoints[endpoint]; ok {
		return endpointURL, nil
	}

	//but ,if we have endpoint like event、analytic
	//and sametime the Targetkey like : events、analytics
	//we use fuzzy way to find the best match url
	var endpointURL string
	for targetKey := range dev.endpoints {
		if strings.Contains(targetKey, endpoint) {
			endpointURL = dev.endpoints[targetKey]
			return endpointURL, nil
		}
	}
	return endpointURL, errors.New("target endpoint service not found")
}

// CallMethod functions call an method, defined <method> struct.
// You should use Authenticate method to call authorized requests.
// func (dev Device) CallMethod(method interface{}) ([]byte, error) {
// 	pkgPath := strings.Split(reflect.TypeOf(method).PkgPath(), "/")
// 	pkg := strings.ToLower(pkgPath[len(pkgPath)-1])

// 	endpoint, err := dev.getEndpoint(pkg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return dev.callMethodDo(endpoint, method)
// }

// CallMethod functions call an method, defined <method> struct with authentication data
func (dev Device) callMedia(method interface{}) ([]byte, error) {
	return dev.callMethodDo(dev.endpoints["media"], method)
}

func (dev Device) callDevice(method interface{}) ([]byte, error) {
	return dev.callMethodDo(dev.endpoints["device"], method)
}

func (dev Device) callMethodDo(endpoint string, method interface{}) ([]byte, error) {
	output, err := xml.MarshalIndent(method, "  ", "    ")
	if err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(output); err != nil {
		return nil, err
	}

	soap := gosoap.NewEmptySOAP()
	soap.AddBodyContent(doc.Root())
	soap.AddRootNamespaces(Xlmns)
	soap.AddAction()

	//Auth Handling
	if dev.params.Username != "" || dev.params.Password != "" {
		soap.AddWSSecurity(dev.params.Username, dev.params.Password)
	}

	return SendSoap(dev.params.HttpClient, endpoint, soap.String())
}

func SendSoap(httpClient *http.Client, endpoint, message string) ([]byte, error) {
	resp, err := httpClient.Post(endpoint, "application/soap+xml; charset=utf-8", bytes.NewBufferString(message))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SOAP request to %s failed with status code: %d", endpoint, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
