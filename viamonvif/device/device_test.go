package device

import (
	"bytes"
	"context"
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
