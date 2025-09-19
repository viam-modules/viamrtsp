// This is a generated file - do not edit.
//
// Generated from google/bytestream/bytestream.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'bytestream.pb.dart' as $0;
import 'bytestream.pbjson.dart';

export 'bytestream.pb.dart';

abstract class ByteStreamServiceBase extends $pb.GeneratedService {
  $async.Future<$0.ReadResponse> read(
      $pb.ServerContext ctx, $0.ReadRequest request);
  $async.Future<$0.WriteResponse> write(
      $pb.ServerContext ctx, $0.WriteRequest request);
  $async.Future<$0.QueryWriteStatusResponse> queryWriteStatus(
      $pb.ServerContext ctx, $0.QueryWriteStatusRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'Read':
        return $0.ReadRequest();
      case 'Write':
        return $0.WriteRequest();
      case 'QueryWriteStatus':
        return $0.QueryWriteStatusRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'Read':
        return read(ctx, request as $0.ReadRequest);
      case 'Write':
        return write(ctx, request as $0.WriteRequest);
      case 'QueryWriteStatus':
        return queryWriteStatus(ctx, request as $0.QueryWriteStatusRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      ByteStreamServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => ByteStreamServiceBase$messageJson;
}
