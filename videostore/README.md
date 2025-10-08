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
TODO

### FetchStream
TODO

### GetStorageStats
TODO
