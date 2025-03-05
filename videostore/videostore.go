// package videostore stores video
package videostore

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bluenviron/mediacommon/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h265"
	"github.com/viam-modules/viamrtsp/registry"
	"github.com/viam-modules/video-store/videostore"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils"
)

const (
	defaultSegmentSeconds      = 30 // seconds
	defaultUploadPath          = ".viam/capture/video-upload"
	defaultStoragePath         = ".viam/video-storage-viamrtsp"
	maxGRPCSize                = 1024 * 1024 * 32 // bytes
	videoStoreInitCloseTimeout = time.Second * 10
)

var Model = resource.ModelNamespace("viam").WithFamily("viamrtsp").WithModel("video-store")

func init() {
	resource.RegisterComponent(generic.API, Model, resource.Registration[resource.Resource, *Config]{
		Constructor: New,
	})
}

type service struct {
	resource.AlwaysRebuild
	name   resource.Name
	logger logging.Logger

	vs  videostore.VideoStore
	mvs *moduleVideoStore
}

type moduleVideoStore struct {
	ctx    context.Context
	cancel context.CancelFunc
	typ    videostore.SourceType
	logger logging.Logger
}

func (vs *moduleVideoStore) WritePacket(typ videostore.SourceType, au [][]byte, pts int64) error {
	if err := vs.ctx.Err(); err != nil {
		return err
	}
	if vs.typ != typ {
		vs.logger.Errorf("WritePacket called with codec type: %s, expected %s", typ, vs.typ)
		return registry.ErrUnsupported
	}

	// TODO: Nick call into the raw segmenter
	return nil
}

func (vs *moduleVideoStore) Close() {
	vs.cancel()
	// TODO: Nick close raw segmenter
}

