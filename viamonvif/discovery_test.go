package viamonvif

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/viam-modules/viamrtsp/ptzclient"
	"github.com/viam-modules/viamrtsp/viamonvif/device"
	"github.com/viam-modules/viamrtsp/viamonvif/xsd/onvif"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
	"go.viam.com/utils"
)

type MockDevice struct {
	GetProfilesFn    func(context.Context) (device.GetProfilesResponse, error)
	GetPTZNodesFn    func(context.Context) ([]onvif.PTZNode, error)
	GetStreamURIFn   func(context.Context, onvif.ReferenceToken, device.Credentials) (*url.URL, error)
	GetSnapshotURIFn func(context.Context, onvif.ReferenceToken, device.Credentials) (*url.URL, error)
	GetXaddrFn       func() *url.URL
}

func NewMockDevice() *MockDevice {
	return &MockDevice{
		GetProfilesFn: func(_ context.Context) (device.GetProfilesResponse, error) {
			return device.GetProfilesResponse{Profiles: []onvif.Profile{
				{Token: "profile1", Name: "Main Profile"},
			}}, nil
		},
		GetPTZNodesFn: func(_ context.Context) ([]onvif.PTZNode, error) {
			return []onvif.PTZNode{}, nil
		},
		GetStreamURIFn: func(_ context.Context, _ onvif.ReferenceToken, _ device.Credentials) (*url.URL, error) {
			return url.Parse("rtsp://192.168.1.100/stream")
		},
		GetSnapshotURIFn: func(_ context.Context, _ onvif.ReferenceToken, _ device.Credentials) (*url.URL, error) {
			return url.Parse("http://example.com/snapshot.jpg")
		},
		GetXaddrFn: func() *url.URL {
			u, _ := url.Parse("http://192.168.1.100:80")
			return u
		},
	}
}

func (m *MockDevice) GetDeviceInformation(_ context.Context) (device.GetDeviceInformationResponse, error) {
	return device.GetDeviceInformationResponse{
		Manufacturer: "Evil Inc.",
		Model:        "Doom Ray Camera of Certain Annihilation",
		SerialNumber: "44444444",
	}, nil
}

func (m *MockDevice) GetProfiles(ctx context.Context) (device.GetProfilesResponse, error) {
	return m.GetProfilesFn(ctx)
}

func (m *MockDevice) GetSnapshotURI(ctx context.Context, token onvif.ReferenceToken, creds device.Credentials) (*url.URL, error) {
	return m.GetSnapshotURIFn(ctx, token, creds)
}

func (m *MockDevice) GetStreamURI(ctx context.Context, token onvif.ReferenceToken, creds device.Credentials) (*url.URL, error) {
	return m.GetStreamURIFn(ctx, token, creds)
}

func (m *MockDevice) GetPTZNodes(ctx context.Context) ([]onvif.PTZNode, error) {
	return m.GetPTZNodesFn(ctx)
}

func (m *MockDevice) GetXaddr() *url.URL {
	return m.GetXaddrFn()
}

