package registry

import (
	"errors"
	"sync"

	"github.com/viam-modules/video-store/videostore"
)

type Mux interface {
	Start(codec videostore.CodecType, au [][]byte) error
	WritePacket(codec videostore.CodecType, au [][]byte, pts int64) error
	Stop() error
}

type ModuleCamera interface {
	Register(vs Mux, codecCandiates []videostore.CodecType) error
	DeRegister(vs Mux) error
}

// type ModuleVideoStores interface {
// 	WritePacket(typ videostore.SourceType, au [][]byte, pts int64) error
// 	Close()
// }

type ModuleRegistry struct {
	mu   sync.Mutex
	cams map[string]ModuleCamera
	// videoStores map[string]ModuleVideoStores
}

var (
	ErrAlreadyInRegistry = errors.New("already in registry")
	ErrNotFound          = errors.New("not in registry")
	ErrUnsupported       = errors.New("unsupported")
	ErrBusy              = errors.New("busy")
	Global               = &ModuleRegistry{cams: map[string]ModuleCamera{}}
)

func (mr *ModuleRegistry) AddCamera(key string, val ModuleCamera) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if _, ok := mr.cams[key]; ok {
		return ErrAlreadyInRegistry
	}
	mr.cams[key] = val
	return nil
}

// func (mr *ModuleRegistry) AddVideoStore(key string, val ModuleVideoStores) error {
// 	mr.mu.Lock()
// 	defer mr.mu.Unlock()
// 	if _, ok := mr.videoStores[key]; ok {
// 		return ErrAlreadyInRegistry
// 	}
// 	mr.videoStores[key] = val
// 	return nil
// }

func (mr *ModuleRegistry) RemoveCamera(key string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if _, ok := mr.cams[key]; !ok {
		return ErrNotFound
	}
	delete(mr.cams, key)
	return nil
}

// func (mr *ModuleRegistry) RemoveVideoStore(key string) error {
// 	mr.mu.Lock()
// 	defer mr.mu.Unlock()
// 	if _, ok := mr.videoStores[key]; !ok {
// 		return ErrNotFound
// 	}
// 	delete(mr.videoStores, key)
// 	return nil
// }

func (mr *ModuleRegistry) Camera(key string) (ModuleCamera, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	cam, ok := mr.cams[key]
	if !ok {
		return nil, ErrNotFound
	}
	return cam, nil
}

// func (mr *ModuleRegistry) VideoStore(key string) (ModuleVideoStores, error) {
// 	mr.mu.Lock()
// 	defer mr.mu.Unlock()
// 	vs, ok := mr.videoStores[key]
// 	if !ok {
// 		return nil, ErrNotFound
// 	}
// 	return vs, nil
// }
