package viamonvif

import (
	"errors"
	"net/url"
	"testing"

	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

type MockDevice struct{}

func NewMockDevice() *MockDevice {
	return &MockDevice{}
}

func (m *MockDevice) GetDeviceInformation() (device.GetDeviceInformationResponse, error) {
	return device.GetDeviceInformationResponse{
		Manufacturer: "Evil Inc.",
		Model:        "Doom Ray Camera of Certain Annihilation",
		SerialNumber: "44444444",
	}, nil
}

func (m *MockDevice) GetProfiles() (device.GetProfilesResponse, error) {
	return device.GetProfilesResponse{
		Profiles: []onvif.Profile{
			{
				Token: "profile1",
				Name:  "Main Profile",
			},
		},
	}, nil
}

func (m *MockDevice) GetStreamURI(profile onvif.Profile, creds device.Credentials) (*url.URL, error) {
	if profile.Token != "profile1" || profile.Name != "Main Profile" {
		return nil, errors.New("invalid mock profile")
	}
	u, err := url.Parse("rtsp://192.168.1.100/stream")
	if err != nil {
		return nil, err
	}
	if creds.User != "" || creds.Pass != "" {
		u.User = url.UserPassword(creds.User, creds.Pass)
	}
	return u, nil
}

func TestGetCameraInfo(t *testing.T) {
	t.Run("GetCameraInfo", func(t *testing.T) {
		mockDevice := &MockDevice{}
		logger := logging.NewTestLogger(t)

		uri, err := url.Parse("192.168.1.100")
		test.That(t, err, test.ShouldBeNil)
		cameraInfo, err := GetCameraInfo(mockDevice, uri, device.Credentials{User: "username", Pass: "password"}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, cameraInfo.Manufacturer, test.ShouldEqual, "Evil Inc.")
		test.That(t, cameraInfo.Model, test.ShouldEqual, "Doom Ray Camera of Certain Annihilation")
		test.That(t, cameraInfo.SerialNumber, test.ShouldEqual, "44444444")
		test.That(t, len(cameraInfo.RTSPURLs), test.ShouldEqual, 1)
		test.That(t, cameraInfo.RTSPURLs[0], test.ShouldEqual, "rtsp://username:password@192.168.1.100/stream")
		test.That(t, cameraInfo.Name(), test.ShouldEqual, "Evil Inc-Doom Ray Camera of Certain Annihilation-44444444")

		t.Run("GetRTSPStreamURLs with credentials", func(t *testing.T) {
			uri, err := url.Parse("192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(mockDevice, uri, device.Credentials{User: "username", Pass: "password"}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.RTSPURLs), test.ShouldEqual, 1)
			test.That(t, cameraInfo.RTSPURLs[0], test.ShouldEqual, "rtsp://username:password@192.168.1.100/stream")
		})
		t.Run("GetRTSPStreamURLs without credentials", func(t *testing.T) {
			uri, err := url.Parse("192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(mockDevice, uri, device.Credentials{}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.RTSPURLs), test.ShouldEqual, 1)
			test.That(t, cameraInfo.RTSPURLs[0], test.ShouldEqual, "rtsp://192.168.1.100/stream")
		})
	})
}

func TestCameraName(t *testing.T) {
	cam := CameraInfo{Manufacturer: "M#a$n&u™facturer1", Model: "Good Mod*el", SerialNumber: "123()456"}
	test.That(t, cam.Name(), test.ShouldEqual, "Manufacturer1-Good Model-123456")
}

func TestExtractXAddrsFromProbeMatch(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		response := `
			<Envelope>
				<Body>
					<ProbeMatches>
						<ProbeMatch>
							<XAddrs>http://192.168.1.100 http://192.168.1.101</XAddrs>
						</ProbeMatch>
					</ProbeMatches>
				</Body>
			</Envelope>`

		expected := []*url.URL{
			{Scheme: "http", Host: "192.168.1.100"},
			{Scheme: "http", Host: "192.168.1.101"},
		}
		xaddrs := extractXAddrsFromProbeMatch(response, logging.NewTestLogger(t))
		t.Logf("xaddrs: %v", xaddrs)
		test.That(t, xaddrs, test.ShouldResemble, expected)
	})

	t.Run("Garbage data", func(t *testing.T) {
		response := `garbage data: ;//\\<>httphttp://ddddddd</</>/>`
		xaddrs := extractXAddrsFromProbeMatch(response, logging.NewTestLogger(t))
		test.That(t, xaddrs, test.ShouldBeEmpty)
	})

	t.Run("Empty Response", func(t *testing.T) {
		response := `
			<Envelope>
				<Body>
					<ProbeMatches>
					</ProbeMatches>
				</Body>
			</Envelope>`

		xaddrs := extractXAddrsFromProbeMatch(response, logging.NewTestLogger(t))
		test.That(t, xaddrs, test.ShouldBeEmpty)
	})
}
