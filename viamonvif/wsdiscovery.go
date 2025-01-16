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

func WSDiscovery(ctx context.Context, logger logging.Logger, iface net.Interface) ([]*url.URL, error) {
	if !validWSDiscoveryInterface(iface) {
		return nil, fmt.Errorf("WS-Discovery skipping interface %s: does not meet WS-Discovery requirements", iface.Name)

	}
	logger.Debugf("sending WS-Discovery probe using interface: %s", iface.Name)
	var discoveryResps []string
	attempts := 3 // run ws-discovery probe 3 times due to sync flakiness between announcer and requester
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context canceled for interface: %s", iface.Name)
		}

		resp, err := SendProbe(iface.Name)
		if err != nil {
			logger.Debugf("attempt %d: failed to send WS-Discovery probe on interface %s: %w", i+1, iface.Name, err)
			continue
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
	addrs, err := iface.Addrs()
	if err != nil {
		panic(err)
	}
	addrsNetworksStr := []string{}
	addrsStr := []string{}
	for _, a := range addrs {
		addrsNetworksStr = append(addrsNetworksStr, a.Network())
		addrsStr = append(addrsStr, a.String())
	}

	multiAddrs, err := iface.MulticastAddrs()
	if err != nil {
		panic(err)
	}
	multiAddrsNetworkStr := []string{}
	multiAddrsStr := []string{}
	for _, a := range multiAddrs {
		addrsNetworksStr = append(multiAddrsNetworkStr, a.Network())
		multiAddrsStr = append(multiAddrsStr, a.String())
	}
	// log.Printf("iface: %s, FlagUp: %d, FlagBroadcast: %d, FlagLoopback: %d, FlagPointToPoint: %d, FlagMulticast: %d, FlagRunning: %d, flags: %s, "+
	// 	"addrs: %#v, addrsNetworks: %#v, multicastaddrs: %#v, multicastaddrsNetworks: %#v\n", iface.Name, iface.Flags&net.FlagUp, iface.Flags&net.FlagBroadcast, iface.Flags&net.FlagLoopback, iface.Flags&net.FlagPointToPoint, iface.Flags&net.FlagMulticast, iface.Flags&net.FlagRunning, iface.Flags.String(),
	// 	addrsStr, addrsNetworksStr, multiAddrsStr, multiAddrsNetworkStr)
	return iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 && iface.Flags&net.FlagLoopback == 0
}
