package viamonvif

import (
	"context"
	"encoding/xml"
	"fmt"
	"maps"
	"net"
	"net/url"
	"slices"
	"strings"

	"go.viam.com/rdk/logging"
)

// rfc2544BenchmarkNet is 198.18.0.0/15, reserved by RFC 2544 for benchmark
// testing. It is not globally routable, and some vendors (notably Raymarine
// marine networks) use it as the LAN range for their devices, so we treat it
// as local for ONVIF discovery purposes.
var rfc2544BenchmarkNet = &net.IPNet{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)}

// isLocalIP checks if the given IP address is a local/private network address.
// This includes RFC 1918 private ranges (10.x, 172.16-31.x, 192.168.x, fc00::/7),
// loopback (127.x, ::1), and the RFC 2544 benchmark range (198.18.0.0/15) which
// is non-globally-routable and used as a LAN range by some vendors.
func isLocalIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || rfc2544BenchmarkNet.Contains(ip)
}

// WSDiscovery runs WS-Discovery on the network interface.
func WSDiscovery(ctx context.Context, logger logging.Logger, iface net.Interface) ([]*url.URL, error) {
	logger.Debugf("WS-Discovery starting on interface: %s\n", iface.Name)
	defer logger.Debugf("WS-Discovery stopping on interface: %s\n", iface.Name)
	if !validWSDiscoveryInterface(iface) {
		return nil, fmt.Errorf("WS-Discovery skipping interface %s: does not meet WS-Discovery requirements", iface.Name)
	}
	var discoveryResps []string
	// run ws-discovery probe 3 times due to sync flakiness between announcer and requester
	attempts := 3
	//nolint:intrange
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context canceled for interface: %s", iface.Name)
		}

		resp, err := SendProbe(iface.Name, logger)
		if err != nil {
			logger.Debugf("breaking at attempt %d: failed to send WS-Discovery probe on interface %s: %w\n", i+1, iface.Name, err)
			break
		}
		logger.Debugf("attempt %d succeeded: WS-Discovery probe on interface %s", i+1, iface.Name)
		for i, r := range resp {
			if i == 0 {
				logger.Debugf("WS-Discovery responses count: %d\n", len(resp))
			}
			logger.Debugf("idx: %d\n%s\n", i, r)
		}

		discoveryResps = append(discoveryResps, resp...)
	}

	if len(discoveryResps) == 0 {
		return nil, fmt.Errorf("no unique discovery responses received on interface %s after multiple attempts", iface.Name)
	}

	xaddrsSet := make(map[string]*url.URL)
	for _, response := range discoveryResps {
		for _, xaddr := range extractXAddrsFromProbeMatch(response, logger) {
			xaddrsSet[xaddr.Host] = xaddr
		}
	}

	return slices.Collect(maps.Values(xaddrsSet)), nil
}

// extractXAddrsFromProbeMatch extracts XAddrs from the WS-Discovery ProbeMatch response.
// It filters out any XAddrs that point to non-local IP addresses to prevent
// reaching out to external/public IPs that may be maliciously advertised by cameras.
func extractXAddrsFromProbeMatch(response string, logger logging.Logger) []*url.URL {
	type ProbeMatch struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			ProbeMatches struct {
				ProbeMatch []struct {
					XAddrs string `xml:"XAddrs"`
				} `xml:"ProbeMatch"`
			} `xml:"ProbeMatches"`
		} `xml:"Body"`
	}

	var probeMatch ProbeMatch
	err := xml.NewDecoder(strings.NewReader(response)).Decode(&probeMatch)
	if err != nil {
		logger.Warnf("error unmarshalling ONVIF discovery xml response: %w\nFull xml resp: %s", err, response)
	}

	xaddrs := []*url.URL{}
	for _, match := range probeMatch.Body.ProbeMatches.ProbeMatch {
		for _, xaddr := range strings.Split(match.XAddrs, " ") {
			parsedURL, err := url.Parse(xaddr)
			if err != nil {
				logger.Warnf("failed to parse XAddr %s: %w", xaddr, err)
				continue
			}

			// Only allow private/loopback IPs
			hostname := parsedURL.Hostname()
			ip := net.ParseIP(hostname)
			if ip == nil {
				logger.Warnf("skipping XAddr with non-IP hostname %s: only local IP addresses are allowed", xaddr)
				continue
			}

			if !isLocalIP(ip) {
				logger.Warnf("skipping XAddr with external IP address %s: only local/private network addresses are allowed", xaddr)
				continue
			}

			xaddrs = append(xaddrs, parsedURL)
		}
	}

	return xaddrs
}

// TODO(Nick S): What happens if we don't do this?
func validWSDiscoveryInterface(iface net.Interface) bool {
	return iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 && iface.Flags&net.FlagLoopback == 0
}
