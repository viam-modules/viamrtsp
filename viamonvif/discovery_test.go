package viamonvif

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

type MockDevice struct{}

func NewMockDevice() *MockDevice {
	return &MockDevice{}
}

func (m *MockDevice) CallMethod(request interface{}) (*http.Response, error) {
	switch request.(type) {
	case device.GetDeviceInformation:
		body := `<Envelope>
				<Body>
					<GetDeviceInformationResponse>
						<Manufacturer>Evil Inc.</Manufacturer>
						<Model>Doom Ray Camera of Certain Annihilation</Model>
						<SerialNumber>44444444</SerialNumber>
					</GetDeviceInformationResponse>
				</Body>
			</Envelope>`
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(body))),
		}, nil

	case media.GetProfiles:
		body := `<Envelope>
				<Body>
					<GetProfilesResponse>
						<Profiles>
							<Profile>
								<Token>profile1</Token>
								<Name>Main Profile</Name>
							</Profile>
						</Profiles>
					</GetProfilesResponse>
				</Body>
			</Envelope>`
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(body))),
		}, nil

	case media.GetStreamUri:
		body := `<Envelope>
				<Body>
					<GetStreamUriResponse>
						<MediaUri>
							<Uri>rtsp://192.168.1.100/stream</Uri>
						</MediaUri>
					</GetStreamUriResponse>
				</Body>
			</Envelope>`
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(body))),
		}, nil

	default:
		return nil, errors.New("unsupported request")
	}
}

func TestGetCameraInfo(t *testing.T) {
	t.Run("GetCameraInfo", func(t *testing.T) {
		mockDevice := &MockDevice{}
		logger := logging.NewTestLogger(t)

		cameraInfo, err := getCameraInfo(mockDevice, "username", "password", logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, cameraInfo.Manufacturer, test.ShouldEqual, "Evil Inc.")
		test.That(t, cameraInfo.Model, test.ShouldEqual, "Doom Ray Camera of Certain Annihilation")
		test.That(t, cameraInfo.SerialNumber, test.ShouldEqual, "44444444")
		test.That(t, len(cameraInfo.RTSPURLs), test.ShouldEqual, 1)
		test.That(t, cameraInfo.RTSPURLs[0], test.ShouldEqual, "rtsp://username:password@192.168.1.100/stream")
		test.That(t, cameraInfo.NoLoginURLs[0], test.ShouldEqual, "192.168.1.100/stream?")
	})
}

func TestGetRTSPStreamURLs(t *testing.T) {
	mockDevice := &MockDevice{}
	logger := logging.NewTestLogger(t)
	t.Run("GetRTSPStreamURLs with credentials", func(t *testing.T) {
		rtspURLs, noLoginURLs, err := getRTSPStreamURLs(mockDevice, "username", "password", logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(rtspURLs), test.ShouldEqual, 1)
		test.That(t, rtspURLs[0], test.ShouldEqual, "rtsp://username:password@192.168.1.100/stream")
		test.That(t, noLoginURLs[0], test.ShouldEqual, "192.168.1.100/stream?")
	})
	t.Run("GetRTSPStreamURLs without credentials", func(t *testing.T) {
		rtspURLs, noLoginURLs, err := getRTSPStreamURLs(mockDevice, "", "", logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, len(rtspURLs), test.ShouldEqual, 1)
		test.That(t, rtspURLs[0], test.ShouldEqual, "rtsp://192.168.1.100/stream")
		test.That(t, noLoginURLs[0], test.ShouldEqual, "192.168.1.100/stream?")
	})
}

func TestExtractXAddrsFromProbeMatch(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		response := []byte(`
			<Envelope>
				<Body>
					<ProbeMatches>
						<ProbeMatch>
							<XAddrs>http://192.168.1.100 http://192.168.1.101</XAddrs>
						</ProbeMatch>
					</ProbeMatches>
				</Body>
			</Envelope>`)

		expected := []string{"192.168.1.100", "192.168.1.101"}
		xaddrs := extractXAddrsFromProbeMatch(response, logging.NewTestLogger(t))
		t.Logf("xaddrs: %v", xaddrs)
		test.That(t, xaddrs, test.ShouldResemble, expected)
	})

	t.Run("Garbage data", func(t *testing.T) {
		response := []byte(`garbage data: ;//\\<>httphttp://ddddddd</</>/>`)
		xaddrs := extractXAddrsFromProbeMatch(response, logging.NewTestLogger(t))
		test.That(t, xaddrs, test.ShouldBeNil)
	})

	t.Run("Empty Response", func(t *testing.T) {
		response := []byte(`
			<Envelope>
				<Body>
					<ProbeMatches>
					</ProbeMatches>
				</Body>
			</Envelope>`)

		xaddrs := extractXAddrsFromProbeMatch(response, logging.NewTestLogger(t))
		test.That(t, xaddrs, test.ShouldBeEmpty)
	})
}
