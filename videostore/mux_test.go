package videostore

import (
	"testing"

	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/viam-modules/video-store/videostore"
	"go.uber.org/zap/zapcore"
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

// h265NALU builds a two-byte H.265 NAL unit header (forbidden_zero_bit=0, nuh_layer_id=0,
// nuh_temporal_id_plus1=1) for the given type, followed by payload.
func h265NALU(typ h265.NALUType, payload ...byte) []byte {
	return append([]byte{byte(typ) << 1, 0x01}, payload...)
}

// h264NALU builds a one-byte H.264 NAL unit header (forbidden_zero_bit=0, nal_ref_idc=3) for the
// given type, followed by payload.
func h264NALU(typ h264.NALUType, payload ...byte) []byte {
	return append([]byte{0x60 | byte(typ)}, payload...)
}

func droppedTypes(d []droppedNALU) []uint8 {
	out := make([]uint8, 0, len(d))
	for _, x := range d {
		out = append(out, x.typ)
	}
	return out
}

func TestFilterH265AU(t *testing.T) {
	sps := h265NALU(h265.NALUType_SPS_NUT, 0xAA)
	pps := h265NALU(h265.NALUType_PPS_NUT, 0xBB)
	vps := h265NALU(h265.NALUType_VPS_NUT, 0xCC)
	idr := h265NALU(h265.NALUType_IDR_W_RADL, 0x01, 0x02)
	trail := h265NALU(h265.NALUType_TRAIL_R, 0x03)

	t.Run("unspecified type 51 is dropped and the rest of the AU is kept", func(t *testing.T) {
		// This is the RSDK-14390 shape: vendor metadata riding on a keyframe. Before the fix the
		// whole AU was rejected with "invalid nalu" and the DTS extractor never initialized.
		private := h265NALU(51, make([]byte, 35)...)
		info := filterH265AU([][]byte{sps, private, idr})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{idr})
		test.That(t, info.sps, test.ShouldResemble, sps)
		test.That(t, info.isRandomAccess, test.ShouldBeTrue)
		test.That(t, info.dropped, test.ShouldResemble, []droppedNALU{{typ: 51, size: 37}})
	})

	t.Run("reserved VCL 24 and unspecified 63 are dropped", func(t *testing.T) {
		info := filterH265AU([][]byte{trail, h265NALU(24), h265NALU(63)})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{trail})
		test.That(t, droppedTypes(info.dropped), test.ShouldResemble, []uint8{24, 63})
		test.That(t, info.isRandomAccess, test.ShouldBeFalse)
	})

	t.Run("reserved non-VCL 41 and 47 are dropped", func(t *testing.T) {
		info := filterH265AU([][]byte{h265NALU(41), trail, h265NALU(47)})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{trail})
		test.That(t, droppedTypes(info.dropped), test.ShouldResemble, []uint8{41, 47})
	})

	t.Run("empty NALUs are skipped without being reported", func(t *testing.T) {
		info := filterH265AU([][]byte{nil, {}, trail})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{trail})
		test.That(t, info.dropped, test.ShouldBeNil)
	})

	t.Run("AU containing only an unrecognized NALU yields nothing to write", func(t *testing.T) {
		info := filterH265AU([][]byte{h265NALU(55, 0x00)})
		test.That(t, info.filtered, test.ShouldBeNil)
		test.That(t, info.isRandomAccess, test.ShouldBeFalse)
		test.That(t, droppedTypes(info.dropped), test.ShouldResemble, []uint8{55})
	})

	t.Run("parameter sets and AUD are stripped, parameter sets are captured", func(t *testing.T) {
		aud := h265NALU(h265.NALUType_AUD_NUT, 0x50)
		idrNLP := h265NALU(h265.NALUType_IDR_N_LP, 0x04)
		info := filterH265AU([][]byte{aud, vps, sps, pps, idrNLP})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{idrNLP})
		test.That(t, info.vps, test.ShouldResemble, vps)
		test.That(t, info.sps, test.ShouldResemble, sps)
		test.That(t, info.pps, test.ShouldResemble, pps)
		test.That(t, info.isRandomAccess, test.ShouldBeTrue)
		test.That(t, info.dropped, test.ShouldBeNil)
	})

	t.Run("CRA is random access, TRAIL_R is not", func(t *testing.T) {
		cra := h265NALU(h265.NALUType_CRA_NUT, 0x05)
		test.That(t, filterH265AU([][]byte{cra}).isRandomAccess, test.ShouldBeTrue)
		test.That(t, filterH265AU([][]byte{trail}).isRandomAccess, test.ShouldBeFalse)
	})

	t.Run("known non-VCL NALUs such as SEI are kept", func(t *testing.T) {
		sei := h265NALU(h265.NALUType_PREFIX_SEI_NUT, 0x06)
		info := filterH265AU([][]byte{sei, trail})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{sei, trail})
		test.That(t, info.dropped, test.ShouldBeNil)
	})

	t.Run("parameter sets absent from the AU are left nil", func(t *testing.T) {
		info := filterH265AU([][]byte{trail})
		test.That(t, info.vps, test.ShouldBeNil)
		test.That(t, info.sps, test.ShouldBeNil)
		test.That(t, info.pps, test.ShouldBeNil)
	})
}

