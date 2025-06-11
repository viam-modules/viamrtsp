package videostore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/viam-modules/video-store/videostore"
	rutils "go.viam.com/rdk/utils"
	"go.viam.com/utils"
)

// Video is the config for storage.
type Video struct {
	Bitrate int    `json:"bitrate,omitempty"`
	Preset  string `json:"preset,omitempty"`
}

// Config is the config for videostore.
type Config struct {
	Camera    *string `json:"camera,omitempty"`
	Storage   Storage `json:"storage"`
	Video     Video   `json:"video,omitempty"`
	Framerate int     `json:"framerate,omitempty"`
}

// Storage is the storage subconfig for videostore.
type Storage struct {
	SizeGB      int    `json:"size_gb"`
	UploadPath  string `json:"upload_path,omitempty"`
	StoragePath string `json:"storage_path,omitempty"`
}

func applyDefaults(cfg *Config, name string) (videostore.Config, error) {
	fps := cfg.Framerate
	if fps < 0 {
		return videostore.Config{}, errors.New("framerate can't be negative")
	}
	if fps == 0 {
		fps = defaultFramerate
	}
	sc, err := applyStorageDefaults(cfg.Storage, name)
	if err != nil {
		return videostore.Config{}, err
	}

	ec := applyVideoEncoderDefaults(cfg.Video)
	return videostore.Config{
		Name:    name,
		Storage: sc,
		Encoder: ec,
		FramePoller: videostore.FramePollerConfig{
			Framerate: fps,
		},
	}, nil
}

// Validate validates the config and returns the resource graph dependencies.
func (cfg *Config) Validate(path string) ([]string, []string, error) {
	if cfg.Storage == (Storage{}) {
		return nil, nil, utils.NewConfigValidationFieldRequiredError(path, "storage")
	}
	if cfg.Storage.SizeGB == 0 {
		return nil, nil, utils.NewConfigValidationFieldRequiredError(path, "size_gb")
	}

	if cfg.Framerate < 0 {
		return nil, nil, fmt.Errorf("invalid framerate %d, must be greater than 0", cfg.Framerate)
	}

	sConfig, err := applyStorageDefaults(cfg.Storage, "someprefix")
	if err != nil {
		return nil, nil, err
	}
	if err := sConfig.Validate(); err != nil {
		return nil, nil, err
	}

	if err := applyVideoEncoderDefaults(cfg.Video).Validate(); err != nil {
		return nil, nil, err
	}
	// This allows for an implicit camera dependency so we do not need to explicitly
	// add the camera dependency in the config.
	if cfg.Camera != nil {
		return []string{*cfg.Camera}, nil, nil
	}
	return []string{}, nil, nil
}

func applyVideoEncoderDefaults(c Video) videostore.EncoderConfig {
	if c.Bitrate == 0 {
		c.Bitrate = defaultVideoBitrate
	}
	if c.Preset == "" {
		c.Preset = defaultVideoPreset
	}
	return videostore.EncoderConfig{
		Bitrate: c.Bitrate,
		Preset:  c.Preset,
	}
}

func applyStorageDefaults(c Storage, name string) (videostore.StorageConfig, error) {
	var zero videostore.StorageConfig
	if c.UploadPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return zero, err
		}
		c.UploadPath = filepath.Join(home, defaultUploadPath, name)
	}
	if c.StoragePath == "" {
		home := rutils.PlatformHomeDir()
		c.StoragePath = filepath.Join(home, defaultStoragePath, name)
	}
	return videostore.StorageConfig{
		SizeGB:               c.SizeGB,
		OutputFileNamePrefix: name,
		UploadPath:           c.UploadPath,
		StoragePath:          c.StoragePath,
	}, nil
}
