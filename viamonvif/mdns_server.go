package viamonvif

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"sync"

	"github.com/edaniels/zeroconf"
	"go.viam.com/rdk/logging"
	"go.viam.com/utils"
)

type mdnsInfo struct {
	ip      net.IP
	cleanup func()
}

type mdnsServer struct {
	mu            sync.Mutex
	mappedDevices map[string]mdnsInfo
	cacheFilepath string
	logger        logging.Logger
}

func newMDNSServer(logger logging.Logger) *mdnsServer {
	return newMDNSServerFromCachedData("", logger)
}

// newMDNSServerFromCachedData creates a new `mdnsServer` and initializes its set of mapped devices
// to what was in the cache file, if it exists. If the cacheFilepath is empty, UpdateCacheFile will
// be a no-op.
func newMDNSServerFromCachedData(cacheFilepath string, logger logging.Logger) *mdnsServer {
	ret := &mdnsServer{
		mappedDevices: make(map[string]mdnsInfo),
		cacheFilepath: cacheFilepath,
		logger:        logger,
	}
	if cacheFilepath != "" {
		entries := ret.readCacheFile()
		ret.apply(entries)
	}

	return ret
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

	avoidLintMagicNumberComplaintThatIsntEvenRelevant := 8080
	mdnsServer, err := zeroconf.RegisterProxy(
		serialNumber, // Dan: As far as I can tell, just a name.
		"_rtsp._tcp", // Dan: The mDNS "service" to register. Doesn't make a difference?
		"local",      // the domain
		avoidLintMagicNumberComplaintThatIsntEvenRelevant, // The service's port is ignored here
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

type cachedEntry struct {
	DNSName string `json:"dnsname"`
	IP      string `json:"ip"`
}

func (server *mdnsServer) readCacheFile() []cachedEntry {
	// If the `cacheFilepath` was given, read the contents and parse it as json. On any error we log
	// and return nil, which is interpreted as an empty slice.
	if server.cacheFilepath == "" {
		return nil
	}

	file, err := os.ReadFile(server.cacheFilepath)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		server.logger.Warn("Error reading dns cache file. File:", server.cacheFilepath, "Err:", err)
		return nil
	}

	var results []cachedEntry
	err = json.Unmarshal(file, &results)
	if err != nil {
		server.logger.Warn("Error unmarshalling JSON. File:", server.cacheFilepath, "Err:", err)
		return nil
	}

	return results
}

func (server *mdnsServer) apply(newEntries []cachedEntry) {
	// For each entry (presumably from the `cachedFilepath`), call `mdnsServer.Add`.
	for _, entry := range newEntries {
		ip := net.ParseIP(entry.IP)
		if ip == nil {
			server.logger.Warn("Unknown IP from cache file. DNSName:", entry.DNSName, "IP:", entry.IP)
			continue
		}

		server.logger.Debug("Adding DNS entry from cache file. DNSName:", entry.DNSName, "IP:", entry.IP)
		server.Add(entry.DNSName, ip)
	}
}

// UpdateCacheFile writes out all of the current mapped devices to a json file. Such that it can be
// read in after a viamrtsp restart to initialize the mdns server.
func (server *mdnsServer) UpdateCacheFile() {
	if server.cacheFilepath == "" {
		return
	}

	// Walk the `mappedDevices` and create a slice of `cachedEntry` objects that are intended to be
	// serialized to/from JSON.
	server.mu.Lock()
	var toWrite []cachedEntry
	for dnsname, info := range server.mappedDevices {
		toWrite = append(toWrite, cachedEntry{dnsname, info.ip.String()})
	}
	server.mu.Unlock()

	jsonBytes, err := json.Marshal(toWrite)
	if err != nil {
		server.logger.Warn("Error serializing dns entries. Err:", err)
	}

	file, err := os.Create(server.cacheFilepath)
	if err != nil {
		server.logger.Warn("Error creating dns cache file. File:", server.cacheFilepath, "Err:", err)
	}
	defer utils.UncheckedErrorFunc(file.Close)

	// Writing a byte array is expected to write out all bytes. Anything less will result in an
	// error. Thus we can ignore the number of bytes written return value.
	_, err = file.Write(jsonBytes)
	if err != nil {
		server.logger.Warn("Error writing dns entries to cache file. File:", server.cacheFilepath, "Err:", err)
	}
}

func (server *mdnsServer) Shutdown() {
	server.mu.Lock()
	defer server.mu.Unlock()

	for _, info := range server.mappedDevices {
		info.cleanup()
	}
	server.mappedDevices = make(map[string]mdnsInfo)
}
