package viamrtsp

import (
	"fmt"
	"image"

	"github.com/bluenviron/gortsplib/v3/pkg/formats"
	"github.com/pion/rtp"
)

type decoder func(pkt *rtp.Packet) (image.Image, error)

func h265Decoding() (*formats.H265, decoder, error) {
	var format formats.H265
	rtpDec, err := format.CreateDecoder2()
	if err != nil {
		return nil, nil, err
	}
	
	mjpegDecoder := func(pkt *rtp.Packet) (image.Image, error) {
		au, pts, err := rtpDec.Decode(pkt)
		if err != nil {
			return nil, err
		}
		fmt.Printf("au: %v pts: %v\n", au, pts)
		return nil, nil
		/*
		   		if err != nil {
			if err != rtph265.ErrNonStartingPacketAndNoPrevious && err != rtph265.ErrMorePacketsNeeded {
				log.Printf("ERR: %v", err)
			}
			return
		}

		for _, nalu := range au {
			log.Printf("received NALU with PTS %v and size %d\n", pts, len(nalu))
		}
		*/
	}
	
	return &format, mjpegDecoder, nil
}
