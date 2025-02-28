package device

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
					// Use our tc's isLocal value instead of calling actual func
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
