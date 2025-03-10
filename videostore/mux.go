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

type rawSegmenterMux struct {
	// are valid for the lifetime of the rawSegmenterMux
	camName resource.Name
	logger  logging.Logger
	worker  *utils.StoppableWorkers
	regDone <-chan struct{}
	cam     registry.ModuleCamera

	mu           sync.Mutex
	rawSeg       *videostore.RawSegmenter
	codec        atomic.Int64
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
}

var codecs = []videostore.CodecType{
	videostore.CodecTypeH265,
	videostore.CodecTypeH264,
}

// init and close are called by videostore.
func (m *rawSegmenterMux) init() error {
	cam, err := registry.Global.Get(m.camName.String())
	if err != nil {
		return err
	}
	regCtx, err := cam.RequestVideo(m, codecs)
	if err != nil {
		return err
	}
	m.regDone = regCtx.Done()
	m.cam = cam
	m.worker.Add(m.monitorRegistration)
	return nil
}

func (m *rawSegmenterMux) monitorRegistration(ctx context.Context) {
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
				} else {
					m.regDone = regCtx.Done()
					m.cam = cam
					registered = true
				}
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
	switch codec {
	case videostore.CodecTypeH264:
		for _, nalu := range au {
			//nolint:mnd
			typ := h264.NALUType(nalu[0] & 0x1F)
			switch typ {
			case h264.NALUTypeSPS:
				m.sps = nalu
			case h264.NALUTypePPS:
				m.pps = nalu
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
				return errors.New("invalid nalu")
			}
		}
	case videostore.CodecTypeH265:
		for _, nalu := range au {
			//nolint:mnd
			typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
			switch typ {
			case h265.NALUType_VPS_NUT:
				m.vps = nalu

			case h265.NALUType_SPS_NUT:
				m.sps = nalu

			case h265.NALUType_PPS_NUT:
				m.pps = nalu
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
				return errors.New("invalid nalu")
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
	m.width = 0
	m.height = 0
	m.vps = nil
	m.sps = nil
	m.pps = nil
	m.dtsExtractor = nil
	m.spsUnChanged = false
	m.firstTimeStampsSet = false
	m.firstPTS = 0
	m.firstDTS = 0

	return nil
}

func (m *rawSegmenterMux) writeH265(au [][]byte, pts int64) error {
	var filteredAU [][]byte

	isRandomAccess := false

	for _, nalu := range au {
		//nolint:mnd
		typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
		switch typ {
		case h265.NALUType_VPS_NUT:
			m.vps = nalu
			continue

		case h265.NALUType_SPS_NUT:
			m.sps = nalu
			m.spsUnChanged = false
			continue

		case h265.NALUType_PPS_NUT:
			m.pps = nalu
			continue

		case h265.NALUType_AUD_NUT:
			continue

		case h265.NALUType_IDR_W_RADL, h265.NALUType_IDR_N_LP, h265.NALUType_CRA_NUT:
			isRandomAccess = true
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
			return errors.New("invalid nalu")
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil {
		return nil
	}

	if err := m.maybeReInitVideoStore(); err != nil {
		return err
	}

	// add VPS, SPS and PPS before random access au
	if isRandomAccess {
		au = append([][]byte{m.vps, m.sps, m.pps}, au...)
	}

	if m.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !isRandomAccess {
			return nil
		}
		m.dtsExtractor = h265.NewDTSExtractor2()
	}

	dts, err := m.dtsExtractor.Extract(au, pts)
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
	if !m.firstTimeStampsSet {
		m.firstPTS = pts
		m.firstDTS = dts
		m.firstTimeStampsSet = true
	}
	err = m.rawSeg.WritePacket(nalu, pts-m.firstPTS, dts-m.firstDTS, isRandomAccess)
	if err != nil {
		m.logger.Errorf("error writing packet to segmenter: %s", err)
	}
	return nil
}

func (m *rawSegmenterMux) writeH264(au [][]byte, pts int64) error {
	var filteredAU [][]byte
	nonIDRPresent := false
	idrPresent := false

	for _, nalu := range au {
		//nolint:mnd
		typ := h264.NALUType(nalu[0] & 0x1F)
		switch typ {
		case h264.NALUTypeSPS:
			m.sps = nalu
			m.spsUnChanged = false
			continue

		case h264.NALUTypePPS:
			m.pps = nalu
			continue

		case h264.NALUTypeAccessUnitDelimiter:
			continue

		case h264.NALUTypeIDR:
			idrPresent = true

		case h264.NALUTypeNonIDR:
			nonIDRPresent = true
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
			return errors.New("invalid nalu")
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil || (!nonIDRPresent && !idrPresent) {
		return nil
	}

	if err := m.maybeReInitVideoStore(); err != nil {
		m.logger.Debugf("unable to init video store: %s", err.Error())
		return nil
	}

	// add SPS and PPS before access unit that contains an IDR
	if idrPresent {
		au = append([][]byte{m.sps, m.pps}, au...)
	}

	if m.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !idrPresent {
			return nil
		}
		m.dtsExtractor = h264.NewDTSExtractor2()
	}

	dts, err := m.dtsExtractor.Extract(au, pts)
	if err != nil {
		m.logger.Debugf("dtsExtractor Extract err: %s", err.Error())
		return nil
	}

	packed, err := h264.AnnexBMarshal(au)
	if err != nil {
		m.logger.Errorf("AnnexBMarshal err: %s", err.Error())
		return err
	}

	if !m.firstTimeStampsSet {
		m.firstPTS = pts
		m.firstDTS = dts
		m.firstTimeStampsSet = true
	}
	err = m.rawSeg.WritePacket(packed, pts-m.firstPTS, dts-m.firstDTS, idrPresent)
	if err != nil {
		m.logger.Errorf("error writing packet to segmenter: %s", err)
	}
	return nil
}

// // maybeReInitVideoStore assumes mu is held by caller.
func (m *rawSegmenterMux) maybeReInitVideoStore() error {
	if m.spsUnChanged {
		return nil
	}
	var width, height int
	codec := videostore.CodecType(m.codec.Load())
	switch codec {
	case videostore.CodecTypeH265:
		var hsps h265.SPS
		if err := hsps.Unmarshal(m.sps); err != nil {
			m.logger.Debugf("unable to init video store: %s", err.Error())
			return nil
		}
		width, height = hsps.Width(), hsps.Height()
	case videostore.CodecTypeH264:
		var hsps h264.SPS
		if err := hsps.Unmarshal(m.sps); err != nil {
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
	if m.rawSeg != nil && m.width == width && m.height == height {
		m.spsUnChanged = true
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

	m.width = width
	m.height = height
	m.spsUnChanged = true
	return nil
}
