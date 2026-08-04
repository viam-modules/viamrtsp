package device

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/beevik/etree"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

func TestSendSoapNoHang(t *testing.T) {
	logger := logging.NewTestLogger(t)

	t.Run("context cancellation works", func(t *testing.T) {
		// Channel to coordinate server shutdown
		done := make(chan struct{})
		// A server that will hang forever
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For GetCapabilities request during initialization, we must return a valid response
			body, err := readBody(r)
			test.That(t, err, test.ShouldBeNil)
			if strings.Contains(r.Header.Get("Content-Type"), "soap") &&
				strings.Contains(body, "GetCapabilities") {
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
					<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
						<SOAP-ENV:Body>
							<GetCapabilitiesResponse>
								<Capabilities>
									<Media>
										<XAddr>http://example.com/onvif/media</XAddr>
									</Media>
									<PTZ>
										<XAddr>http://example.com/onvif/ptz</XAddr>
									</PTZ>
								</Capabilities>
							</GetCapabilitiesResponse>
						</SOAP-ENV:Body>
					</SOAP-ENV:Envelope>`))
				return
			}
			<-done
		}))
		defer func() {
			close(done)
			server.Close()
		}()

		serverURL, err := url.Parse(server.URL)
		test.That(t, err, test.ShouldBeNil)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Create device with the context
		dev, err := NewDevice(ctx, Params{
			Xaddr:      serverURL,
			HTTPClient: &http.Client{},
		}, logger)
		test.That(t, err, test.ShouldBeNil)

		_, err = dev.sendSoap(ctx, server.URL, "test message")
		// Cast to url.Error to check if the error is a context deadline exceeded
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			test.That(t, urlErr.Err, test.ShouldBeError, context.DeadlineExceeded)
		} else {
			t.Fatalf("expected a URL error, got: %v", err)
		}
	})
}

// Helper function to read request body.
func readBody(r *http.Request) (string, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r.Body)
	if err != nil {
		return "", err
	}
	if err := r.Body.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func TestTLSVerificationConfig(t *testing.T) {
	logger := logging.NewTestLogger(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		if strings.Contains(r.Header.Get("Content-Type"), "soap") &&
			strings.Contains(body, "GetCapabilities") {
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
					<SOAP-ENV:Body>
						<GetCapabilitiesResponse>
							<Capabilities>
								<Media>
									<XAddr>http://example.com/onvif/media</XAddr>
								</Media>
							</Capabilities>
						</GetCapabilitiesResponse>
					</SOAP-ENV:Body>
				</SOAP-ENV:Envelope>`))
		}
	}))
	defer server.Close()

	testCases := []struct {
		name                     string
		isLocal                  bool
		skipLocalTLSVerification bool
		expectSkipVerify         bool
	}{
		{
			name:                     "IP local, skip enabled",
			isLocal:                  true,
			skipLocalTLSVerification: true,
			expectSkipVerify:         true,
		},
		{
			name:                     "IP local, skip disabled",
			isLocal:                  true,
			skipLocalTLSVerification: false,
			expectSkipVerify:         false,
		},
		{
			name:                     "IP public, skip enabled",
			isLocal:                  false,
			skipLocalTLSVerification: true,
			expectSkipVerify:         false,
		},
		{
			name:                     "IP public, skip disabled",
			isLocal:                  false,
			skipLocalTLSVerification: false,
			expectSkipVerify:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testNewDevice := func(_ context.Context, params Params, logger logging.Logger) *Device {
				dev := &Device{
					xaddr:     params.Xaddr,
					logger:    logger,
					params:    params,
					endpoints: map[string]string{"device": params.Xaddr.String()},
				}

				if dev.params.HTTPClient == nil {
					// Use our tc's isLocal value instead of calling actual isLocalIPAddress function
					skipVerify := params.SkipLocalTLSVerification && tc.isLocal
					transport := &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: skipVerify,
						},
					}
					dev.params.HTTPClient = &http.Client{
						Transport: transport,
					}
				}

				// Mock the response for GetCapabilities
				dev.endpoints["media"] = "http://example.com/onvif/media"
				return dev
			}

			testURL, err := url.Parse(server.URL)
			test.That(t, err, test.ShouldBeNil)

			dev := testNewDevice(context.Background(), Params{
				Xaddr:                    testURL,
				SkipLocalTLSVerification: tc.skipLocalTLSVerification,
			}, logger)

			transport, ok := dev.params.HTTPClient.Transport.(*http.Transport)
			test.That(t, ok, test.ShouldBeTrue)

			test.That(t, transport.TLSClientConfig.InsecureSkipVerify, test.ShouldEqual, tc.expectSkipVerify)
		})
	}
}

