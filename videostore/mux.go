package videostore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/viam-modules/viamrtsp/registry"
	"github.com/viam-modules/video-store/videostore"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils"
)

const monitorInterval = time.Second * 5

type metadata struct {
	width        int
	height       int
	vps          []byte
	sps          []byte
	pps          []byte
	dtsExtractor interface {
		Extract(au [][]byte, pts int64) (int64, error)
	}
	spsUnChanged       bool
	firstTimeStampsSet bool
	firstPTS           int64
	firstDTS           int64
	lastDTS            int64
	lastDTSSet         bool
	// unknownNALUTypes records unrecognized NALU types already reported at warn level, so each
	// type is only warned about once per mux lifetime (reset in Stop with the rest of metadata).
	unknownNALUTypes map[uint8]struct{}
}
type rawSegmenterMux struct {
	// are valid for the lifetime of the rawSegmenterMux
	camName resource.Name
	logger  logging.Logger
	worker  *utils.StoppableWorkers
	regDone <-chan struct{}
	cam     registry.ModuleCamera

	mu       sync.Mutex
	rawSeg   *videostore.RawSegmenter
	codec    atomic.Int64
	metadata metadata
}

var codecs = []videostore.CodecType{
	videostore.CodecTypeH265,
	videostore.CodecTypeH264,
}

func newRawSegmenterMux(rawSeg *videostore.RawSegmenter, camName resource.Name, logger logging.Logger) *rawSegmenterMux {
	return &rawSegmenterMux{
		rawSeg:  rawSeg,
		camName: camName,
		worker:  utils.NewBackgroundStoppableWorkers(),
		logger:  logger,
	}
}

// init and close are called by videostore.
func (m *rawSegmenterMux) init() error {
	if m.rawSeg == nil {
		return errors.New("videostore.RTPVideoStore.Segmenter() is nil")
	}
	cam, err := registry.Global.Get(m.camName.String())
	if err != nil {
		return fmt.Errorf("resource %s unable to be found in viamrtsp registry err: %s", m.camName.String(), err.Error())
	}
	regCtx, err := cam.RequestVideo(m, codecs)
	if err != nil {
		return err
	}
	m.regDone = regCtx.Done()
	m.cam = cam
	m.worker.Add(m.registrationMonitor)
	return nil
}

func (m *rawSegmenterMux) registrationMonitor(ctx context.Context) {
	registered := true
	timer := time.NewTimer(monitorInterval)
	defer timer.Stop()
	defer m.cleanup()
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-m.regDone:
			registered = false
			// set the channel to nil so we don't go down this case statement next iteration
			m.regDone = nil

		case <-timer.C:
			if !registered {
				cam, err := registry.Global.Get(m.camName.String())
				if err != nil {
					m.logger.Warnf("failed to find camera %s", err.Error())
					timer.Reset(monitorInterval)
					continue
				}

				regCtx, err := cam.RequestVideo(m, codecs)
				if err != nil {
					m.logger.Warnf("failed to register video-store with viamrtsp camera, err: %s", err.Error())
					timer.Reset(monitorInterval)
					continue
				}
				m.regDone = regCtx.Done()
				m.cam = cam
				registered = true
				timer.Reset(monitorInterval)
				continue
			}

			if videostore.CodecType(m.codec.Load()) == videostore.CodecTypeUnknown {
				m.logger.Warn("waiting for viamrtsp camera to send video data to video-store")
			}
			timer.Reset(monitorInterval)
		}
	}
}

func (m *rawSegmenterMux) cleanup() {
	if err := m.Stop(); err != nil {
		m.logger.Warnf("failed to stop raw segmenter %s", err.Error())
	}
	if err := m.cam.CancelRequest(m); err != nil {
		m.logger.Warnf("DeRegister video-store from viamrtsp camera %s", err.Error())
	}
}

func (m *rawSegmenterMux) close() error {
	if m == nil {
		return nil
	}
	m.worker.Stop()
	return nil
}

