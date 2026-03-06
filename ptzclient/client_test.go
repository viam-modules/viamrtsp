package ptzclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/test"
)

const mockCapabilitiesResponse = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
  <SOAP-ENV:Body><GetCapabilitiesResponse><Capabilities>
    <Media><XAddr>http://127.0.0.1:1/onvif/media</XAddr></Media>
  </Capabilities></GetCapabilitiesResponse></SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

// TestNewClientServicePathDefaulting verifies that NewClient defaults the ONVIF
// device service path to /onvif/device_service when the address omits a path,
// providing backwards compatibility for old configs that stored only host:port.
func TestNewClientServicePathDefaulting(t *testing.T) {
	logger := logging.NewTestLogger(t)

	// Only serve capabilities at the standard ONVIF device service path.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/onvif/device_service" {
			_, _ = w.Write([]byte(mockCapabilitiesResponse))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "http://")
	name := resource.NewName(generic.API, "test-ptz")

	testCases := []struct {
		name    string
		address string
	}{
		{"bare host:port without scheme", hostPort},
		{"http://host:port without path", server.URL},
		{"http://host:port/ with root path", server.URL + "/"},
		{"explicit /onvif/device_service path preserved", server.URL + "/onvif/device_service"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Address: tc.address, ProfileToken: "test-profile"}
			client, err := NewClient(context.Background(), nil, name, cfg, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, client, test.ShouldNotBeNil)
		})
	}
}
