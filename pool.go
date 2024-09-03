package viamrtsp

import (
	"sync"

	"go.viam.com/rdk/logging"
)

type framePool struct {
	frames       []*avFrameWrapper
	maxNumFrames int
	mu           sync.Mutex
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
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.frames) == 0 {
		p.logger.Debug("No frames available in pool. Constructing new frame!")
		frame, err := newAVFrameWrapper()
		if err != nil {
			p.logger.Errorf("Failed to allocate AVFrame in frame pool: %v", err)
			return nil
		}
		frame.refs.Store(1)
		p.newCount++
		return frame
	}

	lastIndex := len(p.frames) - 1
	frame := p.frames[lastIndex]
	p.frames = p.frames[:lastIndex]

	p.logger.Debugf("Item was gotten from the pool. Len now: %d", len(p.frames))
	p.getCount++

	frame.isInPool.Store(false)
	frame.refs.Add(1)
	return frame
}

func (p *framePool) put(frame *avFrameWrapper) {
	p.logger.Debug("Trying to put frame back into pool.")

	if frame.isFreed.Load() {
		p.logger.Warn("Frame was already freed. Cannot put.")
		return
	}
	if frame.isInPool.Load() {
		p.logger.Warn("Frame is already in pool. Cannot put")
		return
	}
	if frame.isBeingServed.Load() {
		p.logger.Debug("Frame is currently being served. Cannot put")
	}

	refs := frame.refs.Add(-1)
	if refs < 0 {
		panic("deref at 0 refs")
	} else if refs > 0 {
		p.logger.Debugf("Frame has %d refs, but must have 0 to be put back into pool.", refs)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.frames) >= p.maxNumFrames {
		// Cleanup logic: free the last item and truncate the slice
		lastFrame := p.frames[len(p.frames)-1]
		lastFrame.free()
		p.frames = p.frames[:len(p.frames)-1]
		p.cleanCount++
	}

	p.frames = append(p.frames, frame)
	p.logger.Debugf("Frame was put in pool. Len now: %d", len(p.frames))
	p.putCount++

	frame.isInPool.Store(true)
}

func (p *framePool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

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
}
