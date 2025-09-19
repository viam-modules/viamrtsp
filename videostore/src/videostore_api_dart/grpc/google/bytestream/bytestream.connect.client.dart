//
//  Generated code. Do not modify.
//  source: google/bytestream/bytestream.proto
//

import "package:connectrpc/connect.dart" as connect;
import "bytestream.pb.dart" as googlebytestreambytestream;
import "bytestream.connect.spec.dart" as specs;

/// #### Introduction
/// The Byte Stream API enables a client to read and write a stream of bytes to
/// and from a resource. Resources have names, and these names are supplied in
/// the API calls below to identify the resource that is being read from or
/// written to.
/// All implementations of the Byte Stream API export the interface defined here:
/// * `Read()`: Reads the contents of a resource.
/// * `Write()`: Writes the contents of a resource. The client can call `Write()`
///   multiple times with the same resource and can check the status of the write
///   by calling `QueryWriteStatus()`.
/// #### Service parameters and metadata
/// The ByteStream API provides no direct way to access/modify any metadata
/// associated with the resource.
/// #### Errors
/// The errors returned by the service are in the Google canonical error space.
extension type ByteStreamClient (connect.Transport _transport) {
  /// `Read()` is used to retrieve the contents of a resource as a sequence
  /// of bytes. The bytes are returned in a sequence of responses, and the
  /// responses are delivered as the results of a server-side streaming RPC.
  Stream<googlebytestreambytestream.ReadResponse> read(
    googlebytestreambytestream.ReadRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).server(
      specs.ByteStream.read,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  /// `Write()` is used to send the contents of a resource as a sequence of
  /// bytes. The bytes are sent in a sequence of request protos of a client-side
  /// streaming RPC.
  /// A `Write()` action is resumable. If there is an error or the connection is
  /// broken during the `Write()`, the client should check the status of the
  /// `Write()` by calling `QueryWriteStatus()` and continue writing from the
  /// returned `committed_size`. This may be less than the amount of data the
  /// client previously sent.
  /// Calling `Write()` on a resource name that was previously written and
  /// finalized could cause an error, depending on whether the underlying service
  /// allows over-writing of previously written resources.
  /// When the client closes the request channel, the service will respond with
  /// a `WriteResponse`. The service will not view the resource as `complete`
  /// until the client has sent a `WriteRequest` with `finish_write` set to
  /// `true`. Sending any requests on a stream after sending a request with
  /// `finish_write` set to `true` will cause an error. The client **should**
  /// check the `WriteResponse` it receives to determine how much data the
  /// service was able to commit and whether the service views the resource as
  /// `complete` or not.
  Future<googlebytestreambytestream.WriteResponse> write(
    Stream<googlebytestreambytestream.WriteRequest> input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).client(
      specs.ByteStream.write,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  /// `QueryWriteStatus()` is used to find the `committed_size` for a resource
  /// that is being written, which can then be used as the `write_offset` for
  /// the next `Write()` call.
  /// If the resource does not exist (i.e., the resource has been deleted, or the
  /// first `Write()` has not yet reached the service), this method returns the
  /// error `NOT_FOUND`.
  /// The client **may** call `QueryWriteStatus()` at any time to determine how
  /// much data has been processed for this resource. This is useful if the
  /// client is buffering data and needs to know which data can be safely
  /// evicted. For any sequence of `QueryWriteStatus()` calls for a given
  /// resource name, the sequence of returned `committed_size` values will be
  /// non-decreasing.
  Future<googlebytestreambytestream.QueryWriteStatusResponse> queryWriteStatus(
    googlebytestreambytestream.QueryWriteStatusRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.ByteStream.queryWriteStatus,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }
}