func TestDeviceFlowWithTLSServer(t *testing.T) {
	testCases := []struct {
		name                     string
		skipLocalTLSVerification bool
		expectError              bool
	}{
		{
			name:                     "TLS local IP, skip enabled",
			skipLocalTLSVerification: true,
			expectError:              false,
		},
		{
			name:                     "TLS local IP, skip disabled",
			skipLocalTLSVerification: false,
			expectError:              true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a TLS server that uses a self-signed certificate.
			// The server replies with a valid SOAP response to a GetCapabilities request.
			tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := readBody(r)
				test.That(t, err, test.ShouldBeNil)

				if strings.Contains(r.Header.Get("Content-Type"), "soap") &&
					strings.Contains(body, "GetCapabilities") {
					w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
						<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
							<SOAP-ENV:Body>
								<GetCapabilitiesResponse>
									<Capabilities>
										<Media>
											<XAddr>http://example.com/onvif/media</XAddr>
										</Media>
									</Capabilities>
								</GetCapabilitiesResponse>
							</SOAP-ENV:Body>
						</SOAP-ENV:Envelope>`))
				}
			}))
			defer tlsServer.Close()

			u, err := url.Parse(tlsServer.URL)
			test.That(t, err, test.ShouldBeNil)

			ctx := context.Background()
			logger := logging.NewTestLogger(t)
			params := Params{
				Xaddr:                    u,
				SkipLocalTLSVerification: tc.skipLocalTLSVerification,
			}

			_, err = NewDevice(ctx, params, logger)

			if tc.expectError {
				test.That(t, err, test.ShouldNotBeNil)
				test.That(t, strings.Contains(err.Error(), "x509:"), test.ShouldBeTrue)
			} else {
				test.That(t, err, test.ShouldBeNil)
			}
		})
	}
}

func TestNewDeviceMDNSHostname(t *testing.T) {
	logger := logging.NewTestLogger(t)
	// Verify that NewDevice does not fail to parse .local mDNS hostnames.
	// Before the fix, netip.ParseAddr("mydevice.local") would error and NewDevice would return
	// "failed to parse xaddr hostname mydevice.local".
	u, err := url.Parse("http://mydevice.local:8000")
	test.That(t, err, test.ShouldBeNil)

	_, err = NewDevice(context.Background(), Params{
		Xaddr:                    u,
		SkipLocalTLSVerification: true,
	}, logger)
	// Will fail on GetCapabilities (no server), but must not fail on hostname parsing.
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, strings.Contains(err.Error(), "failed to parse xaddr hostname"), test.ShouldBeFalse)
}

const fakeCameraPassword = "secretpass"

// fakeCameraState records observations made by the fake camera server for
// later assertions.
type fakeCameraState struct {
	mu                     sync.Mutex
	dateTimeRequests       int
	dateTimeRequestHadAuth bool
	authRejections         int
}

// validWSSE validates a WS-Security UsernameToken the way a strict real camera
// does: the password digest must verify, and the Created timestamp must be
// within tolerance of the camera's own clock.
func validWSSE(body, password string, cameraNow time.Time, tolerance time.Duration) bool {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(body); err != nil {
		return false
	}
	passwordEl := doc.FindElement("//UsernameToken/Password")
	nonceEl := doc.FindElement("//UsernameToken/Nonce")
	createdEl := doc.FindElement("//UsernameToken/Created")
	if passwordEl == nil || nonceEl == nil || createdEl == nil {
		return false
	}

	created, err := time.Parse(time.RFC3339Nano, createdEl.Text())
	if err != nil {
		return false
	}
	if created.Sub(cameraNow).Abs() > tolerance {
		return false
	}

	decodedNonce, err := base64.StdEncoding.DecodeString(nonceEl.Text())
	if err != nil {
		return false
	}
	hasher := sha1.New()
	hasher.Write([]byte(string(decodedNonce) + createdEl.Text() + password))
	expected := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	return passwordEl.Text() == expected
}

// newFakeCamera returns an httptest server simulating an ONVIF camera whose
// clock differs from the local clock by clockOffset. GetSystemDateAndTime is
// served without requiring authentication (as the ONVIF spec mandates) unless
// supportsDateTime is false, in which case it returns an error. Every other
// endpoint enforces WS-Security digest auth against the camera's own clock.
func newFakeCamera(t *testing.T, clockOffset time.Duration, supportsDateTime bool) (*httptest.Server, *fakeCameraState) {
	t.Helper()
	state := &fakeCameraState{}
	const tolerance = 30 * time.Second

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		test.That(t, err, test.ShouldBeNil)
		cameraNow := time.Now().UTC().Add(clockOffset)

		if strings.Contains(body, "GetSystemDateAndTime") {
			state.mu.Lock()
			state.dateTimeRequests++
			if strings.Contains(body, "UsernameToken") {
				state.dateTimeRequestHadAuth = true
			}
			state.mu.Unlock()
			if !supportsDateTime {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
				<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
					<SOAP-ENV:Body>
						<GetSystemDateAndTimeResponse>
							<SystemDateAndTime>
								<DateTimeType>Manual</DateTimeType>
								<UTCDateTime>
									<Time><Hour>%d</Hour><Minute>%d</Minute><Second>%d</Second></Time>
									<Date><Year>%d</Year><Month>%d</Month><Day>%d</Day></Date>
								</UTCDateTime>
							</SystemDateAndTime>
						</GetSystemDateAndTimeResponse>
					</SOAP-ENV:Body>
				</SOAP-ENV:Envelope>`,
				cameraNow.Hour(), cameraNow.Minute(), cameraNow.Second(),
				cameraNow.Year(), int(cameraNow.Month()), cameraNow.Day())
			return
		}

		if !validWSSE(body, fakeCameraPassword, cameraNow, tolerance) {
			state.mu.Lock()
			state.authRejections++
			state.mu.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			//nolint: errcheck
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
					<SOAP-ENV:Body>
						<SOAP-ENV:Fault><SOAP-ENV:Code><SOAP-ENV:Subcode>ter:NotAuthorized</SOAP-ENV:Subcode></SOAP-ENV:Code></SOAP-ENV:Fault>
					</SOAP-ENV:Body>
				</SOAP-ENV:Envelope>`))
			return
		}

		switch {
		case strings.Contains(body, "GetCapabilities"):
			//nolint: errcheck
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
					<SOAP-ENV:Body>
						<GetCapabilitiesResponse>
							<Capabilities>
								<Media>
									<XAddr>http://example.com/onvif/media</XAddr>
								</Media>
							</Capabilities>
						</GetCapabilitiesResponse>
					</SOAP-ENV:Body>
				</SOAP-ENV:Envelope>`))
		case strings.Contains(body, "GetDeviceInformation"):
			//nolint: errcheck
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
					<SOAP-ENV:Body>
						<GetDeviceInformationResponse>
							<Manufacturer>FakeCam</Manufacturer>
							<Model>X1</Model>
							<FirmwareVersion>1.0</FirmwareVersion>
							<SerialNumber>ABC123</SerialNumber>
							<HardwareId>HW1</HardwareId>
						</GetDeviceInformationResponse>
					</SOAP-ENV:Body>
				</SOAP-ENV:Envelope>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, state
}