func TestFilterH264AU(t *testing.T) {
	sps := h264NALU(h264.NALUTypeSPS, 0xAA)
	pps := h264NALU(h264.NALUTypePPS, 0xBB)
	idr := h264NALU(h264.NALUTypeIDR, 0x01)
	nonIDR := h264NALU(h264.NALUTypeNonIDR, 0x02)

	t.Run("unspecified types 0, 30 and 31 are dropped and the slice is kept", func(t *testing.T) {
		info := filterH264AU([][]byte{h264NALU(0, 0x00), idr, h264NALU(30), h264NALU(31, 0x00, 0x00)})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{idr})
		test.That(t, info.idrPresent, test.ShouldBeTrue)
		test.That(t, info.nonIDRPresent, test.ShouldBeFalse)
		test.That(t, info.dropped, test.ShouldResemble, []droppedNALU{{typ: 0, size: 2}, {typ: 30, size: 1}, {typ: 31, size: 3}})
	})

	t.Run("SPS, PPS and AUD are stripped and parameter sets captured", func(t *testing.T) {
		aud := h264NALU(h264.NALUTypeAccessUnitDelimiter, 0xF0)
		info := filterH264AU([][]byte{aud, sps, pps, nonIDR})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{nonIDR})
		test.That(t, info.sps, test.ShouldResemble, sps)
		test.That(t, info.pps, test.ShouldResemble, pps)
		test.That(t, info.nonIDRPresent, test.ShouldBeTrue)
		test.That(t, info.idrPresent, test.ShouldBeFalse)
		test.That(t, info.dropped, test.ShouldBeNil)
	})

	t.Run("empty NALUs are skipped without being reported", func(t *testing.T) {
		info := filterH264AU([][]byte{{}, nonIDR, nil})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{nonIDR})
		test.That(t, info.dropped, test.ShouldBeNil)
	})

	t.Run("known non-slice NALUs such as SEI are kept but do not count as slices", func(t *testing.T) {
		sei := h264NALU(h264.NALUTypeSEI, 0x05)
		info := filterH264AU([][]byte{sei})
		test.That(t, info.filtered, test.ShouldResemble, [][]byte{sei})
		test.That(t, info.idrPresent, test.ShouldBeFalse)
		test.That(t, info.nonIDRPresent, test.ShouldBeFalse)
	})
}

func TestLogDroppedWarnsOncePerType(t *testing.T) {
	logger, observed := logging.NewObservedTestLogger(t)
	m := &rawSegmenterMux{logger: logger}

	dropped := []droppedNALU{{typ: 51, size: 37}}
	m.logDropped(videostore.CodecTypeH265, dropped)
	m.logDropped(videostore.CodecTypeH265, dropped)
	m.logDropped(videostore.CodecTypeH265, dropped)
	// A different type gets its own warning.
	m.logDropped(videostore.CodecTypeH265, []droppedNALU{{typ: 63, size: 4}})

	warns := observed.FilterLevelExact(zapcore.WarnLevel).All()
	test.That(t, len(warns), test.ShouldEqual, 2)
	test.That(t, warns[0].Message, test.ShouldContainSubstring, "NALU type 51 (37 bytes)")
	test.That(t, warns[1].Message, test.ShouldContainSubstring, "NALU type 63 (4 bytes)")
	test.That(t, len(observed.FilterLevelExact(zapcore.DebugLevel).All()), test.ShouldEqual, 2)

	// Stop resets metadata wholesale, so a reconnect gets a fresh warning. Stop needs a
	// segmenter, so mirror what it does to the field under test.
	m.metadata = metadata{}
	m.logDropped(videostore.CodecTypeH265, dropped)
	test.That(t, len(observed.FilterLevelExact(zapcore.WarnLevel).All()), test.ShouldEqual, 3)
}

