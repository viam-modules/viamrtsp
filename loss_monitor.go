package viamrtsp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.viam.com/rdk/logging"
)

const (
	// lossReportIntervalSeconds is how often the loss monitor samples client stats and evaluates a window.
	lossReportIntervalSeconds = 30
	// lossWarnPercent is the per-window loss percentage at which loss escalates to a warning.
	lossWarnPercent = 1.0
	// lossWarnMinPackets is the per-window lost-packet count at which loss escalates to a warning
	// even if it is under lossWarnPercent.
	lossWarnMinPackets = 100
	// lossRewarnWindows is how many windows a persistent-loss stream stays quiet between repeated
	// log lines (10 windows * 30s = one reminder every ~5 minutes).
	lossRewarnWindows = 10
	// percentFactor converts a ratio to a percentage.
	percentFactor = 100
)

var lossReportIntervalDuration = lossReportIntervalSeconds * time.Second

// rtpStatsSnapshot holds cumulative counters sampled from gortsplib's Client.Stats(). Counters
// restart at zero whenever the client is replaced on reconnect.
type rtpStatsSnapshot struct {
	packetsReceived uint64
	packetsLost     uint64
	packetsInError  uint64
	bytesReceived   uint64
	jitter          float64
}

// lossMonitor turns cumulative RTP stats into windowed packet-loss reports, escalating from debug
// summaries to actionable warnings as loss gets worse. Sequence gaps over TCP get a more specific
// message: TCP does not lose packets in transit, so a gap means the camera dropped packets before
// sending, typically because the reader can't drain the stream fast enough (socket buffer
// backpressure) — the classic high-bitrate/4K failure mode.
type lossMonitor struct {
	logger logging.Logger

	mu        sync.Mutex
	name      string
	url       string // credentials stripped, safe for logs
	transport string
	width     int
	height    int
	fps       float64

	prev *rtpStatsSnapshot
	// lossWindows counts consecutive windows with nonzero loss; 0 means healthy.
	lossWindows int
	// warned is whether the current loss run has logged at warning severity.
	warned          bool
	windowsSinceLog int
}

// setConnection records the transport of a newly established connection and resets all window
// state, since the new client's counters restart at zero.
func (lm *lossMonitor) setConnection(transport string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.transport = transport
	lm.prev = nil
	lm.lossWindows = 0
	lm.warned = false
	lm.windowsSinceLog = 0
}

// setVideoInfo records stream resolution and frame rate for log context. Zero values are ignored
// so an SPS that parses without timing info doesn't clobber a config-provided frame rate.
func (lm *lossMonitor) setVideoInfo(width, height int, fps float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if width > 0 && height > 0 {
		lm.width = width
		lm.height = height
	}
	if fps > 0 {
		lm.fps = fps
	}
}

// describeStreamLocked returns log context like
// "rtsp://host:554/stream, 3840x2160@30.0fps, transport TCP". lm.mu must be held.
func (lm *lossMonitor) describeStreamLocked() string {
	parts := []string{lm.url}
	if lm.width > 0 && lm.height > 0 {
		res := fmt.Sprintf("%dx%d", lm.width, lm.height)
		if lm.fps > 0 {
			res += fmt.Sprintf("@%.1ffps", lm.fps)
		}
		parts = append(parts, res)
	} else if lm.fps > 0 {
		parts = append(parts, fmt.Sprintf("%.1ffps", lm.fps))
	}
	if lm.transport != "" {
		parts = append(parts, "transport "+lm.transport)
	}
	return strings.Join(parts, ", ")
}

// observe evaluates one stats sample against the previous one and logs accordingly.
func (lm *lossMonitor) observe(curr rtpStatsSnapshot) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	prev := lm.prev
	lm.prev = &curr
	if prev == nil || curr.packetsReceived < prev.packetsReceived || curr.packetsLost < prev.packetsLost {
		// First sample on this connection, or counters restarted; establish a baseline.
		return
	}

	received := curr.packetsReceived - prev.packetsReceived
	lost := curr.packetsLost - prev.packetsLost
	decodeErrors := curr.packetsInError - prev.packetsInError
	bytesReceived := curr.bytesReceived - prev.bytesReceived

	var lossPct float64
	if lost+received > 0 {
		lossPct = float64(lost) / float64(lost+received) * percentFactor
	}

	lm.logger.Debugf("rtsp stats window (%s): received=%d lost=%d (%.2f%%) decodeErrors=%d bitrate=%.0fkbps jitter=%.1f",
		lm.describeStreamLocked(), received, lost, lossPct, decodeErrors,
		float64(bytesReceived)*8/1000/lossReportIntervalSeconds, curr.jitter)

	if lost == 0 {
		if lm.warned {
			lm.logger.Infof("RTSP stream %q (%s): packet loss recovered, no packets lost in the last %ds",
				lm.name, lm.describeStreamLocked(), lossReportIntervalSeconds)
		}
		lm.lossWindows = 0
		lm.warned = false
		lm.windowsSinceLog = 0
		return
	}

	lm.lossWindows++
	warnLevel := lossPct >= lossWarnPercent || lost >= lossWarnMinPackets

	// Log the first lossy window of a run and any escalation to warning severity immediately;
	// after that, repeat only every lossRewarnWindows windows so a persistently lossy stream
	// doesn't flood the logs.
	logNow := lm.lossWindows == 1 ||
		(warnLevel && !lm.warned) ||
		lm.windowsSinceLog >= lossRewarnWindows-1
	if !logNow {
		lm.windowsSinceLog++
		return
	}
	lm.windowsSinceLog = 0

	if !warnLevel {
		lm.logger.Infof("RTSP stream %q (%s) lost %d RTP packets in the last %ds (%.2f%% of packets)",
			lm.name, lm.describeStreamLocked(), lost, lossReportIntervalSeconds, lossPct)
		return
	}
	lm.warned = true

	var corruption string
	if decodeErrors > 0 {
		corruption = fmt.Sprintf(" %d packets could not be processed in the same period; video corruption is likely.", decodeErrors)
	} else {
		corruption = " Video corruption may occur."
	}

	if strings.EqualFold(lm.transport, "tcp") {
		lm.logger.Warnf("RTSP stream %q (%s): RTP sequence gaps detected over TCP — the camera is dropping packets, "+
			"likely because this machine cannot read the stream fast enough (socket buffer overflow / reader backpressure). "+
			"Lost %d packets in the last %ds (%.2f%%).%s Try lowering the camera resolution/bitrate or frame rate. "+
			"If that isn't possible, contact support to discuss advanced socket configuration.",
			lm.name, lm.describeStreamLocked(), lost, lossReportIntervalSeconds, lossPct, corruption)
		return
	}
	lm.logger.Warnf("RTSP stream %q (%s) lost %d RTP packets in the last %ds (%.2f%% of packets), likely due to "+
		"network congestion or socket buffer limits.%s Suggested next steps: use a lower-resolution stream "+
		"(e.g. 1080p instead of 4K), reduce the camera's frame rate/bitrate, or check that the camera and this "+
		"machine are on a stable, low-latency network segment.",
		lm.name, lm.describeStreamLocked(), lost, lossReportIntervalSeconds, lossPct, corruption)
}
