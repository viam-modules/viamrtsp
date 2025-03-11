package viamrtsp

import (
	"context"
	"sync"

	"github.com/viam-modules/viamrtsp/registry"
	"github.com/viam-modules/video-store/videostore"
	"go.viam.com/rdk/logging"
)

type videoRequest struct {
	logger logging.Logger

	mu      sync.Mutex
	mux     registry.Mux
	started bool
	cancel  context.CancelFunc
}

func (vr *videoRequest) active() bool {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	return vr.mux != nil
}

func (vr *videoRequest) newRequest(mux registry.Mux) (context.Context, error) {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	if vr.mux != nil {
		return nil, registry.ErrBusy
	}
	ctx, cancel := context.WithCancel(context.Background())
	vr.mux = mux
	vr.started = false
	vr.cancel = cancel
	return ctx, nil
}

func (vr *videoRequest) cancelRequest(mux registry.Mux) error {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	if vr.mux == nil {
		return nil
	}
	if vr.mux != mux {
		return registry.ErrNotFound
	}
	vr.cancel()
	vr.mux = nil
	vr.started = false
	vr.cancel = nil
	return nil
}

func (vr *videoRequest) write(codec videostore.CodecType, initialParameters [][]byte, au [][]byte, pts int64) {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	if vr.mux == nil {
		return
	}

	if !vr.started {
		if err := vr.mux.Start(codec, initialParameters); err != nil {
			vr.logger.Errorf("codec: %s, failed to start Mux: %s", codec, err.Error())
			return
		}
		vr.started = true
	}
	if err := vr.mux.WritePacket(codec, au, pts); err != nil {
		vr.logger.Errorf("codec: %s, videostore WritePacket returned error, err: %s", codec, err.Error())
	}
}

func (vr *videoRequest) stop() {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	if vr.mux != nil {
		if err := vr.mux.Stop(); err != nil {
			vr.logger.Errorf("error stopping mux: %s", err.Error())
		}
	}
	vr.started = false
}

func (vr *videoRequest) clear() {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	if vr.mux == nil {
		return
	}

	if err := vr.mux.Stop(); err != nil {
		vr.logger.Errorf("error stopping mux: %s", err.Error())
	}

	if vr.cancel != nil {
		vr.cancel()
	}
	vr.mux = nil
	vr.started = false
	vr.cancel = nil
}
