package videostore

import (
	"os"
	"path/filepath"

	"github.com/viam-modules/video-store/videostore"
	"go.viam.com/utils"
)

// Config is the config for videostore.
type Config struct {
	Camera  *string `json:"camera,omitempty"`
	Storage Storage `json:"storage"`
}

// Storage is the storage subconfig for videostore.
type Storage struct {
	SizeGB      int    `json:"size_gb"`
	UploadPath  string `json:"upload_path,omitempty"`
	StoragePath string `json:"storage_path,omitempty"`
}

// Validate validates the config and returns the resource graph dependencies.
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.Storage == (Storage{}) {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "storage")
	}
	if cfg.Storage.SizeGB == 0 {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "size_gb")
	}

	sConfig, err := applyStorageDefaults(cfg.Storage, "someprefix")
	if err != nil {
		return nil, err
	}
	if err := sConfig.Validate(); err != nil {
		return nil, err
	}
	// This allows for an implicit camera dependency so we do not need to explicitly
	// add the camera dependency in the config.
	if cfg.Camera != nil {
		return []string{*cfg.Camera}, nil
	}
	return []string{}, nil
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
		home, err := os.UserHomeDir()
		if err != nil {
			return zero, err
		}
		c.StoragePath = filepath.Join(home, defaultStoragePath, name)
	}
	return videostore.StorageConfig{
		SegmentSeconds:       defaultSegmentSeconds,
		SizeGB:               c.SizeGB,
		OutputFileNamePrefix: name,
		UploadPath:           c.UploadPath,
		StoragePath:          c.StoragePath,
	}, nil
}
