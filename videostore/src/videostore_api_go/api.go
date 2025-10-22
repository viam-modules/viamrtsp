package videostore_api

import (
	"context"
	"io"

	"github.com/google/uuid"
	videostorepb "github.com/viam-modules/viamrtsp/videostore/src/videostore_api_go/grpc/src/proto"
	commonpb "go.viam.com/api/common/v1"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/protoutils"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/robot"
	"go.viam.com/utils/rpc"
)

// API is the full API definition.
var API = resource.APINamespace("viam-modules").WithServiceType("videostore")

// Named is a helper for getting the named speech's typed resource name.
func Named(name string) resource.Name {
	return resource.NewName(API, name)
}

// FromRobot is a helper for getting the named VideoStore from the given Robot.
func FromRobot(r robot.Robot, name string) (VideoStore, error) {
	return robot.ResourceFromRobot[VideoStore](r, Named(name))
}

func init() {
	resource.RegisterAPI(API, resource.APIRegistration[VideoStore]{
		RPCServiceServerConstructor: NewRPCServiceServer,
		RPCServiceHandler:           videostorepb.RegisterVideostoreServiceHandlerFromEndpoint,
		RPCServiceDesc:              &videostorepb.VideostoreService_ServiceDesc,
		RPCClient: func(
			ctx context.Context,
			conn rpc.ClientConn,
			remoteName string,
			name resource.Name,
			logger logging.Logger,
		) (VideoStore, error) {
			return NewClientFromConn(conn, remoteName, name, logger), nil
		},
	})
}

type VideoStore interface {
	resource.Resource
	Save(ctx context.Context, from, to, container, metadata string, async bool) (string, error)
	Fetch(ctx context.Context, from, to, container string) ([]byte, error)
	FetchStream(ctx context.Context, from, to, container string, w io.Writer) error
}

type videostoreServer struct {
	videostorepb.UnimplementedVideostoreServiceServer
	coll   resource.APIResourceCollection[VideoStore]
	logger logging.Logger
}

func NewRPCServiceServer(coll resource.APIResourceCollection[VideoStore]) interface{} {
	logger := logging.NewLogger("VideoStoreServer")
	return &videostoreServer{coll: coll, logger: logger}
}

func (s *videostoreServer) Save(ctx context.Context, req *videostorepb.SaveRequest) (*videostorepb.SaveResponse, error) {
	s.logger.Debugf("Received Save request: %+v", req)
	vs, err := s.coll.Resource(req.Name)
	if err != nil {
		return nil, err
	}
	resp, err := vs.Save(ctx, req.From, req.To, req.Container, req.Metadata, req.Async)
	if err != nil {
		return nil, err
	}

	return &videostorepb.SaveResponse{Filename: resp, RequestId: req.RequestId}, nil
}

func (s *videostoreServer) Fetch(ctx context.Context, req *videostorepb.FetchRequest) (*videostorepb.FetchResponse, error) {
	s.logger.Debugf("Received Fetch request: %+v", req)
	vs, err := s.coll.Resource(req.Name)
	if err != nil {
		return nil, err
	}
	resp, err := vs.Fetch(ctx, req.From, req.To, req.Container)
	if err != nil {
		return nil, err
	}
	return &videostorepb.FetchResponse{VideoData: resp, RequestId: req.RequestId}, nil
}

func (s *videostoreServer) FetchStream(req *videostorepb.FetchStreamRequest, stream videostorepb.VideostoreService_FetchStreamServer) error {
	s.logger.Debugf("Received FetchStream request: %+v", req)
	vs, err := s.coll.Resource(req.Name)
	if err != nil {
		return err
	}
	// Stream directly via the interface to avoid buffering.
	return vs.FetchStream(stream.Context(), req.From, req.To, req.Container, streamWriter{stream: stream, requestID: req.RequestId})
}

func (s *videostoreServer) DoCommand(ctx context.Context, req *commonpb.DoCommandRequest) (*commonpb.DoCommandResponse, error) {
	audioIn, err := s.coll.Resource(req.GetName())
	if err != nil {
		return nil, err
	}
	return protoutils.DoFromResourceServer(ctx, audioIn, req)
}

type streamWriter struct {
	stream    videostorepb.VideostoreService_FetchStreamServer
	requestID string
}

func (w streamWriter) Write(p []byte) (int, error) {
	if err := w.stream.Send(&videostorepb.FetchStreamResponse{VideoData: p, RequestId: w.requestID}); err != nil {
		return 0, err
	}
	return len(p), nil
}

func NewClientFromConn(conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) VideoStore {
	svcClient := newSvcClientFromConn(conn, remoteName, name, logger)
	return clientFromSvcClient(svcClient, name.ShortName())
}

func newSvcClientFromConn(conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) *videostoreClient {
	client := videostorepb.NewVideostoreServiceClient(conn)
	return &videostoreClient{
		Named:  name.PrependRemote(remoteName).AsNamed(),
		client: client,
		logger: logger,
	}
}

type videostoreClient struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable
	client videostorepb.VideostoreServiceClient
	logger logging.Logger
}

type namedVideostoreClient struct {
	*videostoreClient
	name string
}

func clientFromSvcClient(sc *videostoreClient, name string) VideoStore {
	return &namedVideostoreClient{sc, name}
}

func (nvc *namedVideostoreClient) Fetch(ctx context.Context, from, to, container string) ([]byte, error) {
	req := &videostorepb.FetchRequest{
		Name:      nvc.name,
		From:      from,
		To:        to,
		Container: container,
		RequestId: uuid.New().String(),
	}
	resp, err := nvc.client.Fetch(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.VideoData, nil
}

func (nvc *namedVideostoreClient) Save(ctx context.Context, from, to, container, metadata string, async bool) (string, error) {
	req := &videostorepb.SaveRequest{
		Name:      nvc.name,
		From:      from,
		To:        to,
		Container: container,
		Metadata:  metadata,
		Async:     async,
		RequestId: uuid.New().String(),
	}
	resp, err := nvc.client.Save(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Filename, nil
}

func (nvc *namedVideostoreClient) FetchStream(ctx context.Context, from, to, container string, w io.Writer) error {
	req := &videostorepb.FetchStreamRequest{
		Name:      nvc.name,
		From:      from,
		To:        to,
		Container: container,
		RequestId: uuid.New().String(),
	}
	st, err := nvc.client.FetchStream(ctx, req)
	if err != nil {
		return err
	}
	for {
		msg, err := st.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, werr := w.Write(msg.GetVideoData()); werr != nil {
			return werr
		}
	}
}
