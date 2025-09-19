//
//  Generated code. Do not modify.
//  source: src/proto/videostore.proto
//

import "package:connectrpc/connect.dart" as connect;
import "videostore.pb.dart" as srcprotovideostore;

abstract final class videostoreService {
  /// Fully-qualified name of the videostoreService service.
  static const name = 'viammodules.service.videostore.v1.videostoreService';

  static const fetchStream = connect.Spec(
    '/$name/FetchStream',
    connect.StreamType.server,
    srcprotovideostore.FetchStreamRequest.new,
    srcprotovideostore.FetchStreamResponse.new,
  );

  /// Unary fetch between [from, to]
  static const fetch = connect.Spec(
    '/$name/Fetch',
    connect.StreamType.unary,
    srcprotovideostore.FetchRequest.new,
    srcprotovideostore.FetchResponse.new,
  );

  /// Unary save between [from, to]
  static const save = connect.Spec(
    '/$name/Save',
    connect.StreamType.unary,
    srcprotovideostore.SaveRequest.new,
    srcprotovideostore.SaveResponse.new,
  );
}