func TestClockSkewedCameraAuth(t *testing.T) {
	logger := logging.NewTestLogger(t)
	// A camera whose clock reads 2000-01-01 — decades outside any replay window.
	skew := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Sub(time.Now().UTC())

	t.Run("authenticates against camera with wrong system date", func(t *testing.T) {
		server, state := newFakeCamera(t, skew, true)
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		test.That(t, err, test.ShouldBeNil)

		dev, err := NewDevice(context.Background(), Params{
			Xaddr:      serverURL,
			Username:   "admin",
			Password:   fakeCameraPassword,
			HTTPClient: &http.Client{},
		}, logger)
		test.That(t, err, test.ShouldBeNil)
		// Offset should be ~= skew (allowing for test execution time).
		test.That(t, (dev.clockOffset - skew).Abs(), test.ShouldBeLessThan, 10*time.Second)

		info, err := dev.GetDeviceInformation(context.Background())
		test.That(t, err, test.ShouldBeNil)
		test.That(t, info.Manufacturer, test.ShouldEqual, "FakeCam")
		test.That(t, info.SerialNumber, test.ShouldEqual, "ABC123")

		state.mu.Lock()
		defer state.mu.Unlock()
		test.That(t, state.dateTimeRequests, test.ShouldBeGreaterThan, 0)
		// The clock query must be unauthenticated per the ONVIF Core Spec: on a
		// skewed camera an authenticated request would be rejected for exactly
		// the skew the call exists to measure.
		test.That(t, state.dateTimeRequestHadAuth, test.ShouldBeFalse)
		test.That(t, state.authRejections, test.ShouldEqual, 0)
	})

	t.Run("without clock sync the skewed camera rejects auth", func(t *testing.T) {
		// Negative control: a device with zero clock offset (the old behavior)
		// fails digest auth against the skewed camera, proving the fake camera
		// enforces the skew this fix addresses.
		server, state := newFakeCamera(t, skew, true)
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		test.That(t, err, test.ShouldBeNil)

		dev := &Device{
			xaddr:  serverURL,
			logger: logger,
			params: Params{
				Xaddr:      serverURL,
				Username:   "admin",
				Password:   fakeCameraPassword,
				HTTPClient: &http.Client{},
			},
			endpoints: map[string]string{"device": server.URL},
		}

		_, err = dev.GetDeviceInformation(context.Background())
		test.That(t, err, test.ShouldNotBeNil)
		test.That(t, strings.Contains(err.Error(), "status code: 400"), test.ShouldBeTrue)
		state.mu.Lock()
		defer state.mu.Unlock()
		test.That(t, state.authRejections, test.ShouldBeGreaterThan, 0)
	})

	t.Run("falls back to local clock when GetSystemDateAndTime unsupported", func(t *testing.T) {
		// Camera with a correct clock that errors on GetSystemDateAndTime: the
		// zero-offset fallback must preserve today's behavior.
		server, _ := newFakeCamera(t, 0, false)
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		test.That(t, err, test.ShouldBeNil)

		dev, err := NewDevice(context.Background(), Params{
			Xaddr:      serverURL,
			Username:   "admin",
			Password:   fakeCameraPassword,
			HTTPClient: &http.Client{},
		}, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, dev.clockOffset, test.ShouldEqual, time.Duration(0))

		info, err := dev.GetDeviceInformation(context.Background())
		test.That(t, err, test.ShouldBeNil)
		test.That(t, info.Manufacturer, test.ShouldEqual, "FakeCam")
	})
}

