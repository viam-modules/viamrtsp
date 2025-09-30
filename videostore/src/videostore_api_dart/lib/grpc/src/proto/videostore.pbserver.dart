///
//  Generated code. Do not modify.
//  source: src/proto/videostore.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,constant_identifier_names,deprecated_member_use_from_same_package,directives_ordering,library_prefixes,non_constant_identifier_names,prefer_final_fields,return_of_invalid_type,unnecessary_const,unnecessary_import,unnecessary_this,unused_import,unused_shown_name

import 'dart:async' as $async;

import 'package:protobuf/protobuf.dart' as $pb;

import 'dart:core' as $core;
import 'videostore.pb.dart' as $0;
import 'videostore.pbjson.dart';

export 'videostore.pb.dart';

abstract class videostoreServiceBase extends $pb.GeneratedService {
  $async.Future<$0.FetchStreamResponse> fetchStream($pb.ServerContext ctx, $0.FetchStreamRequest request);
  $async.Future<$0.FetchResponse> fetch($pb.ServerContext ctx, $0.FetchRequest request);
  $async.Future<$0.SaveResponse> save($pb.ServerContext ctx, $0.SaveRequest request);

  $pb.GeneratedMessage createRequest($core.String method) {
    switch (method) {
      case 'FetchStream': return $0.FetchStreamRequest();
      case 'Fetch': return $0.FetchRequest();
      case 'Save': return $0.SaveRequest();
      default: throw $core.ArgumentError('Unknown method: $method');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String method, $pb.GeneratedMessage request) {
    switch (method) {
      case 'FetchStream': return this.fetchStream(ctx, request as $0.FetchStreamRequest);
      case 'Fetch': return this.fetch(ctx, request as $0.FetchRequest);
      case 'Save': return this.save(ctx, request as $0.SaveRequest);
      default: throw $core.ArgumentError('Unknown method: $method');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => videostoreServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => videostoreServiceBase$messageJson;
}

