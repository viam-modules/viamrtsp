# Videostore API

Protobuf API and gRPC bindgings for `videostore` service.

## API

### Save
Prepares and saves a video clip to storage. If `async` is true, the operation is performed asynchronously, and waits for the current segment to finish before saving.

#### Input:
- `from`: Start timestamp in format `YYYY-MM-DD_HH-MM-SS`
- `to`: End timestamp in format `YYYY-MM-DD_HH-MM-SS`
- `container`: Container format for the saved video (e.g., `mp4`, `
fmp4`)
- `async`: Boolean flag to indicate if the save operation should be asynchronous.

#### Output:
- `filename`: The name of the saved video file.

### Fetch
Prepares a clip and returns the video data as bytes.

#### Input:
- `from`: Start timestamp in format `YYYY-MM-DD_HH-MM-SS`
- `to`: End timestamp in format `YYYY-MM-DD_HH-MM-SS`
- `container`: Container format for the fetched video (e.g., `mp4`, `
fmp4`)
#### Output:
- `video_data`: The video data as bytes.

### FetchStream
Prepares a clip and returns the video data as a stream of bytes.

#### Input:
- `from`: Start timestamp in format `YYYY-MM-DD_HH-MM-SS`
- `to`: End timestamp in format `YYYY-MM-DD_HH-MM-SS`
- `container`: Container format for the fetched video (e.g., `mp4`, `
`fmp4`)

#### Output:
- Stream of `video_data` bytes.

### GetStorageState
Prepares a request and returns the storage statistics.

#### Input:
- `name`: The name of the video.

#### Output:
- `storage_used_bytes`: The amount of storage used by the video in bytes.
- `total_duration_ms`: The total duration of the video in milliseconds.
- `video_count`: The number of videos stored.
- `ranges`: The ranges of the video.
- `storage_limit_gb`: The storage limit in gigabytes.
- `device_storage_remaining_gb`: The remaining device storage in gigabytes.
- `storage_path`: The path to the storage location.
