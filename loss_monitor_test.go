package viamrtsp

import (
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

func newTestLossMonitor(t *testing.T, transport string) (*lossMonitor, *observer.ObservedLogs) {
	t.Helper()
	logger, logs := logging.NewObservedTestLogger(t)
	lm := &lossMonitor{
		logger:    logger,
		name:      "cam1",
		url:       "rtsp://camera.local:554/stream",
		transport: transport,
	}
	return lm, logs
}

func logsAtLevel(logs *observer.ObservedLogs, level zapcore.Level) []observer.LoggedEntry {
	var out []observer.LoggedEntry
	for _, entry := range logs.All() {
		if entry.Level == level {
			out = append(out, entry)
		}
	}
	return out
}

func TestLossMonitorBaselineAndCleanWindows(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	// The first sample only establishes a baseline; clean windows after it stay quiet.
	lm.observe(rtpStatsSnapshot{packetsReceived: 1000})
	lm.observe(rtpStatsSnapshot{packetsReceived: 2000})
	lm.observe(rtpStatsSnapshot{packetsReceived: 3000})

	test.That(t, logsAtLevel(logs, zapcore.WarnLevel), test.ShouldHaveLength, 0)
	test.That(t, logsAtLevel(logs, zapcore.InfoLevel), test.ShouldHaveLength, 0)
	// Every complete window still emits a debug summary for support bundles.
	test.That(t, logsAtLevel(logs, zapcore.DebugLevel), test.ShouldHaveLength, 2)
}

func TestLossMonitorTransientLossLogsInfo(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	lm.observe(rtpStatsSnapshot{packetsReceived: 10000})
	// 10 lost out of 10010 is 0.1%, under both warn thresholds.
	lm.observe(rtpStatsSnapshot{packetsReceived: 20000, packetsLost: 10})

	test.That(t, logsAtLevel(logs, zapcore.WarnLevel), test.ShouldHaveLength, 0)
	infos := logsAtLevel(logs, zapcore.InfoLevel)
	test.That(t, infos, test.ShouldHaveLength, 1)
	test.That(t, infos[0].Message, test.ShouldContainSubstring, "lost 10 RTP packets")
}

func TestLossMonitorTCPWarn(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	lm.observe(rtpStatsSnapshot{packetsReceived: 1000})
	// 500 lost out of 10500 is ~4.8%, over the warn threshold.
	lm.observe(rtpStatsSnapshot{packetsReceived: 11000, packetsLost: 500, packetsInError: 7})

	warns := logsAtLevel(logs, zapcore.WarnLevel)
	test.That(t, warns, test.ShouldHaveLength, 1)
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "sequence gaps detected over TCP")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "socket buffer overflow")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "Lost 500 packets")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "7 packets could not be processed")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "rtsp://camera.local:554/stream")
}

func TestLossMonitorUDPWarn(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "UDP")
	lm.setVideoInfo(3840, 2160, 30)

	lm.observe(rtpStatsSnapshot{packetsReceived: 1000})
	lm.observe(rtpStatsSnapshot{packetsReceived: 11000, packetsLost: 500})

	warns := logsAtLevel(logs, zapcore.WarnLevel)
	test.That(t, warns, test.ShouldHaveLength, 1)
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "network congestion or socket buffer limits")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "lower-resolution stream")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "3840x2160@30.0fps")
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "transport UDP")
	test.That(t, warns[0].Message, test.ShouldNotContainSubstring, "sequence gaps detected over TCP")
}

func TestLossMonitorPersistentLossIsRateLimited(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	received := uint64(0)
	lost := uint64(0)
	step := func() {
		received += 10000
		lost += 500
		lm.observe(rtpStatsSnapshot{packetsReceived: received, packetsLost: lost})
	}

	step() // baseline
	// First lossy window warns; the following lossRewarnWindows-1 windows are suppressed,
	// then the next lossy window warns again.
	for range lossRewarnWindows + 1 {
		step()
	}

	warns := logsAtLevel(logs, zapcore.WarnLevel)
	test.That(t, warns, test.ShouldHaveLength, 2)
}