func (m *rawSegmenterMux) Start(codec videostore.CodecType, au [][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if vsCodec := videostore.CodecType(m.codec.Load()); vsCodec != videostore.CodecTypeUnknown {
		return fmt.Errorf("init called when codec already set to %s", vsCodec)
	}
	// Start only receives the parameter sets advertised in the SDP, so anything that is not a
	// parameter set means the SDP itself is malformed. Unlike writeH264/writeH265, this stays a
	// hard error on purpose: dropping it would leave the segmenter with no SPS and nothing recorded.
	switch codec {
	case videostore.CodecTypeH264:
		for _, nalu := range au {
			if len(nalu) == 0 {
				continue
			}
			//nolint:mnd
			typ := h264.NALUType(nalu[0] & 0x1F)
			switch typ {
			case h264.NALUTypeSPS:
				m.metadata.sps = nalu
			case h264.NALUTypePPS:
				m.metadata.pps = nalu
			case h264.NALUTypeNonIDR,
				h264.NALUTypeDataPartitionA,
				h264.NALUTypeDataPartitionB,
				h264.NALUTypeDataPartitionC,
				h264.NALUTypeIDR,
				h264.NALUTypeSEI,
				h264.NALUTypeAccessUnitDelimiter,
				h264.NALUTypeEndOfSequence,
				h264.NALUTypeEndOfStream,
				h264.NALUTypeFillerData,
				h264.NALUTypeSPSExtension,
				h264.NALUTypePrefix,
				h264.NALUTypeSubsetSPS,
				h264.NALUTypeReserved16,
				h264.NALUTypeReserved17,
				h264.NALUTypeReserved18,
				h264.NALUTypeSliceLayerWithoutPartitioning,
				h264.NALUTypeSliceExtension,
				h264.NALUTypeSliceExtensionDepth,
				h264.NALUTypeReserved22,
				h264.NALUTypeReserved23,
				h264.NALUTypeSTAPA,
				h264.NALUTypeSTAPB,
				h264.NALUTypeMTAP16,
				h264.NALUTypeMTAP24,
				h264.NALUTypeFUA,
				h264.NALUTypeFUB:
				fallthrough
			default:
				return fmt.Errorf("unexpected NALU type %d in SDP parameter sets", typ)
			}
		}
	case videostore.CodecTypeH265:
		for _, nalu := range au {
			if len(nalu) == 0 {
				continue
			}
			//nolint:mnd
			typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
			switch typ {
			case h265.NALUType_VPS_NUT:
				m.metadata.vps = nalu

			case h265.NALUType_SPS_NUT:
				m.metadata.sps = nalu

			case h265.NALUType_PPS_NUT:
				m.metadata.pps = nalu
			case h265.NALUType_TRAIL_N,
				h265.NALUType_TRAIL_R,
				h265.NALUType_TSA_N,
				h265.NALUType_TSA_R,
				h265.NALUType_STSA_N,
				h265.NALUType_STSA_R,
				h265.NALUType_RADL_N,
				h265.NALUType_RADL_R,
				h265.NALUType_RASL_N,
				h265.NALUType_RASL_R,
				h265.NALUType_RSV_VCL_N10,
				h265.NALUType_RSV_VCL_N12,
				h265.NALUType_RSV_VCL_N14,
				h265.NALUType_RSV_VCL_R11,
				h265.NALUType_RSV_VCL_R13,
				h265.NALUType_RSV_VCL_R15,
				h265.NALUType_BLA_W_LP,
				h265.NALUType_BLA_W_RADL,
				h265.NALUType_BLA_N_LP,
				h265.NALUType_IDR_W_RADL,
				h265.NALUType_IDR_N_LP,
				h265.NALUType_CRA_NUT,
				h265.NALUType_RSV_IRAP_VCL22,
				h265.NALUType_RSV_IRAP_VCL23,
				h265.NALUType_AUD_NUT,
				h265.NALUType_EOS_NUT,
				h265.NALUType_EOB_NUT,
				h265.NALUType_FD_NUT,
				h265.NALUType_PREFIX_SEI_NUT,
				h265.NALUType_SUFFIX_SEI_NUT,
				h265.NALUType_AggregationUnit,
				h265.NALUType_FragmentationUnit,
				h265.NALUType_PACI:
				fallthrough
			default:
				return fmt.Errorf("unexpected NALU type %d in SDP parameter sets", typ)
			}
		}
	case videostore.CodecTypeUnknown:
		fallthrough
	default:
		return errors.New("invalid codec")
	}
	m.codec.Store(int64(codec))
	return nil
}

func (m *rawSegmenterMux) WritePacket(codec videostore.CodecType, au [][]byte, pts int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if vsCodec := videostore.CodecType(m.codec.Load()); vsCodec == videostore.CodecTypeUnknown {
		return errors.New("WritePacket called before Init")
	}

	if vsCodec := videostore.CodecType(m.codec.Load()); vsCodec != codec {
		return errors.New("WritePacket called with different codec than Init")
	}

	switch codec {
	case videostore.CodecTypeH264:
		return m.writeH264(au, pts)
	case videostore.CodecTypeH265:
		return m.writeH265(au, pts)
	case videostore.CodecTypeUnknown:
		fallthrough
	default:
		return errors.New("invalid codec")
	}
}

func (m *rawSegmenterMux) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.rawSeg.Close(); err != nil {
		return err
	}
	m.codec.Store(int64(videostore.CodecTypeUnknown))
	m.metadata = metadata{}
	return nil
}

