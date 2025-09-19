//
//  Generated code. Do not modify.
//  source: src/proto/videostore.proto
//

import "package:connectrpc/connect.dart" as connect;
import "videostore.pb.dart" as srcprotovideostore;
import "videostore.connect.spec.dart" as specs;

extension type videostoreServiceClient (connect.Transport _transport) {
  Stream<srcprotovideostore.FetchStreamResponse> fetchStream(
    srcprotovideostore.FetchStreamRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).server(
      specs.videostoreService.fetchStream,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  /// Unary fetch between [from, to]
  Future<srcprotovideostore.FetchResponse> fetch(
    srcprotovideostore.FetchRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.videostoreService.fetch,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  /// Unary save between [from, to]
  Future<srcprotovideostore.SaveResponse> save(
    srcprotovideostore.SaveRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.videostoreService.save,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }
}