func TestParseSystemDateAndTimeResponse(t *testing.T) {
	t.Run("UTCDateTime present", func(t *testing.T) {
		resp := `<?xml version="1.0" encoding="UTF-8"?>
			<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope"
				xmlns:tds="http://www.onvif.org/ver10/device/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
				<SOAP-ENV:Body>
					<tds:GetSystemDateAndTimeResponse>
						<tds:SystemDateAndTime>
							<tt:DateTimeType>NTP</tt:DateTimeType>
							<tt:UTCDateTime>
								<tt:Time><tt:Hour>7</tt:Hour><tt:Minute>18</tt:Minute><tt:Second>27</tt:Second></tt:Time>
								<tt:Date><tt:Year>2000</tt:Year><tt:Month>1</tt:Month><tt:Day>1</tt:Day></tt:Date>
							</tt:UTCDateTime>
						</tds:SystemDateAndTime>
					</tds:GetSystemDateAndTimeResponse>
				</SOAP-ENV:Body>
			</SOAP-ENV:Envelope>`
		parsed, err := parseSystemDateAndTimeResponse([]byte(resp))
		test.That(t, err, test.ShouldBeNil)
		test.That(t, parsed, test.ShouldEqual, time.Date(2000, 1, 1, 7, 18, 27, 0, time.UTC))
	})

	t.Run("falls back to LocalDateTime when UTCDateTime absent", func(t *testing.T) {
		resp := `<?xml version="1.0" encoding="UTF-8"?>
			<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope"
				xmlns:tds="http://www.onvif.org/ver10/device/wsdl" xmlns:tt="http://www.onvif.org/ver10/schema">
				<SOAP-ENV:Body>
					<tds:GetSystemDateAndTimeResponse>
						<tds:SystemDateAndTime>
							<tt:LocalDateTime>
								<tt:Time><tt:Hour>12</tt:Hour><tt:Minute>0</tt:Minute><tt:Second>5</tt:Second></tt:Time>
								<tt:Date><tt:Year>2024</tt:Year><tt:Month>6</tt:Month><tt:Day>15</tt:Day></tt:Date>
							</tt:LocalDateTime>
						</tds:SystemDateAndTime>
					</tds:GetSystemDateAndTimeResponse>
				</SOAP-ENV:Body>
			</SOAP-ENV:Envelope>`
		parsed, err := parseSystemDateAndTimeResponse([]byte(resp))
		test.That(t, err, test.ShouldBeNil)
		test.That(t, parsed, test.ShouldEqual, time.Date(2024, 6, 15, 12, 0, 5, 0, time.UTC))
	})

	t.Run("errors when no date and time present", func(t *testing.T) {
		resp := `<?xml version="1.0" encoding="UTF-8"?>
			<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
				<SOAP-ENV:Body>
					<GetSystemDateAndTimeResponse><SystemDateAndTime></SystemDateAndTime></GetSystemDateAndTimeResponse>
				</SOAP-ENV:Body>
			</SOAP-ENV:Envelope>`
		_, err := parseSystemDateAndTimeResponse([]byte(resp))
		test.That(t, err, test.ShouldNotBeNil)
	})

	t.Run("errors on invalid XML", func(t *testing.T) {
		_, err := parseSystemDateAndTimeResponse([]byte("not xml"))
		test.That(t, err, test.ShouldNotBeNil)
	})
}