func TestLossMonitorEscalationLogsImmediately(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	lm.observe(rtpStatsSnapshot{packetsReceived: 10000})
	// Window 1: transient loss, logs info.
	lm.observe(rtpStatsSnapshot{packetsReceived: 20000, packetsLost: 10})
	// Window 2: escalates past the warn threshold; must not be suppressed by rate limiting.
	lm.observe(rtpStatsSnapshot{packetsReceived: 30000, packetsLost: 510})

	test.That(t, logsAtLevel(logs, zapcore.InfoLevel), test.ShouldHaveLength, 1)
	test.That(t, logsAtLevel(logs, zapcore.WarnLevel), test.ShouldHaveLength, 1)
}

func TestLossMonitorRecoveryLogsOnceAfterWarn(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	lm.observe(rtpStatsSnapshot{packetsReceived: 1000})
	lm.observe(rtpStatsSnapshot{packetsReceived: 11000, packetsLost: 500})
	// Two clean windows: recovery is logged once, not repeatedly.
	lm.observe(rtpStatsSnapshot{packetsReceived: 21000, packetsLost: 500})
	lm.observe(rtpStatsSnapshot{packetsReceived: 31000, packetsLost: 500})

	var recoveries []string
	for _, entry := range logsAtLevel(logs, zapcore.InfoLevel) {
		if strings.Contains(entry.Message, "recovered") {
			recoveries = append(recoveries, entry.Message)
		}
	}
	test.That(t, recoveries, test.ShouldHaveLength, 1)
}

func TestLossMonitorNoRecoveryLogAfterTransientLoss(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	lm.observe(rtpStatsSnapshot{packetsReceived: 10000})
	lm.observe(rtpStatsSnapshot{packetsReceived: 20000, packetsLost: 10})
	lm.observe(rtpStatsSnapshot{packetsReceived: 30000, packetsLost: 10})

	for _, entry := range logsAtLevel(logs, zapcore.InfoLevel) {
		test.That(t, entry.Message, test.ShouldNotContainSubstring, "recovered")
	}
}

func TestLossMonitorCounterResetOnReconnect(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	lm.observe(rtpStatsSnapshot{packetsReceived: 50000, packetsLost: 5000})
	// Reconnect: the new client's counters restart from zero.
	lm.setConnection("UDP")
	lm.observe(rtpStatsSnapshot{packetsReceived: 100})
	lm.observe(rtpStatsSnapshot{packetsReceived: 200})

	test.That(t, logsAtLevel(logs, zapcore.WarnLevel), test.ShouldHaveLength, 0)
	test.That(t, lm.transport, test.ShouldEqual, "UDP")
}

func TestLossMonitorCounterResetWithoutSetConnection(t *testing.T) {
	lm, logs := newTestLossMonitor(t, "TCP")

	// Even without an explicit reset, a counter going backwards must re-baseline
	// instead of underflowing into a huge bogus loss count.
	lm.observe(rtpStatsSnapshot{packetsReceived: 50000, packetsLost: 5000})
	lm.observe(rtpStatsSnapshot{packetsReceived: 100, packetsLost: 0})
	lm.observe(rtpStatsSnapshot{packetsReceived: 200, packetsLost: 0})

	test.That(t, logsAtLevel(logs, zapcore.WarnLevel), test.ShouldHaveLength, 0)
}

func TestLossMonitorSetVideoInfoIgnoresZeroes(t *testing.T) {
	lm, _ := newTestLossMonitor(t, "TCP")

	lm.setVideoInfo(1920, 1080, 25)
	// An SPS without timing info must not clobber a known frame rate.
	lm.setVideoInfo(3840, 2160, 0)

	lm.mu.Lock()
	defer lm.mu.Unlock()
	test.That(t, lm.width, test.ShouldEqual, 3840)
	test.That(t, lm.height, test.ShouldEqual, 2160)
	test.That(t, lm.fps, test.ShouldEqual, 25.0)
}
