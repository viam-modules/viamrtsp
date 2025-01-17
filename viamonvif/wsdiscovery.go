package viamonvif

import (
	"context"
	"fmt"
	"maps"
	"net"
	"net/url"
	"slices"

	"go.viam.com/rdk/logging"
)

// WSDiscovery runs WS-Discovery on the network interface.
func WSDiscovery(ctx context.Context, logger logging.Logger, iface net.Interface) ([]*url.URL, error) {
	logger.Debugf("WS-Discovery starting on interface: %s\n", iface.Name)
	defer logger.Debugf("WS-Discovery stopping on interface: %s\n", iface.Name)
	if !validWSDiscoveryInterface(iface) {
		return nil, fmt.Errorf("WS-Discovery skipping interface %s: does not meet WS-Discovery requirements", iface.Name)
	}
	var discoveryResps []string
	attempts := 3 // run ws-discovery probe 3 times due to sync flakiness between announcer and requester
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

// TODO(Nick S): What happens if we don't do this?
func validWSDiscoveryInterface(iface net.Interface) bool {
	return iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 && iface.Flags&net.FlagLoopback == 0
}
