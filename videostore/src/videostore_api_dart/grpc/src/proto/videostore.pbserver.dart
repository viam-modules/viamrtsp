// This is a generated file - do not edit.
//
// Generated from src/proto/videostore.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'videostore.pb.dart' as $0;
import 'videostore.pbjson.dart';

export 'videostore.pb.dart';

abstract class videostoreServiceBase extends $pb.GeneratedService {
  $async.Future<$0.FetchStreamResponse> fetchStream(
      $pb.ServerContext ctx, $0.FetchStreamRequest request);
  $async.Future<$0.FetchResponse> fetch(
      $pb.ServerContext ctx, $0.FetchRequest request);
  $async.Future<$0.SaveResponse> save(
      $pb.ServerContext ctx, $0.SaveRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'FetchStream':
        return $0.FetchStreamRequest();
      case 'Fetch':
        return $0.FetchRequest();
      case 'Save':
        return $0.SaveRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'FetchStream':
        return fetchStream(ctx, request as $0.FetchStreamRequest);
      case 'Fetch':
        return fetch(ctx, request as $0.FetchRequest);
      case 'Save':
        return save(ctx, request as $0.SaveRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      videostoreServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => videostoreServiceBase$messageJson;
}
