package viamonvif

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"testing"
	"time"

	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
	"go.viam.com/utils"
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
		test.That(t, cameraInfo.Name(0), test.ShouldEqual, "EvilInc-DoomRayCameraofCertainAnnihilation-44444444-url0")

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
	cam := CameraInfo{Manufacturer: "M#a$n&u™facturer1", Model: "G o od M o d*e l", SerialNumber: "123()456"}
	test.That(t, cam.Name(7), test.ShouldEqual, "Manufacturer1-GoodModel-123456-url7")
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

func TestStrToHostName(t *testing.T) {
	test.That(t, strToHostName("ABC-DEF-123"), test.ShouldEqual, "ABC-DEF-123")
	test.That(t, strToHostName("ABC*DEF/123"), test.ShouldEqual, "ABC-DEF-123")
	// Dan: Not claiming the following test cases are a good idea. Just asserting the multiple dash
	// compression is working as expected.
	test.That(t, strToHostName("____"), test.ShouldEqual, "-")
	test.That(t, strToHostName("A______3"), test.ShouldEqual, "A-3")
	test.That(t, strToHostName("--__--A--__--3--__--"), test.ShouldEqual, "-A-3-")
	test.That(t, strToHostName("$$__$$A$$__$$3$$__$$"), test.ShouldEqual, "-A-3-")
}

func TestMDNSMapping(t *testing.T) {
	logger := logging.NewTestLogger(t)
	mdnsServer := newMDNSServer(logger)
	defer mdnsServer.Shutdown()

	// This test will:
	// 1) Try pinging a non-sense.local DNS name.
	//  - Assert it gets an error
	// 2) Add non-sense.local to the mdnsServer mapped to 127.0.0.1
	// 3) Retry the ping command
	//  - Assert ping succeeds.
	//
	// This is a risky test to write. It makes assumptions about OS/distributions that the test is
	// run on. We'll try to assert on some of the most conservative/safest properties of the ping
	// output: if the IP address `127.0.0.1` appears, the ping succeeded.
	nonSense := utils.RandomAlphaString(10)
	nonSenseWithLocal := fmt.Sprintf("%v.local", nonSense)
	logger.Info("Conjured DNS Name:", nonSenseWithLocal)

	// `-c1` terminates the ping command after one response. Rather than going on forever.
	cmd := exec.Command("ping", "-c1", nonSenseWithLocal)
	output, _ := cmd.CombinedOutput()
	test.That(t, string(output), test.ShouldNotContainSubstring, "127.0.0.1")

	err := mdnsServer.Add(nonSense, net.ParseIP("127.0.0.1"))
	test.That(t, err, test.ShouldBeNil)

	time.Sleep(10 * time.Second)

	cmd = exec.Command("ping", "-c1", nonSenseWithLocal)
	output, _ = cmd.CombinedOutput()
	test.That(t, string(output), test.ShouldContainSubstring, "127.0.0.1")

	// Dan: Ideally we'd additionally test that removing + re-pinging results in an error
	// again. However, I've observed a small sleep is necessary to ensure that the mdns entry is no
	// longer being served. I'd rather not play the game of having to use increasingly larger sleeps
	// to ensure the test remains reliable.
}
