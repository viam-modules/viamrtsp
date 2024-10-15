package viamonvif

import (
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

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
