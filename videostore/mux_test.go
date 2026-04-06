package videostore

import (
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"
)

func TestEnforceMonotonicTimestamps(t *testing.T) {
	t.Run("monotonic DTS passes through unchanged", func(t *testing.T) {
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 1000
		m.metadata.firstDTS = 900
		m.metadata.firstTimeStampsSet = true

		// First packet: DTS=900, PTS=1000 -> norm DTS=0, PTS=0
		pts, dts := m.enforceMonotonicTimestamps(1000, 900)
		test.That(t, dts, test.ShouldEqual, 0)
		test.That(t, pts, test.ShouldEqual, 0)

		// Second packet: DTS=950, PTS=1050 -> norm DTS=50, PTS=50
		pts, dts = m.enforceMonotonicTimestamps(1050, 950)
		test.That(t, dts, test.ShouldEqual, 50)
		test.That(t, pts, test.ShouldEqual, 50)

		// Third packet: DTS=1000, PTS=1100 -> norm DTS=100, PTS=100
		pts, dts = m.enforceMonotonicTimestamps(1100, 1000)
		test.That(t, dts, test.ShouldEqual, 100)
		test.That(t, pts, test.ShouldEqual, 100)
	})

	t.Run("non-monotonic DTS gets adjusted", func(t *testing.T) {
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 1000
		m.metadata.firstDTS = 900
		m.metadata.firstTimeStampsSet = true

		// First packet: norm DTS=0
		_, _ = m.enforceMonotonicTimestamps(1000, 900)

		// Second packet: DTS=950 -> norm DTS=50
		_, _ = m.enforceMonotonicTimestamps(1050, 950)

		// Third packet goes backwards: DTS=940 -> norm DTS=40, but lastDTS=50
		// Should be adjusted to 51
		pts, dts := m.enforceMonotonicTimestamps(1040, 940)
		test.That(t, dts, test.ShouldEqual, 51)
		// PTS (norm 40) < DTS (51), so PTS should also be bumped
		test.That(t, pts, test.ShouldEqual, 51)
	})

	t.Run("equal DTS gets adjusted", func(t *testing.T) {
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 0
		m.metadata.firstDTS = 0
		m.metadata.firstTimeStampsSet = true

		_, _ = m.enforceMonotonicTimestamps(100, 100)

		// Same DTS as previous (100 == 100) -> should become 101
		pts, dts := m.enforceMonotonicTimestamps(100, 100)
		test.That(t, dts, test.ShouldEqual, 101)
		test.That(t, pts, test.ShouldEqual, 101)
	})

	t.Run("PTS stays above DTS when already higher", func(t *testing.T) {
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 0
		m.metadata.firstDTS = 0
		m.metadata.firstTimeStampsSet = true

		// PTS=200, DTS=100 -> PTS should remain 200 (not clamped down to DTS)
		pts, dts := m.enforceMonotonicTimestamps(200, 100)
		test.That(t, dts, test.ShouldEqual, 100)
		test.That(t, pts, test.ShouldEqual, 200)
	})

	t.Run("multiple consecutive non-monotonic packets", func(t *testing.T) {
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 0
		m.metadata.firstDTS = 0
		m.metadata.firstTimeStampsSet = true

		// Normal packet at DTS=100
		_, _ = m.enforceMonotonicTimestamps(100, 100)

		// Three consecutive backwards packets
		_, dts := m.enforceMonotonicTimestamps(50, 50)
		test.That(t, dts, test.ShouldEqual, 101)

		_, dts = m.enforceMonotonicTimestamps(50, 50)
		test.That(t, dts, test.ShouldEqual, 102)

		_, dts = m.enforceMonotonicTimestamps(50, 50)
		test.That(t, dts, test.ShouldEqual, 103)

		// Normal packet resumes above the adjusted values
		_, dts = m.enforceMonotonicTimestamps(200, 200)
		test.That(t, dts, test.ShouldEqual, 200)
	})

	t.Run("first timestamps are subtracted to zero-base", func(t *testing.T) {
		// Verifies that large absolute PTS/DTS values get normalized by subtracting
		// firstPTS/firstDTS, so the segmenter starts from zero.
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 90000
		m.metadata.firstDTS = 85000
		m.metadata.firstTimeStampsSet = true

		// First packet: PTS=90000, DTS=85000 -> normPTS=0, normDTS=0
		pts, dts := m.enforceMonotonicTimestamps(90000, 85000)
		test.That(t, dts, test.ShouldEqual, 0)
		test.That(t, pts, test.ShouldEqual, 0)

		// Second packet: PTS=93600, DTS=88600 -> normPTS=3600, normDTS=3600
		pts, dts = m.enforceMonotonicTimestamps(93600, 88600)
		test.That(t, dts, test.ShouldEqual, 3600)
		test.That(t, pts, test.ShouldEqual, 3600)

		// Verify PTS/DTS difference is preserved: PTS=95000, DTS=89000 -> normPTS=5000, normDTS=4000
		pts, dts = m.enforceMonotonicTimestamps(95000, 89000)
		test.That(t, dts, test.ShouldEqual, 4000)
		test.That(t, pts, test.ShouldEqual, 5000)
	})

	t.Run("real world scenario from ticket", func(t *testing.T) {
		// Simulates the actual error from the ticket:
		// DTS sequence: 2454, 2457, 2454 (non-monotonic), 2456 (non-monotonic), 2946 (equal)
		logger := logging.NewTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.metadata.firstPTS = 2454
		m.metadata.firstDTS = 2454
		m.metadata.firstTimeStampsSet = true

		// DTS=2454 -> norm 0
		_, dts := m.enforceMonotonicTimestamps(2454, 2454)
		test.That(t, dts, test.ShouldEqual, 0)

		// DTS=2457 -> norm 3
		_, dts = m.enforceMonotonicTimestamps(2457, 2457)
		test.That(t, dts, test.ShouldEqual, 3)

		// DTS=2454 -> norm 0, but lastDTS=3 -> adjusted to 4
		_, dts = m.enforceMonotonicTimestamps(2454, 2454)
		test.That(t, dts, test.ShouldEqual, 4)

		// DTS=2456 -> norm 2, but lastDTS=4 -> adjusted to 5
		_, dts = m.enforceMonotonicTimestamps(2456, 2456)
		test.That(t, dts, test.ShouldEqual, 5)

		// DTS=2946 -> norm 492 (fine, > 5)
		_, dts = m.enforceMonotonicTimestamps(2946, 2946)
		test.That(t, dts, test.ShouldEqual, 492)
	})
}
