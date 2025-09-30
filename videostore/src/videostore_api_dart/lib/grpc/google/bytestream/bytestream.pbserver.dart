///
//  Generated code. Do not modify.
//  source: google/bytestream/bytestream.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,constant_identifier_names,deprecated_member_use_from_same_package,directives_ordering,library_prefixes,non_constant_identifier_names,prefer_final_fields,return_of_invalid_type,unnecessary_const,unnecessary_import,unnecessary_this,unused_import,unused_shown_name

import 'dart:async' as $async;

import 'package:protobuf/protobuf.dart' as $pb;

import 'dart:core' as $core;
import 'bytestream.pb.dart' as $13;
import 'bytestream.pbjson.dart';

export 'bytestream.pb.dart';

abstract class ByteStreamServiceBase extends $pb.GeneratedService {
  $async.Future<$13.ReadResponse> read($pb.ServerContext ctx, $13.ReadRequest request);
  $async.Future<$13.WriteResponse> write($pb.ServerContext ctx, $13.WriteRequest request);
  $async.Future<$13.QueryWriteStatusResponse> queryWriteStatus($pb.ServerContext ctx, $13.QueryWriteStatusRequest request);

  $pb.GeneratedMessage createRequest($core.String method) {
    switch (method) {
      case 'Read': return $13.ReadRequest();
      case 'Write': return $13.WriteRequest();
      case 'QueryWriteStatus': return $13.QueryWriteStatusRequest();
      default: throw $core.ArgumentError('Unknown method: $method');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String method, $pb.GeneratedMessage request) {
    switch (method) {
      case 'Read': return this.read(ctx, request as $13.ReadRequest);
      case 'Write': return this.write(ctx, request as $13.WriteRequest);
      case 'QueryWriteStatus': return this.queryWriteStatus(ctx, request as $13.QueryWriteStatusRequest);
      default: throw $core.ArgumentError('Unknown method: $method');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => ByteStreamServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => ByteStreamServiceBase$messageJson;
}

