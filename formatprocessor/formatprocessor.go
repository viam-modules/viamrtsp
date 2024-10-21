// Package formatprocessor processes RTP packets into Units when can then be re-encoded
// heavily copied from https://github.com/bluenviron/mediamtx/blob/main/internal/formatprocessor/h264.go
// https://github.com/bluenviron/mediamtx/blob/main/internal/unit/h264.go & related package & the rest of that package
package formatprocessor

import (
	"bytes"
	"errors"
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/pion/rtp"
)

const (
	// headerSizeRTP is the size of the RTP header in bytes.
	headerSizeRTP = 12
	// maxNALUTypeValue is the maximum value of the NALU type.
	maxNALUTypeValue = 0x1F
	// headerNALUByteLength is the length of the NALU header in bytes.
	headerNALUByteLength = 2
)

// Unit is the elementary data unit routed across the server.
type Unit interface {
	// returns RTP packets contained into the unit.
	GetRTPPackets() []*rtp.Packet

	// returns the NTP timestamp of the unit.
	GetNTP() time.Time

	// returns the PTS of the unit.
	GetPTS() time.Duration
}

// H264 is a H264 data unit.
type H264 struct {
	Base
	AU [][]byte
}

// Base contains fields shared across all units.
type Base struct {
	RTPPackets []*rtp.Packet
	NTP        time.Time
	PTS        time.Duration
}

// GetRTPPackets implements Unit.
func (u *Base) GetRTPPackets() []*rtp.Packet {
	return u.RTPPackets
}

// GetNTP implements Unit.
func (u *Base) GetNTP() time.Time {
	return u.NTP
}

// GetPTS implements Unit.
func (u *Base) GetPTS() time.Duration {
	return u.PTS
}

// Processor processes RTP packets & turns them into Units.
type Processor interface {
	// process a Unit.
	ProcessUnit(u Unit) error

	// process a RTP packet and convert it into a unit.
	ProcessRTPPacket(
		pkt *rtp.Packet,
		ntp time.Time,
		pts time.Duration,
		hasNonRTSPReaders bool,
	) (Unit, error)
}

// New returns a new Processor.
func New(
	udpMaxPayloadSize int,
	forma format.Format,
	generateRTPPackets bool,
) (Processor, error) {
	switch forma := forma.(type) {
	case *format.H264:
		return newH264(udpMaxPayloadSize, forma, generateRTPPackets)

	default:
		return nil, errors.New("unsupported formatprocessor")
	}
}

// extract SPS and PPS without decoding RTP packets.
func rtpH264ExtractParams(payload []byte) ([]byte, []byte) {
	if len(payload) < 1 {
		return nil, nil
	}

	typ := h264.NALUType(payload[0] & maxNALUTypeValue)

	switch typ {
	case h264.NALUTypeSPS:
		return payload, nil

	case h264.NALUTypePPS:
		return nil, payload

	case h264.NALUTypeSTAPA:
		payload := payload[1:]
		var sps []byte
		var pps []byte

		for len(payload) > 0 {
			if len(payload) < headerNALUByteLength {
				break
			}

			// Extract the size of the NALU from the first two bytes of the payload.
			// This is done by shifting the first byte 8 bits to the left and combining
			// it with the second byte.
			//nolint:mnd
			size := uint16(payload[0])<<8 | uint16(payload[1])
			payload = payload[2:]

			if size == 0 {
				break
			}

			if int(size) > len(payload) {
				return nil, nil
			}

			nalu := payload[:size]
			payload = payload[size:]

			typ = h264.NALUType(nalu[0] & maxNALUTypeValue)

			switch typ {
			case h264.NALUTypeSPS:
				sps = nalu

			case h264.NALUTypePPS:
				pps = nalu

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
				h264.NALUTypeSTAPB,
				h264.NALUTypeMTAP16,
				h264.NALUTypeMTAP24,
				h264.NALUTypeFUA,
				h264.NALUTypeSTAPA,
				h264.NALUTypeFUB:
			}
		}

		return sps, pps
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
		h264.NALUTypeSTAPB,
		h264.NALUTypeMTAP16,
		h264.NALUTypeMTAP24,
		h264.NALUTypeFUA,
		h264.NALUTypeFUB:
		fallthrough

	default:
		return nil, nil
	}
}