func TestGetCameraInfo(t *testing.T) {
	t.Run("GetCameraInfo", func(t *testing.T) {
		// mockDevice := &MockDevice{}
		mockDevice := NewMockDevice()
		logger := logging.NewTestLogger(t)

		mockDevice.GetStreamURIFn = func(_ context.Context, token onvif.ReferenceToken, creds device.Credentials) (*url.URL, error) {
			if token != "profile1" {
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

		uri, err := url.Parse("192.168.1.100")
		test.That(t, err, test.ShouldBeNil)
		cameraInfo, err := GetCameraInfo(context.Background(), mockDevice, uri, device.Credentials{User: "username", Pass: "password"}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, cameraInfo.Manufacturer, test.ShouldEqual, "Evil Inc.")
		test.That(t, cameraInfo.Model, test.ShouldEqual, "Doom Ray Camera of Certain Annihilation")
		test.That(t, cameraInfo.SerialNumber, test.ShouldEqual, "44444444")
		test.That(t, len(cameraInfo.MediaEndpoints), test.ShouldEqual, 1)
		test.That(t, cameraInfo.MediaEndpoints[0].StreamURI, test.ShouldEqual, "rtsp://username:password@192.168.1.100/stream")
		test.That(t, cameraInfo.Name(0), test.ShouldEqual, "EvilInc-DoomRayCameraofCertainAnnihilation-44444444-url0")

		t.Run("GetCameraInfo with credentials", func(t *testing.T) {
			uri, err := url.Parse("192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(context.Background(), mockDevice, uri, device.Credentials{User: "username", Pass: "password"}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.MediaEndpoints), test.ShouldEqual, 1)
			test.That(t, cameraInfo.MediaEndpoints[0].StreamURI, test.ShouldEqual, "rtsp://username:password@192.168.1.100/stream")
			test.That(t, cameraInfo.MediaEndpoints[0].SnapshotURI, test.ShouldEqual, "http://example.com/snapshot.jpg")
		})
		t.Run("GetCameraInfo without credentials", func(t *testing.T) {
			uri, err := url.Parse("192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(context.Background(), mockDevice, uri, device.Credentials{}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.MediaEndpoints), test.ShouldEqual, 1)
			test.That(t, cameraInfo.MediaEndpoints[0].StreamURI, test.ShouldEqual, "rtsp://192.168.1.100/stream")
			test.That(t, cameraInfo.MediaEndpoints[0].SnapshotURI, test.ShouldEqual, "http://example.com/snapshot.jpg")
		})

		t.Run("GetCameraInfo with no PTZ nodes", func(t *testing.T) {
			uri, err := url.Parse("192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(context.Background(), mockDevice, uri, device.Credentials{}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.PTZEndpoints), test.ShouldEqual, 0)
		})

		t.Run("GetCameraInfo with one PTZ node", func(t *testing.T) {
			mockDevice.GetProfilesFn = func(_ context.Context) (device.GetProfilesResponse, error) {
				return device.GetProfilesResponse{
					Profiles: []onvif.Profile{
						{
							Token: "profile1",
							Name:  "Main Profile",
							PTZConfiguration: onvif.PTZConfiguration{
								NodeToken: onvif.ReferenceToken("PTZNode1"),
							},
						},
					},
				}, nil
			}
			mockDevice.GetPTZNodesFn = func(_ context.Context) ([]onvif.PTZNode, error) {
				node := onvif.PTZNode{
					DeviceEntity: onvif.DeviceEntity{
						Token: "PTZNode1",
					},
					Name: "PTZNode1",
					SupportedPTZSpaces: onvif.PTZSpaces{
						ContinuousPanTiltVelocitySpace: onvif.Space2DDescription{
							URI:    "http://www.onvif.org/ver10/tptz/PanTiltSpaces/VelocityGenericSpace",
							XRange: onvif.FloatRange{Min: -1.0, Max: 1.0},
							YRange: onvif.FloatRange{Min: -1.0, Max: 1.0},
						},
						ContinuousZoomVelocitySpace: onvif.Space1DDescription{
							URI:    "http://www.onvif.org/ver10/tptz/ZoomSpaces/VelocityGenericSpace",
							XRange: onvif.FloatRange{Min: 0.0, Max: 1.0},
						},
					},
				}
				return []onvif.PTZNode{node}, nil
			}
			uri, err := url.Parse("192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(context.Background(), mockDevice, uri, device.Credentials{}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.MediaEndpoints), test.ShouldEqual, 1)
			test.That(t, len(cameraInfo.PTZEndpoints), test.ShouldEqual, 1)
			expectedMovements := map[string]ptzclient.PTZMovement{
				"continuous": {
					PanTilt: ptzclient.PanTiltSpace{
						XMin:  -1.0,
						XMax:  1.0,
						YMin:  -1.0,
						YMax:  1.0,
						Space: "VelocityGenericSpace",
					},
					Zoom: ptzclient.ZoomSpace{
						XMin:  0.0,
						XMax:  1.0,
						Space: "VelocityGenericSpace",
					},
				},
			}
			test.That(t,
				cameraInfo.PTZEndpoints[0].Movements,
				test.ShouldResemble,
				expectedMovements,
			)
			test.That(t, cameraInfo.PTZEndpoints[0].PTZNodeToken, test.ShouldEqual, "PTZNode1")
			test.That(t, cameraInfo.PTZEndpoints[0].ProfileToken, test.ShouldEqual, "profile1")
			test.That(t, cameraInfo.PTZEndpoints[0].RTSPAddress, test.ShouldEqual, "rtsp://192.168.1.100/stream")
			test.That(t, cameraInfo.PTZEndpoints[0].Address, test.ShouldEqual, "192.168.1.100:80")
		})

		t.Run("GetCameraInfo with multiple PTZ nodes", func(t *testing.T) {
			mockDevice.GetProfilesFn = func(_ context.Context) (device.GetProfilesResponse, error) {
				return device.GetProfilesResponse{
					Profiles: []onvif.Profile{
						{
							Token: "profile1",
							Name:  "Main Profile",
							PTZConfiguration: onvif.PTZConfiguration{
								NodeToken: onvif.ReferenceToken("PTZNode1"),
							},
						},
						{
							Token: "profile2",
							Name:  "Secondary Profile",
							PTZConfiguration: onvif.PTZConfiguration{
								NodeToken: onvif.ReferenceToken("PTZNode2"),
							},
						},
					},
				}, nil
			}
			mockDevice.GetStreamURIFn = func(_ context.Context, token onvif.ReferenceToken, _ device.Credentials) (*url.URL, error) {
				if token == "profile1" {
					return url.Parse("rtsp://192.168.1.100/stream1")
				}
				if token == "profile2" {
					return url.Parse("rtsp://192.168.1.100/stream2")
				}
				return nil, errors.New("invalid mock profile")
			}
			mockDevice.GetPTZNodesFn = func(_ context.Context) ([]onvif.PTZNode, error) {
				nodes := []onvif.PTZNode{
					{
						DeviceEntity: onvif.DeviceEntity{
							Token: "PTZNode1",
						},
						Name: "PTZNode1",
						SupportedPTZSpaces: onvif.PTZSpaces{
							ContinuousPanTiltVelocitySpace: onvif.Space2DDescription{
								URI:    "http://www.onvif.org/ver10/tptz/PanTiltSpaces/VelocityGenericSpace",
								XRange: onvif.FloatRange{Min: -1.0, Max: 1.0},
								YRange: onvif.FloatRange{Min: -1.0, Max: 1.0},
							},
							ContinuousZoomVelocitySpace: onvif.Space1DDescription{
								URI:    "http://www.onvif.org/ver10/tptz/ZoomSpaces/VelocityGenericSpace",
								XRange: onvif.FloatRange{Min: 0.0, Max: 1.0},
							},
						},
					},
					{
						DeviceEntity: onvif.DeviceEntity{
							Token: "PTZNode2",
						},
						Name: "PTZNode2",
						SupportedPTZSpaces: onvif.PTZSpaces{
							ContinuousPanTiltVelocitySpace: onvif.Space2DDescription{
								URI:    "http://www.onvif.org/ver10/tptz/PanTiltSpaces/VelocityGenericSpace",
								XRange: onvif.FloatRange{Min: -0.5, Max: 0.5},
								YRange: onvif.FloatRange{Min: -0.5, Max: 0.5},
							},
							ContinuousZoomVelocitySpace: onvif.Space1DDescription{
								URI:    "http://www.onvif.org/ver10/tptz/ZoomSpaces/VelocityGenericSpace",
								XRange: onvif.FloatRange{Min: 0.0, Max: 2.0},
							},
						},
					},
				}
				return nodes, nil
			}
			uri, err := url.Parse("http://192.168.1.100")
			test.That(t, err, test.ShouldBeNil)
			cameraInfo, err := GetCameraInfo(context.Background(), mockDevice, uri, device.Credentials{}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, len(cameraInfo.MediaEndpoints), test.ShouldEqual, 2)
			test.That(t, len(cameraInfo.PTZEndpoints), test.ShouldEqual, 2)
			expectedMovements1 := map[string]ptzclient.PTZMovement{
				"continuous": {
					PanTilt: ptzclient.PanTiltSpace{
						XMin:  -1.0,
						XMax:  1.0,
						YMin:  -1.0,
						YMax:  1.0,
						Space: "VelocityGenericSpace",
					},
					Zoom: ptzclient.ZoomSpace{
						XMin:  0.0,
						XMax:  1.0,
						Space: "VelocityGenericSpace",
					},
				},
			}
			expectedMovements2 := map[string]ptzclient.PTZMovement{
				"continuous": {
					PanTilt: ptzclient.PanTiltSpace{
						XMin:  -0.5,
						XMax:  0.5,
						YMin:  -0.5,
						YMax:  0.5,
						Space: "VelocityGenericSpace",
					},
					Zoom: ptzclient.ZoomSpace{
						XMin:  0.0,
						XMax:  2.0,
						Space: "VelocityGenericSpace",
					},
				},
			}
			test.That(t,
				cameraInfo.PTZEndpoints[0].Movements,
				test.ShouldResemble,
				expectedMovements1,
			)
			test.That(t,
				cameraInfo.PTZEndpoints[1].Movements,
				test.ShouldResemble,
				expectedMovements2,
			)
			test.That(t, cameraInfo.PTZEndpoints[0].PTZNodeToken, test.ShouldEqual, "PTZNode1")
			test.That(t, cameraInfo.PTZEndpoints[0].ProfileToken, test.ShouldEqual, "profile1")
			test.That(t, cameraInfo.PTZEndpoints[0].RTSPAddress, test.ShouldEqual, "rtsp://192.168.1.100/stream1")
			test.That(t, cameraInfo.PTZEndpoints[0].Address, test.ShouldEqual, "192.168.1.100:80")
			test.That(t, cameraInfo.PTZEndpoints[1].PTZNodeToken, test.ShouldEqual, "PTZNode2")
			test.That(t, cameraInfo.PTZEndpoints[1].ProfileToken, test.ShouldEqual, "profile2")
			test.That(t, cameraInfo.PTZEndpoints[1].RTSPAddress, test.ShouldEqual, "rtsp://192.168.1.100/stream2")
			test.That(t, cameraInfo.PTZEndpoints[1].Address, test.ShouldEqual, "192.168.1.100:80")
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

func TestParseIPFromHost(t *testing.T) {
	// net.ParseIP fails on host:port strings
	test.That(t, net.ParseIP("192.168.10.89:80"), test.ShouldBeNil)
	// parseIPFromHost strips the port and parses correctly
	test.That(t, parseIPFromHost("192.168.10.89:80").String(), test.ShouldEqual, "192.168.10.89")
	// Hostnames return nil
	test.That(t, parseIPFromHost("example.com:80"), test.ShouldBeNil)
}

func TestMDNSMapping(t *testing.T) {
	// This test relies on having access to multicast network interfaces. Cloud instances often
	// disable multicast capabilities. We skip this test by default as we can't expect CI testing to
	// succeed.
	t.Skip("fails in CI")
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

	cmd = exec.Command("ping", "-c1", nonSenseWithLocal)
	output, _ = cmd.CombinedOutput()
	test.That(t, string(output), test.ShouldContainSubstring, "127.0.0.1")

	// Dan: Ideally we'd additionally test that removing + re-pinging results in an error
	// again. However, I've observed a small sleep is necessary to ensure that the mdns entry is no
	// longer being served. I'd rather not play the game of having to use increasingly larger sleeps
	// to ensure the test remains reliable.
}

func TestFileCaching(t *testing.T) {
	logger := logging.NewTestLogger(t)
	file, err := os.CreateTemp("", "*")
	test.That(t, err, test.ShouldBeNil)
	cachedMDNSMappingsFilename := file.Name()
	file.Close()

	mdnsServer := newMDNSServerFromCachedData(cachedMDNSMappingsFilename, logger)
	defer mdnsServer.Shutdown()

	// The above create/close should create an empty file. Given the empty string is not valid json,
	// we expect nil to be returned.
	cachedDNSResults := mdnsServer.readCacheFile()
	test.That(t, cachedDNSResults, test.ShouldBeNil)
	test.That(t, len(cachedDNSResults), test.ShouldEqual, 0)

	// Delete the file
	test.That(t, os.Remove(cachedMDNSMappingsFilename), test.ShouldBeNil)

	// Similarly, a file that does not exist also returns nil.
	cachedDNSResults = mdnsServer.readCacheFile()
	test.That(t, cachedDNSResults, test.ShouldBeNil)

	// Choose a random string for our two mappings `nonSenseOne` and `nonSenseTwo`. Start
	// `nonSenseOne` with an `a` and `nonSenseTwo` with a `z`. Such that `nonSenseOne` sorts before
	// `nonSenseTwo`. To ease assertion testing.
	nonSenseOne := "a" + utils.RandomAlphaString(10)
	logger.Info("First conjured DNS name:", nonSenseOne)

	nonSenseTwo := "z" + utils.RandomAlphaString(10)
	logger.Info("Second conjured DNS name:", nonSenseTwo)

	// Add the two mappings to the mdns server.
	mdnsServer.Add(nonSenseOne, net.ParseIP("127.0.0.1"))
	mdnsServer.Add(nonSenseTwo, net.ParseIP("127.0.0.2"))
	test.That(t, len(mdnsServer.mappedDevices), test.ShouldEqual, 2)

	// Write out a cache file that ought to contain both mappings.
	mdnsServer.UpdateCacheFile()

	// Read it in by hand and verify the results make match.
	cachedDNSResults = mdnsServer.readCacheFile()
	test.That(t, len(cachedDNSResults), test.ShouldEqual, 2)

	slices.SortFunc(cachedDNSResults, func(left, right cachedEntry) int {
		if left.DNSName < right.DNSName {
			return -1
		}

		// Can't be equal -- omitting equality check for brevity.
		return 1
	})

	// Leverage the sorting to safely assume the first entry represents `nonSenseOne`.
	test.That(t, cachedDNSResults[0].DNSName, test.ShouldEqual, nonSenseOne)
	test.That(t, cachedDNSResults[0].IP, test.ShouldEqual, "127.0.0.1")
	test.That(t, cachedDNSResults[1].DNSName, test.ShouldEqual, nonSenseTwo)
	test.That(t, cachedDNSResults[1].IP, test.ShouldEqual, "127.0.0.2")

	// Start a new mdns server against the same file. This will load/apply the cache file. This test
	// is not skipped, so we do not assert `ping`ing works. Just the existence of the expected
	// entries in the `mappedDevices` map.
	cleanMDNSServer := newMDNSServerFromCachedData(cachedMDNSMappingsFilename, logger)
	test.That(t, len(cleanMDNSServer.mappedDevices), test.ShouldEqual, 2)
	test.That(t, cleanMDNSServer.mappedDevices[nonSenseOne].ip.String(), test.ShouldEqual, "127.0.0.1")
	test.That(t, cleanMDNSServer.mappedDevices[nonSenseTwo].ip.String(), test.ShouldEqual, "127.0.0.2")
}