// enforceMonotonicTimestamps ensures DTS is strictly increasing and PTS >= DTS.
// If DTS is non-monotonic, it is replaced with lastDTS + 1.
func (m *rawSegmenterMux) enforceMonotonicTimestamps(pts, dts int64) (int64, int64) {
	// We need to subtract the first pts & dts from the current one so that the
	// first pts & dts are zero. Otherwise the ffmpeg segmenter will create an mp4 file
	// that is a black screen for a few seconds (we believe the amount of time represented by the non zero
	// timestamps).
	normDTS := dts - m.metadata.firstDTS
	normPTS := pts - m.metadata.firstPTS

	if m.metadata.lastDTSSet && normDTS <= m.metadata.lastDTS {
		m.logger.Debugf("non-monotonic DTS detected: %d <= %d, adjusting", normDTS, m.metadata.lastDTS)
		normDTS = m.metadata.lastDTS + 1
	}

	if normPTS < normDTS {
		normPTS = normDTS
	}

	m.metadata.lastDTS = normDTS
	m.metadata.lastDTSSet = true
	return normPTS, normDTS
}

// droppedNALU records a NALU removed from an access unit because its type is not one mediacommon
// defines (reserved or vendor-private ranges), so the caller can log what the camera is emitting.
type droppedNALU struct {
	typ  uint8
	size int
}

// h265AUInfo is the result of filterH265AU.
type h265AUInfo struct {
	// filtered is the access unit with parameter sets, access unit delimiters, empty NALUs and
	// unrecognized NALUs removed. It is nil when nothing remains.
	filtered [][]byte
	// vps, sps and pps are set only when the corresponding parameter set was present in the AU.
	vps []byte
	sps []byte
	pps []byte
	// isRandomAccess is true when the AU contains an IDR or CRA slice.
	isRandomAccess bool
	// dropped lists the unrecognized NALUs that were removed, in order.
	dropped []droppedNALU
}

