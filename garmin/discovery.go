package garmin

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/viamrobotics/zeroconf"
	"go.viam.com/rdk/logging"
)

const (
	// mdnsService is the DNS-SD service type Garmin marine cameras advertise.
	mdnsService = "_garmin-mrn-svcm._tcp"
	// mdnsDomain is the mDNS domain to browse.
	mdnsDomain = "local"
	// browseTimeout bounds how long we listen for mDNS responses during discovery.
	browseTimeout = 5 * time.Second
	// entriesBufferSize is the buffer size for the channel zeroconf pushes entries into.
	entriesBufferSize = 100
)

// Camera is a single Garmin camera discovered over mDNS.
type Camera struct {
	// Instance is the mDNS instance name, e.g. "garmin-cv28-copepod-3530195020".
	Instance string
	// Host is the advertised hostname without the trailing dot, e.g.
	// "garmin-cv28-copepod-3530195020.local". May be empty if not advertised.
	Host string
	// IPs are the resolved IPv4 addresses for the host.
	IPs []net.IP
	// Text holds the raw TXT records, retained for future enrichment.
	Text []string
}

// addressHost returns the host to use when building an RTSP URL. We prefer the
// camera's self-advertised mDNS hostname (Garmin advertises a stable *.local
// name), falling back to the first resolved IPv4 address.
func (c Camera) addressHost() string {
	if c.Host != "" {
		return c.Host
	}
	if len(c.IPs) > 0 {
		return c.IPs[0].String()
	}
	return ""
}

// browseGarmin browses the local network over mDNS for Garmin cameras for the
// duration of browseTimeout and returns the deduplicated set found.
func browseGarmin(ctx context.Context, logger logging.Logger) ([]Camera, error) {
	// zeroconf expects a *zap.SugaredLogger; massage the viam logger into one.
	resolver, err := zeroconf.NewResolver(logger.Desugar().Sugar())
	if err != nil {
		return nil, err
	}
	defer resolver.Shutdown()

	entries := make(chan *zeroconf.ServiceEntry, entriesBufferSize)
	browseCtx, cancel := context.WithTimeout(ctx, browseTimeout)
	defer cancel()

	if err := resolver.Browse(browseCtx, mdnsService, mdnsDomain, entries); err != nil {
		return nil, err
	}

	// Collect until the browse window closes. zeroconf may emit the same instance
	// multiple times; dedupe by instance name, preferring entries that carry a
	// resolved hostname/IP.
	found := make(map[string]Camera)
	for {
		select {
		case <-browseCtx.Done():
			cams := make([]Camera, 0, len(found))
			for _, c := range found {
				cams = append(cams, c)
			}
			logger.Debugf("garmin mDNS browse found %d camera(s)", len(cams))
			return cams, nil
		case entry := <-entries:
			if entry == nil {
				continue
			}
			cam := serviceEntryToCamera(entry)
			existing, ok := found[cam.Instance]
			if !ok || (existing.addressHost() == "" && cam.addressHost() != "") {
				found[cam.Instance] = cam
			}
		}
	}
}

// serviceEntryToCamera converts a zeroconf ServiceEntry into a Camera.
func serviceEntryToCamera(entry *zeroconf.ServiceEntry) Camera {
	return Camera{
		Instance: entry.Instance,
		Host:     strings.TrimSuffix(entry.HostName, "."),
		IPs:      entry.AddrIPv4,
		Text:     entry.Text,
	}
}