type formatProcessorH264 struct {
	udpMaxPayloadSize int
	format            *format.H264

	encoder *rtph264.Encoder
	decoder *rtph264.Decoder
}

func newH264(
	udpMaxPayloadSize int,
	forma *format.H264,
	generateRTPPackets bool,
) (*formatProcessorH264, error) {
	t := &formatProcessorH264{
		udpMaxPayloadSize: udpMaxPayloadSize,
		format:            forma,
	}

	if generateRTPPackets {
		err := t.createEncoder(nil, nil)
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (t *formatProcessorH264) createEncoder(
	ssrc *uint32,
	initialSequenceNumber *uint16,
) error {
	t.encoder = &rtph264.Encoder{
		PayloadMaxSize:        t.udpMaxPayloadSize - headerSizeRTP,
		PayloadType:           t.format.PayloadTyp,
		SSRC:                  ssrc,
		InitialSequenceNumber: initialSequenceNumber,
		PacketizationMode:     t.format.PacketizationMode,
	}
	return t.encoder.Init()
}

func (t *formatProcessorH264) updateTrackParametersFromRTPPacket(payload []byte) {
	sps, pps := rtpH264ExtractParams(payload)

	if (sps != nil && !bytes.Equal(sps, t.format.SPS)) ||
		(pps != nil && !bytes.Equal(pps, t.format.PPS)) {
		if sps == nil {
			sps = t.format.SPS
		}
		if pps == nil {
			pps = t.format.PPS
		}
		t.format.SafeSetParams(sps, pps)
	}
}

func (t *formatProcessorH264) updateTrackParametersFromAU(au [][]byte) {
	sps := t.format.SPS
	pps := t.format.PPS
	update := false

	for _, nalu := range au {
		typ := h264.NALUType(nalu[0] & maxNALUTypeValue)

		switch typ {
		case h264.NALUTypeSPS:
			if !bytes.Equal(nalu, sps) {
				sps = nalu
				update = true
			}

		case h264.NALUTypePPS:
			if !bytes.Equal(nalu, pps) {
				pps = nalu
				update = true
			}

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
			h264.NALUTypeSTAPB,
			h264.NALUTypeMTAP16,
			h264.NALUTypeMTAP24,
			h264.NALUTypeFUA,
			h264.NALUTypeSTAPA,
			h264.NALUTypeFUB:
		default:
		}
	}

	if update {
		t.format.SafeSetParams(sps, pps)
	}
}

func (t *formatProcessorH264) remuxAccessUnit(au [][]byte) [][]byte {
	isKeyFrame := false
	n := 0

	for _, nalu := range au {
		typ := h264.NALUType(nalu[0] & maxNALUTypeValue)

		switch typ {
		case h264.NALUTypeSPS, h264.NALUTypePPS: // parameters: remove
			continue

		case h264.NALUTypeAccessUnitDelimiter: // AUD: remove
			continue

		case h264.NALUTypeIDR: // key frame
			if !isKeyFrame {
				isKeyFrame = true

				// prepend parameters
				if t.format.SPS != nil && t.format.PPS != nil {
					n += 2
				}
			}
		case h264.NALUTypeNonIDR,
			h264.NALUTypeDataPartitionA,
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
			h264.NALUTypeSTAPB,
			h264.NALUTypeMTAP16,
			h264.NALUTypeMTAP24,
			h264.NALUTypeFUA,
			h264.NALUTypeSTAPA,
			h264.NALUTypeFUB:
		default:
		}
		n++
	}

	if n == 0 {
		return nil
	}

	filteredNALUs := make([][]byte, n)
	i := 0

	if isKeyFrame && t.format.SPS != nil && t.format.PPS != nil {
		filteredNALUs[0] = t.format.SPS
		filteredNALUs[1] = t.format.PPS
		i = 2
	}

	for _, nalu := range au {
		typ := h264.NALUType(nalu[0] & maxNALUTypeValue)

		switch typ {
		case h264.NALUTypeSPS, h264.NALUTypePPS:
			continue

		case h264.NALUTypeAccessUnitDelimiter:
			continue
		case h264.NALUTypeNonIDR,
			h264.NALUTypeDataPartitionA,
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
			h264.NALUTypeSTAPB,
			h264.NALUTypeMTAP16,
			h264.NALUTypeMTAP24,
			h264.NALUTypeFUA,
			h264.NALUTypeSTAPA,
			h264.NALUTypeIDR,
			h264.NALUTypeFUB:
		default:
		}

		filteredNALUs[i] = nalu
		i++
	}

	return filteredNALUs
}

func (t *formatProcessorH264) ProcessUnit(uu Unit) error {
	u := uu.(*H264)

	t.updateTrackParametersFromAU(u.AU)
	u.AU = t.remuxAccessUnit(u.AU)

	if u.AU != nil {
		pkts, err := t.encoder.Encode(u.AU)
		if err != nil {
			return err
		}
		u.RTPPackets = pkts

		// wraparound is expected for rtp timestamps
		//nolint:gosec
		ts := uint32(multiplyAndDivide(u.PTS, time.Duration(t.format.ClockRate()), time.Second))
		for _, pkt := range u.RTPPackets {
			pkt.Timestamp += ts
		}
	}

	return nil
}

func (t *formatProcessorH264) ProcessRTPPacket(
	pkt *rtp.Packet,
	ntp time.Time,
	pts time.Duration,
	hasNonRTSPReaders bool,
) (Unit, error) {
	u := &H264{
		Base: Base{
			RTPPackets: []*rtp.Packet{pkt},
			NTP:        ntp,
			PTS:        pts,
		},
	}

	t.updateTrackParametersFromRTPPacket(pkt.Payload)

	if t.encoder == nil {
		// remove padding
		pkt.Header.Padding = false
		pkt.PaddingSize = 0

		// RTP packets exceed maximum size: start re-encoding them
		if pkt.MarshalSize() > t.udpMaxPayloadSize {
			v1 := pkt.SSRC
			v2 := pkt.SequenceNumber
			err := t.createEncoder(&v1, &v2)
			if err != nil {
				return nil, err
			}
		}
	}

	// decode from RTP
	if hasNonRTSPReaders || t.decoder != nil || t.encoder != nil {
		if t.decoder == nil {
			var err error
			t.decoder, err = t.format.CreateDecoder()
			if err != nil {
				return nil, err
			}
		}

		au, err := t.decoder.Decode(pkt)

		if t.encoder != nil {
			u.RTPPackets = nil
		}

		if err != nil {
			if errors.Is(err, rtph264.ErrNonStartingPacketAndNoPrevious) ||
				errors.Is(err, rtph264.ErrMorePacketsNeeded) {
				return u, nil
			}
			return nil, err
		}

		u.AU = t.remuxAccessUnit(au)
	}

	// route packet as is
	if t.encoder == nil {
		return u, nil
	}

	// encode into RTP
	if len(u.AU) != 0 {
		pkts, err := t.encoder.Encode(u.AU)
		if err != nil {
			return nil, err
		}
		u.RTPPackets = pkts

		for _, newPKT := range u.RTPPackets {
			newPKT.Timestamp = pkt.Timestamp
		}
	}

	return u, nil
}

// avoid an int64 overflow and preserve resolution by splitting division into two parts:
// first add the integer part, then the decimal part.
func multiplyAndDivide(v, m, d time.Duration) time.Duration {
	secs := v / d
	dec := v % d
	//nolint:durationcheck
	return (secs*m + dec*m/d)
}