func TestGetProfiles(t *testing.T) {
	logger := logging.NewTestLogger(t)

	t.Run("Test GetProfiles parses XML response correctly", func(t *testing.T) {
		filePath := filepath.Join("..", "xsd", "onvif", "body_response.xml")
		bodyResponse, err := os.ReadFile(filePath)
		test.That(t, err, test.ShouldBeNil)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := readBody(r)
			test.That(t, err, test.ShouldBeNil)

			if strings.Contains(r.Header.Get("Content-Type"), "soap") &&
				strings.Contains(body, "GetProfiles") {
				w.Write(bodyResponse)
			}
		}))
		defer server.Close()

		serverURL, err := url.Parse(server.URL)
		test.That(t, err, test.ShouldBeNil)

		dev, err := NewDevice(context.Background(), Params{
			Xaddr:      serverURL,
			HTTPClient: &http.Client{},
		}, logger)
		test.That(t, err, test.ShouldBeNil)

		// Hardcode the media endpoint to point to mock server
		dev.endpoints["media"] = server.URL

		resp, err := dev.GetProfiles(context.Background())
		test.That(t, err, test.ShouldBeNil)

		test.That(t, len(resp.Profiles), test.ShouldEqual, 2)

		mainStream := resp.Profiles[0]
		test.That(t, mainStream.Token, test.ShouldEqual, "MainStream")
		test.That(t, mainStream.Name, test.ShouldEqual, "MainStream")
		test.That(t, mainStream.VideoEncoderConfiguration.Resolution.Width, test.ShouldEqual, 2560)
		test.That(t, mainStream.VideoEncoderConfiguration.Resolution.Height, test.ShouldEqual, 1440)
		test.That(t, mainStream.VideoEncoderConfiguration.RateControl.FrameRateLimit, test.ShouldEqual, 20)
		test.That(t, string(mainStream.VideoEncoderConfiguration.Encoding), test.ShouldEqual, "H264")

		subStream := resp.Profiles[1]
		test.That(t, subStream.Token, test.ShouldEqual, "SubStream")
		test.That(t, subStream.Name, test.ShouldEqual, "SubStream")
		test.That(t, subStream.VideoEncoderConfiguration.Resolution.Width, test.ShouldEqual, 640)
		test.That(t, subStream.VideoEncoderConfiguration.Resolution.Height, test.ShouldEqual, 360)
		test.That(t, subStream.VideoEncoderConfiguration.RateControl.FrameRateLimit, test.ShouldEqual, 25)
		test.That(t, string(subStream.VideoEncoderConfiguration.Encoding), test.ShouldEqual, "H264")
	})
}
