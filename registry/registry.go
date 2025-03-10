// Package registry provides a means for videostores to find viamrtsp cameras within the same OS process
package registry

import (
	"context"
	"errors"
	"sync"

	"github.com/viam-modules/video-store/videostore"
)

// Mux is how viamrtsp writes it's video data to a video-store that has requested it.
type Mux interface {
	// Start starts saving an rtsp stream's video
	Start(codec videostore.CodecType, initialParameters [][]byte) error
	// WritePacket writes a packet
	WritePacket(codec videostore.CodecType, au [][]byte, pts int64) error
	// Stop stops the mux so any resources taken during Start can be released
	Stop() error
}

// ModuleCamera allows videostore to request video from a camera and cancel that request.
type ModuleCamera interface {
	// RequestVideo requests the camera write video to the mux if the camera supports one of the codecs
	RequestVideo(vs Mux, codecCandiates []videostore.CodecType) (context.Context, error)
	// CancelRequest cancels a request
	CancelRequest(vs Mux) error
}

// ModuleRegistry allows ModuleCamera s to be added, removed and queried from the global registry.
type ModuleRegistry struct {
	mu   sync.Mutex
	cams map[string]ModuleCamera
}

var (
	// ErrAlreadyInRegistry means that the resource is already in the registry.
	ErrAlreadyInRegistry = errors.New("already in registry")
	// ErrNotFound means that the resource was not found.
	ErrNotFound = errors.New("not in registry")
	// ErrUnsupported means that the codec is unsupported.
	ErrUnsupported = errors.New("unsupported codec")
	// ErrBusy means that the resource is busy.
	ErrBusy = errors.New("busy")
	// Global is the global registry.
	Global = &ModuleRegistry{cams: map[string]ModuleCamera{}}
)

// Get returns the ModuleCamera with the name from the global registry.
func (mr *ModuleRegistry) Get(name string) (ModuleCamera, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	cam, ok := mr.cams[name]
	if !ok {
		return nil, ErrNotFound
	}
	return cam, nil
}

// Add adds a ModuleCamera with a name to the global registry.
func (mr *ModuleRegistry) Add(name string, val ModuleCamera) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if _, ok := mr.cams[name]; ok {
		return ErrAlreadyInRegistry
	}
	mr.cams[name] = val
	return nil
}

// Remove removes a ModuleCamera with a name from the global registry.
func (mr *ModuleRegistry) Remove(name string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if _, ok := mr.cams[name]; !ok {
		return ErrNotFound
	}
	delete(mr.cams, name)
	return nil
}