// filterH265AU classifies every NALU in an H265 access unit. Parameter sets are pulled out so the
// caller can cache them, access unit delimiters are stripped, and NALUs whose type mediacommon does
// not define (reserved VCL 24-31, reserved non-VCL 41-47, unspecified 51-63) are dropped instead of
// failing the whole AU. NVRs commonly carry vendor metadata in the unspecified range, and the slices
// next to it are still perfectly good video (RSDK-14390).
func filterH265AU(au [][]byte) h265AUInfo {
	var info h265AUInfo
	for _, nalu := range au {
		if len(nalu) == 0 {
			continue
		}
		//nolint:mnd
		typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
		switch typ {
		case h265.NALUType_VPS_NUT:
			info.vps = nalu
			continue

		case h265.NALUType_SPS_NUT:
			info.sps = nalu
			continue

		case h265.NALUType_PPS_NUT:
			info.pps = nalu
			continue

		case h265.NALUType_AUD_NUT:
			continue

		case h265.NALUType_IDR_W_RADL, h265.NALUType_IDR_N_LP, h265.NALUType_CRA_NUT:
			info.isRandomAccess = true
		case h265.NALUType_TRAIL_N,
			h265.NALUType_TRAIL_R,
			h265.NALUType_TSA_N,
			h265.NALUType_TSA_R,
			h265.NALUType_STSA_N,
			h265.NALUType_STSA_R,
			h265.NALUType_RADL_N,
			h265.NALUType_RADL_R,
			h265.NALUType_RASL_N,
			h265.NALUType_RASL_R,
			h265.NALUType_RSV_VCL_N10,
			h265.NALUType_RSV_VCL_N12,
			h265.NALUType_RSV_VCL_N14,
			h265.NALUType_RSV_VCL_R11,
			h265.NALUType_RSV_VCL_R13,
			h265.NALUType_RSV_VCL_R15,
			h265.NALUType_BLA_W_LP,
			h265.NALUType_BLA_W_RADL,
			h265.NALUType_BLA_N_LP,
			h265.NALUType_RSV_IRAP_VCL22,
			h265.NALUType_RSV_IRAP_VCL23,
			h265.NALUType_EOS_NUT,
			h265.NALUType_EOB_NUT,
			h265.NALUType_FD_NUT,
			h265.NALUType_PREFIX_SEI_NUT,
			h265.NALUType_SUFFIX_SEI_NUT,
			h265.NALUType_AggregationUnit,
			h265.NALUType_FragmentationUnit,
			h265.NALUType_PACI:
		default:
			info.dropped = append(info.dropped, droppedNALU{typ: uint8(typ), size: len(nalu)})
			continue
		}

		info.filtered = append(info.filtered, nalu)
	}
	return info
}

// h264AUInfo is the result of filterH264AU.
type h264AUInfo struct {
	// filtered is the access unit with parameter sets, access unit delimiters, empty NALUs and
	// unrecognized NALUs removed. It is nil when nothing remains.
	filtered [][]byte
	// sps and pps are set only when the corresponding parameter set was present in the AU.
	sps []byte
	pps []byte
	// idrPresent / nonIDRPresent report which slice types the AU carries.
	idrPresent    bool
	nonIDRPresent bool
	// dropped lists the unrecognized NALUs that were removed, in order.
	dropped []droppedNALU
}

// filterH264AU is the H264 counterpart of filterH265AU. The types mediacommon does not define are
// 0 and 30-31 (unspecified).
func filterH264AU(au [][]byte) h264AUInfo {
	var info h264AUInfo
	for _, nalu := range au {
		if len(nalu) == 0 {
			continue
		}
		//nolint:mnd
		typ := h264.NALUType(nalu[0] & 0x1F)
		switch typ {
		case h264.NALUTypeSPS:
			info.sps = nalu
			continue

		case h264.NALUTypePPS:
			info.pps = nalu
			continue

		case h264.NALUTypeAccessUnitDelimiter:
			continue

		case h264.NALUTypeIDR:
			info.idrPresent = true

		case h264.NALUTypeNonIDR:
			info.nonIDRPresent = true
		case h264.NALUTypeDataPartitionA,
			h264.NALUTypeDataPartitionB,
			h264.NALUTypeDataPartitionC,
			h264.NALUTypeSEI,
			h264.NALUTypeEndOfSequence,
			h264.NALUTypeEndOfStream,
			h264.NALUTypeFillerData,
			h264.NALUTypeSPSExtension,
			h264.NALUTypePrefix,
			h264.NALUTypeSubsetSPS,
			h264.NALUTypeReserved16,
			h264.NALUTypeReserved17,
			h264.NALUTypeReserved18,
			h264.NALUTypeSliceLayerWithoutPartitioning,
			h264.NALUTypeSliceExtension,
			h264.NALUTypeSliceExtensionDepth,
			h264.NALUTypeReserved22,
			h264.NALUTypeReserved23,
			h264.NALUTypeSTAPA,
			h264.NALUTypeSTAPB,
			h264.NALUTypeMTAP16,
			h264.NALUTypeMTAP24,
			h264.NALUTypeFUA,
			h264.NALUTypeFUB:
		default:
			info.dropped = append(info.dropped, droppedNALU{typ: uint8(typ), size: len(nalu)})
			continue
		}

		info.filtered = append(info.filtered, nalu)
	}
	return info
}

