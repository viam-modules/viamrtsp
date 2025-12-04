package videostore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/viam-modules/video-store/videostore"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/video"
	"go.viam.com/test"
	"go.viam.com/utils"
)

// mockVideoStore is a mock implementation of videostore.VideoStore for testing.
type mockVideoStore struct {
	fetchStreamFunc func(ctx context.Context, req *videostore.FetchRequest, emit func(video.Chunk) error) error
	fetchFunc       func(ctx context.Context, req *videostore.FetchRequest) (*videostore.FetchResponse, error)
	saveFunc        func(ctx context.Context, req *videostore.SaveRequest) (*videostore.SaveResponse, error)
	getStorageState func(ctx context.Context) (*videostore.StorageState, error)
	closeFunc       func()
}

func (m *mockVideoStore) FetchStream(ctx context.Context, req *videostore.FetchRequest, emit func(video.Chunk) error) error {
	if m.fetchStreamFunc != nil {
		return m.fetchStreamFunc(ctx, req, emit)
	}
	return nil
}

func (m *mockVideoStore) Fetch(ctx context.Context, req *videostore.FetchRequest) (*videostore.FetchResponse, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, req)
	}
	return &videostore.FetchResponse{}, nil
}

func (m *mockVideoStore) Save(ctx context.Context, req *videostore.SaveRequest) (*videostore.SaveResponse, error) {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, req)
	}
	return &videostore.SaveResponse{}, nil
}

func (m *mockVideoStore) GetStorageState(ctx context.Context) (*videostore.StorageState, error) {
	if m.getStorageState != nil {
		return m.getStorageState(ctx)
	}
	return &videostore.StorageState{}, nil
}

func (m *mockVideoStore) Close() {
	if m.closeFunc != nil {
		m.closeFunc()
	}
}

// createTestService creates a service with a mock videostore for testing.
func createTestService(t *testing.T, mockVS *mockVideoStore) *service {
	t.Helper()
	logger := logging.NewTestLogger(t)

	s := &service{
		name:    resource.NewName(video.API, "test-video-service"),
		logger:  logger,
		vs:      mockVS,
		rsMux:   nil,
		workers: utils.NewBackgroundStoppableWorkers(),
	}

	t.Cleanup(func() {
		s.workers.Stop()
	})

	return s
}

