package viamonvif

import (
	"net"
	"sync"

	"github.com/edaniels/zeroconf"
	"go.viam.com/rdk/logging"
)

type mdnsInfo struct {
	ip      net.IP
	cleanup func()
}

type mdnsServer struct {
	mu            sync.Mutex
	mappedDevices map[string]mdnsInfo
	logger        logging.Logger
}

func newMDNSServer(logger logging.Logger) mdnsServer {
	return mdnsServer{
		mappedDevices: make(map[string]mdnsInfo),
		logger:        logger,
	}
}

// serialNumber is expected to be a device serial number. Without any `.local` TLD suffix. The IP is
// the IP to be mapped to. An error is returned if the mdns mapping failed. Informing the caller
// that doing an ip -> hostname substitution is not expected to work.
func (server *mdnsServer) Add(serialNumber string, ip net.IP) error {
	server.mu.Lock()
	defer server.mu.Unlock()

	if info, exists := server.mappedDevices[serialNumber]; exists {
		if info.ip.Equal(ip) {
			// We're already managing an mdns entry for the same hostname -> ip mapping.
			return nil
		}

		info.cleanup()
	}
	delete(server.mappedDevices, serialNumber)

	mdnsServer, err := zeroconf.RegisterProxy(
		serialNumber,          // Dan: As far as I can tell, just a name.
		"_rtsp._tcp",          // Dan: The mDNS "service" to register. Doesn't make a difference?
		"local",               // the domain
		8080,                  // The service's port is ignored here
		serialNumber,          // actual mDNS hostname, without the .local domain
		[]string{ip.String()}, // ip to use
		[]string{},            // txt fields, not needed
		nil,                   // resolve this name for requests from all network interfaces
		// RSDK-8205: logger.Desugar().Sugar() is necessary to massage a ZapCompatibleLogger into a
		// *zap.SugaredLogger to match zeroconf function signatures.
		server.logger.Desugar().Sugar(),
	)
	if err != nil {
		return err
	}

	server.mappedDevices[serialNumber] = mdnsInfo{
		ip:      ip,
		cleanup: mdnsServer.Shutdown,
	}

	return nil
}

func (server *mdnsServer) Remove(hostname string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	info, exists := server.mappedDevices[hostname]
	if !exists {
		return
	}

	info.cleanup()
	delete(server.mappedDevices, hostname)
}

func (server *mdnsServer) Shutdown() {
	server.mu.Lock()
	defer server.mu.Unlock()

	for _, info := range server.mappedDevices {
		info.cleanup()
	}
	server.mappedDevices = make(map[string]mdnsInfo)
}