// logDropped reports NALUs removed by the AU filters. The first time a given type is seen on this
// mux it is logged at warn so support can tell what a camera is emitting; after that it goes to
// debug, replacing the once-per-keyframe "invalid nalu" error this used to be. Assumes mu is held.
func (m *rawSegmenterMux) logDropped(codec videostore.CodecType, dropped []droppedNALU) {
	if len(dropped) == 0 {
		return
	}
	if m.metadata.unknownNALUTypes == nil {
		m.metadata.unknownNALUTypes = map[uint8]struct{}{}
	}
	for _, d := range dropped {
		if _, seen := m.metadata.unknownNALUTypes[d.typ]; !seen {
			m.metadata.unknownNALUTypes[d.typ] = struct{}{}
			m.logger.Warnf("dropping unrecognized %s NALU type %d (%d bytes) from access unit for camera %s; "+
				"this is usually vendor metadata and the rest of the frame is kept",
				codec, d.typ, d.size, m.camName.ShortName())
			continue
		}
		m.logger.Debugf("dropping unrecognized %s NALU type %d (%d bytes) from access unit", codec, d.typ, d.size)
	}
}

func (m *rawSegmenterMux) writeH265(au [][]byte, pts int64) error {
	info := filterH265AU(au)
	if info.vps != nil {
		m.metadata.vps = info.vps
	}
	if info.sps != nil {
		m.metadata.sps = info.sps
		m.metadata.spsUnChanged = false
	}
	if info.pps != nil {
		m.metadata.pps = info.pps
	}
	m.logDropped(videostore.CodecTypeH265, info.dropped)

	au = info.filtered
	isRandomAccess := info.isRandomAccess

	if au == nil {
		return nil
	}

	if err := m.maybeReInitVideoStore(); err != nil {
		return err
	}

	// add VPS, SPS and PPS before random access au
	if isRandomAccess {
		// The DTS extractor indexes nalu[0] on every entry without a length check, so a parameter
		// set that has not arrived yet (absent from the SDP and not yet sent in-band) must not be
		// prepended as nil. Nothing can be recorded until the camera sends them anyway.
		if m.metadata.vps == nil || m.metadata.sps == nil || m.metadata.pps == nil {
			m.logger.Debug("random access AU before all H265 parameter sets were seen, skipping")
			return nil
		}
		au = append([][]byte{m.metadata.vps, m.metadata.sps, m.metadata.pps}, au...)
	}

	if m.metadata.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !isRandomAccess {
			return nil
		}
		m.metadata.dtsExtractor = h265.NewDTSExtractor2()
	}

	dts, err := m.metadata.dtsExtractor.Extract(au, pts)
	if err != nil {
		m.logger.Errorf("error extracting dts: %s", err)
		return nil
	}

	// h265 uses the same annexb format as h264
	nalu, err := h264.AnnexBMarshal(au)
	if err != nil {
		m.logger.Errorf("failed to marshal annex b: %s", err)
		return err
	}
	if !m.metadata.firstTimeStampsSet {
		m.metadata.firstPTS = pts
		m.metadata.firstDTS = dts
		m.metadata.firstTimeStampsSet = true
	}

	normPTS, normDTS := m.enforceMonotonicTimestamps(pts, dts)
	err = m.rawSeg.WritePacket(nalu, normPTS, normDTS, isRandomAccess)
	if err != nil {
		m.logger.Errorf("error writing packet to segmenter: %s", err)
	}
	return nil
}