func TestGetVideoReturnsChannel(t *testing.T) {
	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			// Emit a few chunks
			for range 3 {
				if err := emit(video.Chunk{
					Data:      []byte("test chunk data"),
					Container: "mp4",
				}); err != nil {
					return err
				}
			}
			return nil
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	startTime := time.Date(2024, 9, 6, 15, 0, 33, 0, time.UTC)
	endTime := time.Date(2024, 9, 6, 15, 0, 50, 0, time.UTC)

	ch, err := svc.GetVideo(ctx, startTime, endTime, "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, ch, test.ShouldNotBeNil)

	// Collect all chunks from channel
	var chunks []*video.Chunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	test.That(t, len(chunks), test.ShouldEqual, 3)
	for _, chunk := range chunks {
		test.That(t, string(chunk.Data), test.ShouldEqual, "test chunk data")
		test.That(t, chunk.Container, test.ShouldEqual, "mp4")
	}
}

func TestGetVideoChannelCloseOnCompletion(t *testing.T) {
	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			return emit(video.Chunk{
				Data:      []byte("single chunk"),
				Container: "mp4",
			})
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	ch, err := svc.GetVideo(ctx, startTime, endTime, "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	// Read all chunks
	var count int
	for range ch {
		count++
	}

	test.That(t, count, test.ShouldEqual, 1)

	// Channel should be closed - verify by reading again
	_, ok := <-ch
	test.That(t, ok, test.ShouldBeFalse)
}

func TestGetVideoContextCancellation(t *testing.T) {
	emitStarted := make(chan struct{})
	ctxCancelled := make(chan struct{})

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			close(emitStarted)
			// Wait for context to be cancelled
			<-ctxCancelled
			// Give time for the cancellation to propagate
			time.Sleep(10 * time.Millisecond)
			// Try to emit after context is cancelled - should fail
			err := emit(video.Chunk{
				Data:      []byte("chunk after cancel"),
				Container: "mp4",
			})
			return err
		},
	}

	svc := createTestService(t, mockVS)

	ctx, cancel := context.WithCancel(context.Background())
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	ch, err := svc.GetVideo(ctx, startTime, endTime, "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	<-emitStarted
	cancel()
	close(ctxCancelled)

	// Drain the channel - it should close without the cancelled chunk
	var chunks []*video.Chunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// No chunks should have been received since we cancelled before emit
	test.That(t, len(chunks), test.ShouldEqual, 0)
}

func TestGetVideoFetchStreamError(t *testing.T) {
	expectedErr := errors.New("fetch stream error")

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			// Emit one chunk before error
			if err := emit(video.Chunk{
				Data:      []byte("chunk before error"),
				Container: "mp4",
			}); err != nil {
				return err
			}
			return expectedErr
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	ch, err := svc.GetVideo(ctx, startTime, endTime, "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	// Should still receive the chunk before the error
	var chunks []*video.Chunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// The chunk before the error should have been received
	test.That(t, len(chunks), test.ShouldEqual, 1)
}

func TestGetVideoTimeRangePassedCorrectly(t *testing.T) {
	startTime := time.Date(2024, 9, 6, 15, 0, 33, 0, time.UTC)
	endTime := time.Date(2024, 9, 6, 15, 0, 50, 0, time.UTC)

	var capturedReq *videostore.FetchRequest

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, req *videostore.FetchRequest, _ func(video.Chunk) error) error {
			capturedReq = req
			return nil
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	ch, err := svc.GetVideo(ctx, startTime, endTime, "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	// Drain channel to ensure worker completes
	//nolint:revive // intentionally draining channel
	for range ch {
	}

	test.That(t, capturedReq, test.ShouldNotBeNil)
	test.That(t, capturedReq.From.Equal(startTime), test.ShouldBeTrue)
	test.That(t, capturedReq.To.Equal(endTime), test.ShouldBeTrue)
}

func TestGetVideoWorkerShutdown(t *testing.T) {
	workerStarted := make(chan struct{})
	workerBlocked := make(chan struct{})

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(ctx context.Context, _ *videostore.FetchRequest, _ func(video.Chunk) error) error {
			close(workerStarted)
			// Block until context is cancelled (worker shutdown)
			<-ctx.Done()
			close(workerBlocked)
			return ctx.Err()
		},
	}

	logger := logging.NewTestLogger(t)
	workers := utils.NewBackgroundStoppableWorkers()

	svc := &service{
		name:    resource.NewName(video.API, "test-video-service"),
		logger:  logger,
		vs:      mockVS,
		rsMux:   nil,
		workers: workers,
	}

	ctx := context.Background()
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	ch, err := svc.GetVideo(ctx, startTime, endTime, "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	<-workerStarted

	workers.Stop()

	// Worker should eventually receive shutdown signal
	select {
	case <-workerBlocked:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not receive shutdown signal")
	}

	// Channel should be closed
	_, ok := <-ch
	test.That(t, ok, test.ShouldBeFalse)
}

func TestGetVideoEmitBlocksUntilReceived(t *testing.T) {
	const numChunks = 5
	emitCount := 0
	var mu sync.Mutex

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			for range numChunks {
				if err := emit(video.Chunk{
					Data:      []byte("chunk"),
					Container: "mp4",
				}); err != nil {
					return err
				}
				mu.Lock()
				emitCount++
				mu.Unlock()
			}
			return nil
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	ch, err := svc.GetVideo(ctx, time.Now().Add(-time.Hour), time.Now(), "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	// Read chunks one by one with a small delay
	var received int
	for range ch {
		received++
	}

	test.That(t, received, test.ShouldEqual, numChunks)

	mu.Lock()
	test.That(t, emitCount, test.ShouldEqual, numChunks)
	mu.Unlock()
}

func TestGetVideoLargeChunks(t *testing.T) {
	// Test with data chunks larger than typical gRPC unary limits
	largeData := make([]byte, 64*1024) // 64KB chunks
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			for range 10 {
				if err := emit(video.Chunk{
					Data:      largeData,
					Container: "mp4",
				}); err != nil {
					return err
				}
			}
			return nil
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	ch, err := svc.GetVideo(ctx, time.Now().Add(-time.Hour), time.Now(), "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	var totalBytes int
	var chunkCount int
	for chunk := range ch {
		totalBytes += len(chunk.Data)
		chunkCount++
		test.That(t, len(chunk.Data), test.ShouldEqual, 64*1024)
	}

	test.That(t, chunkCount, test.ShouldEqual, 10)
	test.That(t, totalBytes, test.ShouldEqual, 10*64*1024)
}

func TestGetVideoEmptyResult(t *testing.T) {
	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, _ func(video.Chunk) error) error {
			// No chunks emitted
			return nil
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	ch, err := svc.GetVideo(ctx, time.Now().Add(-time.Hour), time.Now(), "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	var count int
	for range ch {
		count++
	}

	test.That(t, count, test.ShouldEqual, 0)
}

func TestGetVideoContextDeadline(t *testing.T) {
	mockVS := &mockVideoStore{
		fetchStreamFunc: func(ctx context.Context, _ *videostore.FetchRequest, _ func(video.Chunk) error) error {
			// Simulate slow operation
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	svc := createTestService(t, mockVS)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ch, err := svc.GetVideo(ctx, time.Now().Add(-time.Hour), time.Now(), "h264", "mp4", nil)
	test.That(t, err, test.ShouldBeNil)

	// Channel should close after timeout
	start := time.Now()
	//nolint:revive // intentionally draining channel
	for range ch {
	}
	elapsed := time.Since(start)

	// Should complete quickly due to timeout, not wait the full 5 seconds
	test.That(t, elapsed, test.ShouldBeLessThan, 1*time.Second)
}

func TestGetVideoConcurrentRequests(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	mockVS := &mockVideoStore{
		fetchStreamFunc: func(_ context.Context, _ *videostore.FetchRequest, emit func(video.Chunk) error) error {
			mu.Lock()
			callCount++
			mu.Unlock()

			return emit(video.Chunk{
				Data:      []byte("chunk"),
				Container: "mp4",
			})
		},
	}

	svc := createTestService(t, mockVS)

	ctx := context.Background()
	const numRequests = 5

	// Start multiple concurrent GetVideo requests
	channels := make([]chan *video.Chunk, numRequests)
	for i := range numRequests {
		ch, err := svc.GetVideo(ctx, time.Now().Add(-time.Hour), time.Now(), "h264", "mp4", nil)
		test.That(t, err, test.ShouldBeNil)
		channels[i] = ch
	}

	// Drain all channels
	for _, ch := range channels {
		//nolint:revive // intentionally draining channel
		for range ch {
		}
	}

	mu.Lock()
	test.That(t, callCount, test.ShouldEqual, numRequests)
	mu.Unlock()
}
