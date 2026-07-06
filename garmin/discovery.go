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
	Instance string `json:"instance"`
	// Host is the advertised hostname without the trailing dot, e.g.
	// "garmin-cv28-copepod-3530195020.local". May be empty if not advertised.
	Host string `json:"host"`
	// IPs are the resolved IPv4 addresses for the host.
	IPs []net.IP `json:"ips"`
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

	return collectCameras(browseCtx, entries, logger), nil
}

// collectCameras drains ServiceEntries until the browse window closes or the
// entries channel is closed, deduping by instance name and preferring entries
// that carry a resolved hostname/IP.
func collectCameras(ctx context.Context, entries <-chan *zeroconf.ServiceEntry, logger logging.Logger) []Camera {
	found := make(map[string]Camera)
	add := func(entry *zeroconf.ServiceEntry) {
		if entry == nil {
			return
		}
		cam := serviceEntryToCamera(entry)
		if existing, seen := found[cam.Instance]; !seen || (existing.addressHost() == "" && cam.addressHost() != "") {
			found[cam.Instance] = cam
		}
	}
	finalize := func() []Camera {
		cams := make([]Camera, 0, len(found))
		for _, c := range found {
			logger.Debugf("discovered garmin camera: instance=%s host=%s ips=%v", c.Instance, c.Host, c.IPs)
			cams = append(cams, c)
		}
		logger.Debugf("garmin mDNS browse found %d camera(s)", len(cams))
		return cams
	}

	for {
		select {
		case <-ctx.Done():
			// The browse window closed. Drain whatever is still buffered before
			// finalizing: select picks uniformly at random among ready cases, so
			// otherwise a camera that responded right at the deadline could be
			// dropped whenever this Done case wins over a ready receive.
			for {
				select {
				case entry, ok := <-entries:
					if !ok {
						return finalize()
					}
					add(entry)
				default:
					return finalize()
				}
			}
		case entry, ok := <-entries:
			// zeroconf closes the channel when the browse ends or on a transient
			// error (via params.done()). A single-value receive can't tell a
			// closed channel from a nil entry, so check ok explicitly: otherwise
			// a close while ctx is still live spins the loop at 100% CPU.
			if !ok {
				return finalize()
			}
			add(entry)
		}
	}
}

// serviceEntryToCamera converts a zeroconf ServiceEntry into a Camera.
func serviceEntryToCamera(entry *zeroconf.ServiceEntry) Camera {
	return Camera{
		Instance: entry.Instance,
		Host:     strings.TrimSuffix(entry.HostName, "."),
		IPs:      entry.AddrIPv4,
	}
}