func TestWriteH265SkipsRandomAccessUntilParameterSetsSeen(t *testing.T) {
	// The parameter sets are deliberately garbage: filterH265AU only reads the header byte, and
	// maybeReInitVideoStore fails to unmarshal them and returns before touching the segmenter, so
	// no RawSegmenter is needed. What matters is that a nil parameter set never reaches the DTS
	// extractor, which indexes nalu[0] on every entry unguarded.
	vps := h265NALU(h265.NALUType_VPS_NUT, 0xCC)
	sps := h265NALU(h265.NALUType_SPS_NUT, 0xAA)
	pps := h265NALU(h265.NALUType_PPS_NUT, 0xBB)
	idr := h265NALU(h265.NALUType_IDR_W_RADL, 0x01, 0x02)

	for _, tc := range []struct {
		name          string
		vps, sps, pps []byte
	}{
		{"no parameter sets at all", nil, nil, nil},
		{"VPS missing", nil, sps, pps},
		{"SPS missing", vps, nil, pps},
		{"PPS missing", vps, sps, nil},
	} {
		t.Run(tc.name+" skips the AU", func(t *testing.T) {
			logger, observed := logging.NewObservedTestLogger(t)
			m := &rawSegmenterMux{logger: logger}
			m.codec.Store(int64(videostore.CodecTypeH265))
			m.metadata.vps, m.metadata.sps, m.metadata.pps = tc.vps, tc.sps, tc.pps

			err := m.writeH265([][]byte{idr}, 1000)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, m.metadata.dtsExtractor, test.ShouldBeNil)
			test.That(t, observed.FilterMessageSnippet("parameter sets were seen").Len(), test.ShouldEqual, 1)
		})
	}

	t.Run("all parameter sets present proceeds to the DTS extractor", func(t *testing.T) {
		logger, observed := logging.NewObservedTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.codec.Store(int64(videostore.CodecTypeH265))
		m.metadata.vps, m.metadata.sps, m.metadata.pps = vps, sps, pps

		err := m.writeH265([][]byte{idr}, 1000)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, m.metadata.dtsExtractor, test.ShouldNotBeNil)
		test.That(t, observed.FilterMessageSnippet("parameter sets were seen").Len(), test.ShouldEqual, 0)
	})
}

func TestWriteH264SkipsIDRUntilParameterSetsSeen(t *testing.T) {
	sps := h264NALU(h264.NALUTypeSPS, 0xAA)
	pps := h264NALU(h264.NALUTypePPS, 0xBB)
	idr := h264NALU(h264.NALUTypeIDR, 0x01)

	for _, tc := range []struct {
		name     string
		sps, pps []byte
	}{
		{"no parameter sets at all", nil, nil},
		{"SPS missing", nil, pps},
		{"PPS missing", sps, nil},
	} {
		t.Run(tc.name+" skips the AU", func(t *testing.T) {
			logger, observed := logging.NewObservedTestLogger(t)
			m := &rawSegmenterMux{logger: logger}
			m.codec.Store(int64(videostore.CodecTypeH264))
			m.metadata.sps, m.metadata.pps = tc.sps, tc.pps

			err := m.writeH264([][]byte{idr}, 1000)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, m.metadata.dtsExtractor, test.ShouldBeNil)
			test.That(t, observed.FilterMessageSnippet("parameter sets were seen").Len(), test.ShouldEqual, 1)
		})
	}

	t.Run("SPS and PPS present proceeds to the DTS extractor", func(t *testing.T) {
		logger, observed := logging.NewObservedTestLogger(t)
		m := &rawSegmenterMux{logger: logger}
		m.codec.Store(int64(videostore.CodecTypeH264))
		m.metadata.sps, m.metadata.pps = sps, pps

		err := m.writeH264([][]byte{idr}, 1000)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, m.metadata.dtsExtractor, test.ShouldNotBeNil)
		test.That(t, observed.FilterMessageSnippet("parameter sets were seen").Len(), test.ShouldEqual, 0)
	})
}
