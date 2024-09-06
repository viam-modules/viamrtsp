package viamrtsp

import (
	"sync"

	"go.viam.com/rdk/logging"
)

type framePool struct {
	frames       []*avFrameWrapper
	maxNumFrames int
	framesMu     sync.Mutex
	logger       logging.Logger

	putCount   int
	getCount   int
	cleanCount int
	newCount   int
}

func newFramePool(maxNumFrames int, logger logging.Logger) *framePool {
	pool := &framePool{
		frames:       make([]*avFrameWrapper, 0, maxNumFrames),
		maxNumFrames: maxNumFrames,
		logger:       logger,
	}

	return pool
}

func (p *framePool) get() *avFrameWrapper {
	p.framesMu.Lock()
	defer p.framesMu.Unlock()

	if len(p.frames) == 0 {
		p.logger.Debug("No frames available in pool. Constructing new frame!")
		frame, err := newAVFrameWrapper()
		if err != nil {
			p.logger.Errorf("Failed to allocate AVFrame in frame pool: %v", err)
			return nil
		}
		p.newCount++
		return frame
	}

	lastIndex := len(p.frames) - 1
	frame := p.frames[lastIndex]
	p.frames = p.frames[:lastIndex]

	p.logger.Debugf("Item was gotten from the pool. Len now: %d", len(p.frames))
	p.getCount++

	frame.isInPool.Store(false)
	return frame
}

func (p *framePool) put(frame *avFrameWrapper) {
	p.logger.Debug("Trying to put frame back into pool.")

	if frame.isFreed.Load() {
		p.logger.Error("Frame was already freed. Cannot put.")
		return
	}
	if frame.isInPool.Load() {
		p.logger.Error("Frame is already in pool. Cannot put")
		return
	}

	p.framesMu.Lock()
	defer p.framesMu.Unlock()

	if len(p.frames) >= p.maxNumFrames {
		frame.free()
		p.cleanCount++
		return
	}

	p.frames = append(p.frames, frame)
	p.logger.Debugf("Frame was put in pool. Len now: %d", len(p.frames))
	p.putCount++

	frame.isInPool.Store(true)
}

func (p *framePool) close() {
	p.framesMu.Lock()
	defer p.framesMu.Unlock()

	// Free remaining pool frames
	p.logger.Debugf("Num pool frames remaining at close: %d", len(p.frames))
	for _, frame := range p.frames {
		frame.free()
	}

	// Clear the slice to release references
	p.frames = nil

	// Report stats
	p.logger.Debugf("getCount: %d", p.getCount)
	p.logger.Debugf("putCount: %d", p.putCount)
	p.logger.Debugf("newCount: %d", p.newCount)
	p.logger.Debugf("cleanCount: %d", p.cleanCount)
}