func (m *rawSegmenterMux) writeH264(au [][]byte, pts int64) error {
	info := filterH264AU(au)
	if info.sps != nil {
		m.metadata.sps = info.sps
		m.metadata.spsUnChanged = false
	}
	if info.pps != nil {
		m.metadata.pps = info.pps
	}
	m.logDropped(videostore.CodecTypeH264, info.dropped)

	au = info.filtered
	idrPresent, nonIDRPresent := info.idrPresent, info.nonIDRPresent

	if au == nil || (!nonIDRPresent && !idrPresent) {
		return nil
	}

	if err := m.maybeReInitVideoStore(); err != nil {
		m.logger.Debugf("unable to init video store: %s", err.Error())
		return nil
	}

	// add SPS and PPS before access unit that contains an IDR
	if idrPresent {
		// See writeH265: a nil parameter set here would panic in the DTS extractor.
		if m.metadata.sps == nil || m.metadata.pps == nil {
			m.logger.Debug("IDR AU before SPS and PPS parameter sets were seen, skipping")
			return nil
		}
		au = append([][]byte{m.metadata.sps, m.metadata.pps}, au...)
	}

	if m.metadata.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !idrPresent {
			return nil
		}
		m.metadata.dtsExtractor = h264.NewDTSExtractor2()
	}

	dts, err := m.metadata.dtsExtractor.Extract(au, pts)
	if err != nil {
		m.logger.Debugf("dtsExtractor Extract err: %s", err.Error())
		return nil
	}

	packed, err := h264.AnnexBMarshal(au)
	if err != nil {
		m.logger.Errorf("AnnexBMarshal err: %s", err.Error())
		return err
	}

	if !m.metadata.firstTimeStampsSet {
		m.metadata.firstPTS = pts
		m.metadata.firstDTS = dts
		m.metadata.firstTimeStampsSet = true
	}

	normPTS, normDTS := m.enforceMonotonicTimestamps(pts, dts)
	err = m.rawSeg.WritePacket(packed, normPTS, normDTS, idrPresent)
	if err != nil {
		m.logger.Errorf("error writing packet to segmenter: %s", err)
	}
	return nil
}

// // maybeReInitVideoStore assumes mu is held by caller.
func (m *rawSegmenterMux) maybeReInitVideoStore() error {
	if m.metadata.spsUnChanged {
		return nil
	}
	var width, height int
	codec := videostore.CodecType(m.codec.Load())
	switch codec {
	case videostore.CodecTypeH265:
		var hsps h265.SPS
		if err := hsps.Unmarshal(m.metadata.sps); err != nil {
			m.logger.Debugf("unable to init video store: %s", err.Error())
			return nil
		}
		width, height = hsps.Width(), hsps.Height()
	case videostore.CodecTypeH264:
		var hsps h264.SPS
		if err := hsps.Unmarshal(m.metadata.sps); err != nil {
			m.logger.Debugf("unable to init video store: %s", err.Error())
			return nil
		}
		width, height = hsps.Width(), hsps.Height()
	case videostore.CodecTypeUnknown:
		fallthrough
	default:
		return errors.New("invalid videostore.CodecType")
	}

	if width <= 0 || height <= 0 {
		err := errors.New("width and height must both be greater than 0")
		m.logger.Infof("unable to init video store: %s", err.Error())
		return nil
	}
	// if vs is initialized and the height & width have not changed,
	// record the sps as unchanged and return
	if m.metadata.width == width && m.metadata.height == height {
		m.metadata.spsUnChanged = true
		return nil
	}

	// if initialized and the height & width have changed,
	// close and nil out the videostore
	if err := m.rawSeg.Close(); err != nil {
		return err
	}

	if err := m.rawSeg.Init(codec, width, height); err != nil {
		return err
	}

	m.metadata.width = width
	m.metadata.height = height
	m.metadata.spsUnChanged = true
	return nil
}