func New(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (resource.Resource, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	sc, err := applyStorageDefaults(newConf.Storage, conf.ResourceName().Name)
	if err != nil {
		return nil, err
	}
	if err := sc.Validate(); err != nil {
		return nil, err
	}

	var vs videostore.VideoStore
	// var mvs *moduleVideoStore
	if newConf.Camera != nil {
		c, err := camera.FromDependencies(deps, *newConf.Camera)
		if err != nil {
			return nil, err
		}
		cam, err := registry.Global.Camera(c.Name().String())
		if err != nil {
			return nil, err
		}
		// ctx, cancel := context.WithCancel(context.Background())
		// mvs = &moduleVideoStore{ctx: ctx, cancel: cancel, logger: logger}
		vsc := videostore.Config{
			Type:    videostore.SourceTypeRTP,
			Storage: sc,
		}
		if err := vsc.Validate(); err != nil {
			return nil, err
		}

		rtpVs, err := videostore.NewRTPVideoStore(vsc, logger)
		if err != nil {
			return nil, err
		}
		if err := cam.Register(rtpVs); err != nil {
			return nil, err
		}
		// TODO: Start background goroutine to re-register if the stream breaks
		// TODO: Nick create vs and have it be able to live for the lifetime of the videostore
	} else {
		vsc := videostore.Config{
			Type:    videostore.SourceTypeReadOnly,
			Storage: sc,
		}
		if err := vsc.Validate(); err != nil {
			return nil, err
		}

		vs, err = videostore.NewReadOnlyVideoStore(vsc, logger)
		if err != nil {
			return nil, err
		}
	}
	// vs := &videoStoreMuxer{
	// 	sc:     sc,
	// 	logger: logger,
	// }

	s := &service{
		name:   conf.ResourceName(),
		logger: logger,
		mvs:    mvs,
		vs:     vs,
	}
	return s, nil
}
func (s *service) Name() resource.Name {
	return s.name
}

func (s *service) Close(_ context.Context) error {
	return nil
}
func (s *service) DoCommand(ctx context.Context, command map[string]interface{}) (map[string]interface{}, error) {
	// if rc.videoStoreMuxer == nil {
	// 	return nil, errors.New("not implemented")
	// }
	cmd, ok := command["command"].(string)
	if !ok {
		return nil, errors.New("invalid command type")
	}

	switch cmd {
	// Save command is used to concatenate video clips between the given timestamps.
	// The concatenated video file is then uploaded to the cloud the upload path.
	// The response contains the name of the uploaded file.
	case "save":
		s.logger.Debug("save command received")
		req, err := toSaveCommand(command)
		if err != nil {
			return nil, err
		}

		res, err := s.vs.Save(ctx, req)
		if err != nil {
			return nil, err
		}

		ret := map[string]interface{}{
			"command":  "save",
			"filename": res.Filename,
		}

		if req.Async {
			ret["status"] = "async"
		}
		return ret, nil
	case "fetch":
		s.logger.Debug("fetch command received")
		req, err := toFetchCommand(command)
		if err != nil {
			return nil, err
		}
		res, err := s.vs.Fetch(ctx, req)
		if err != nil {
			return nil, err
		}
		if len(res.Video) > maxGRPCSize {
			return nil, errors.New("video file size exceeds max grpc size")
		}
		// TODO(seanp): Do we need to encode the video bytes to base64?
		videoBytesBase64 := base64.StdEncoding.EncodeToString(res.Video)
		return map[string]interface{}{
			"command": "fetch",
			"video":   videoBytesBase64,
		}, nil
	default:
		return nil, errors.New("invalid command")
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

type Config struct {
	Camera  *string `json:"camera,omitempty"`
	Storage Storage `json:"storage"`
}

type Storage struct {
	SizeGB      int    `json:"size_gb"`
	UploadPath  string `json:"upload_path,omitempty"`
	StoragePath string `json:"storage_path,omitempty"`
}

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
func toSaveCommand(command map[string]interface{}) (*videostore.SaveRequest, error) {
	fromStr, ok := command["from"].(string)
	if !ok {
		return nil, errors.New("from timestamp not found")
	}
	from, err := videostore.ParseDateTimeString(fromStr)
	if err != nil {
		return nil, err
	}
	toStr, ok := command["to"].(string)
	if !ok {
		return nil, errors.New("to timestamp not found")
	}
	to, err := videostore.ParseDateTimeString(toStr)
	if err != nil {
		return nil, err
	}
	metadata, ok := command["metadata"].(string)
	if !ok {
		metadata = ""
	}
	async, ok := command["async"].(bool)
	if !ok {
		async = false
	}
	return &videostore.SaveRequest{
		From:     from,
		To:       to,
		Metadata: metadata,
		Async:    async,
	}, nil
}

func toFetchCommand(command map[string]interface{}) (*videostore.FetchRequest, error) {
	fromStr, ok := command["from"].(string)
	if !ok {
		return nil, errors.New("from timestamp not found")
	}
	from, err := videostore.ParseDateTimeString(fromStr)
	if err != nil {
		return nil, err
	}
	toStr, ok := command["to"].(string)
	if !ok {
		return nil, errors.New("to timestamp not found")
	}
	to, err := videostore.ParseDateTimeString(toStr)
	if err != nil {
		return nil, err
	}
	return &videostore.FetchRequest{From: from, To: to}, nil
}

type extractor interface {
	Extract(au [][]byte, pts int64) (int64, error)
}

type videoStoreMuxer struct {
	sc           videostore.StorageConfig
	width        int
	height       int
	vps          []byte
	sps          []byte
	pps          []byte
	dtsExtractor extractor
	spsUnChanged bool
	mu           sync.Mutex
	vs           videostore.RTPVideoStore
	logger       logging.Logger
}

// // maybeReInitVideoStore assumes mu is held by caller.
func (e *videoStoreMuxer) maybeReInitVideoStore() error {
	if e.spsUnChanged {
		return nil
	}
	var width, height int
	switch e.Config.Type {
	case videostore.SourceTypeH265RTPPacket:
		var hsps h265.SPS
		if err := hsps.Unmarshal(e.sps); err != nil {
			return err
		}
		width, height = hsps.Width(), hsps.Height()
	case videostore.SourceTypeH264RTPPacket:
		var hsps h264.SPS
		if err := hsps.Unmarshal(e.sps); err != nil {
			return err
		}
		width, height = hsps.Width(), hsps.Height()
	case videostore.SourceTypeFrame:
		fallthrough
	case videostore.SourceTypeUnknown:
		fallthrough
	default:
		return errors.New("invalid videostore.SourceType")
	}

	if width <= 0 || height <= 0 {
		return errors.New("width and height must both be greater than 0")
	}
	// if vs is initialized and the height & width have not changed,
	// record the sps as unchanged and return
	if e.vs != nil && e.width == width && e.height == height {
		e.spsUnChanged = true
		return nil
	}

	// if initialized and the height & width have changed,
	// close and nil out the videostore
	if e.vs != nil {
		e.vs.Close()
		e.vs = nil
	}

	// if we don't have a video-store we should attempt to initialize one
	vs, err := videostore.NewRTPVideoStore(e.Config, e.logger)
	if err != nil {
		return err
	}

	if err := vs.Init(width, height); err != nil {
		return err
	}

	e.width = width
	e.height = height
	e.vs = vs
	e.spsUnChanged = true
	return nil
}

func (e *videoStoreMuxer) writeH265(au [][]byte, pts int64) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	var filteredAU [][]byte

	isRandomAccess := false

	for _, nalu := range au {
		//nolint:mnd
		typ := h265.NALUType((nalu[0] >> 1) & 0b111111)
		switch typ {
		case h265.NALUType_VPS_NUT:
			e.vps = nalu
			continue

		case h265.NALUType_SPS_NUT:
			e.sps = nalu
			e.spsUnChanged = false
			continue

		case h265.NALUType_PPS_NUT:
			e.pps = nalu
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
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil {
		return
	}

	if err := e.maybeReInitVideoStore(); err != nil {
		e.logger.Debugf("unable to init video store: %s", err.Error())
		return
	}

	// add VPS, SPS and PPS before random access au
	if isRandomAccess {
		au = append([][]byte{e.vps, e.sps, e.pps}, au...)
	}

	if e.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !isRandomAccess {
			return
		}
		e.dtsExtractor = h265.NewDTSExtractor2()
	}

	dts, err := e.dtsExtractor.Extract(au, pts)
	if err != nil {
		e.logger.Errorf("error extracting dts: %s", err)
		return
	}

	// h265 uses the same annexb format as h264
	nalu, err := h264.AnnexBMarshal(au)
	if err != nil {
		e.logger.Errorf("failed to marshal annex b: %s", err)
		return
	}
	err = e.vs.WritePacket(nalu, pts, dts, isRandomAccess)
	if err != nil {
		e.logger.Errorf("error writing packet to segmenter: %s", err)
	}
}

func (e *videoStoreMuxer) writeH264(au [][]byte, pts int64) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	var filteredAU [][]byte
	nonIDRPresent := false
	idrPresent := false

	for _, nalu := range au {
		//nolint:mnd
		typ := h264.NALUType(nalu[0] & 0x1F)
		switch typ {
		case h264.NALUTypeSPS:
			e.sps = nalu
			e.spsUnChanged = false
			continue

		case h264.NALUTypePPS:
			e.pps = nalu
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
		}

		filteredAU = append(filteredAU, nalu)
	}

	au = filteredAU

	if au == nil || (!nonIDRPresent && !idrPresent) {
		return
	}

	if err := e.maybeReInitVideoStore(); err != nil {
		e.logger.Debugf("unable to init video store: %s", err.Error())
		return
	}

	// add SPS and PPS before access unit that contains an IDR
	if idrPresent {
		au = append([][]byte{e.sps, e.pps}, au...)
	}

	if e.dtsExtractor == nil {
		// skip samples silently until we find one with a IDR
		if !idrPresent {
			return
		}
		e.dtsExtractor = h264.NewDTSExtractor2()
	}

	dts, err := e.dtsExtractor.Extract(au, pts)
	if err != nil {
		return
	}

	packed, err := h264.AnnexBMarshal(au)
	if err != nil {
		e.logger.Errorf("AnnexBMarshal err: %s", err.Error())
		return
	}
	err = e.vs.WritePacket(packed, pts, dts, idrPresent)
	if err != nil {
		e.logger.Errorf("error writing packet to segmenter: %s", err)
	}
}
func (e *videoStoreMuxer) Save(ctx context.Context, r *videostore.SaveRequest) (*videostore.SaveResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vs == nil {
		return nil, errors.New("video-store uninitialized")
	}
	return e.vs.Save(ctx, r)
}

func (e *videoStoreMuxer) Fetch(ctx context.Context, r *videostore.FetchRequest) (*videostore.FetchResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vs == nil {
		return nil, errors.New("video-store uninitialized")
	}
	return e.vs.Fetch(ctx, r)
}

func (e *videoStoreMuxer) CloseVideoStore() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vs == nil {
		return
	}
	e.vs.Close()
}
