//
//  Generated code. Do not modify.
//  source: src/proto/videostore.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:grpc/service_api.dart' as $grpc;
import 'package:protobuf/protobuf.dart' as $pb;

import 'videostore.pb.dart' as $0;

export 'videostore.pb.dart';

@$pb.GrpcServiceName('viammodules.service.videostore.v1.videostoreService')
class videostoreServiceClient extends $grpc.Client {
  static final _$fetchStream = $grpc.ClientMethod<$0.FetchStreamRequest, $0.FetchStreamResponse>(
      '/viammodules.service.videostore.v1.videostoreService/FetchStream',
      ($0.FetchStreamRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.FetchStreamResponse.fromBuffer(value));
  static final _$fetch = $grpc.ClientMethod<$0.FetchRequest, $0.FetchResponse>(
      '/viammodules.service.videostore.v1.videostoreService/Fetch',
      ($0.FetchRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.FetchResponse.fromBuffer(value));
  static final _$save = $grpc.ClientMethod<$0.SaveRequest, $0.SaveResponse>(
      '/viammodules.service.videostore.v1.videostoreService/Save',
      ($0.SaveRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.SaveResponse.fromBuffer(value));
  static final _$getStorageState = $grpc.ClientMethod<$0.GetStorageStateRequest, $0.GetStorageStateResponse>(
      '/viammodules.service.videostore.v1.videostoreService/GetStorageState',
      ($0.GetStorageStateRequest value) => value.writeToBuffer(),
      ($core.List<$core.int> value) => $0.GetStorageStateResponse.fromBuffer(value));

  videostoreServiceClient($grpc.ClientChannel channel,
      {$grpc.CallOptions? options,
      $core.Iterable<$grpc.ClientInterceptor>? interceptors})
      : super(channel, options: options,
        interceptors: interceptors);

  $grpc.ResponseStream<$0.FetchStreamResponse> fetchStream($0.FetchStreamRequest request, {$grpc.CallOptions? options}) {
    return $createStreamingCall(_$fetchStream, $async.Stream.fromIterable([request]), options: options);
  }

  $grpc.ResponseFuture<$0.FetchResponse> fetch($0.FetchRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$fetch, request, options: options);
  }

  $grpc.ResponseFuture<$0.SaveResponse> save($0.SaveRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$save, request, options: options);
  }

  $grpc.ResponseFuture<$0.GetStorageStateResponse> getStorageState($0.GetStorageStateRequest request, {$grpc.CallOptions? options}) {
    return $createUnaryCall(_$getStorageState, request, options: options);
  }
}

@$pb.GrpcServiceName('viammodules.service.videostore.v1.videostoreService')
abstract class videostoreServiceBase extends $grpc.Service {
  $core.String get $name => 'viammodules.service.videostore.v1.videostoreService';

  videostoreServiceBase() {
    $addMethod($grpc.ServiceMethod<$0.FetchStreamRequest, $0.FetchStreamResponse>(
        'FetchStream',
        fetchStream_Pre,
        false,
        true,
        ($core.List<$core.int> value) => $0.FetchStreamRequest.fromBuffer(value),
        ($0.FetchStreamResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.FetchRequest, $0.FetchResponse>(
        'Fetch',
        fetch_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.FetchRequest.fromBuffer(value),
        ($0.FetchResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.SaveRequest, $0.SaveResponse>(
        'Save',
        save_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.SaveRequest.fromBuffer(value),
        ($0.SaveResponse value) => value.writeToBuffer()));
    $addMethod($grpc.ServiceMethod<$0.GetStorageStateRequest, $0.GetStorageStateResponse>(
        'GetStorageState',
        getStorageState_Pre,
        false,
        false,
        ($core.List<$core.int> value) => $0.GetStorageStateRequest.fromBuffer(value),
        ($0.GetStorageStateResponse value) => value.writeToBuffer()));
  }

  $async.Stream<$0.FetchStreamResponse> fetchStream_Pre($grpc.ServiceCall call, $async.Future<$0.FetchStreamRequest> request) async* {
    yield* fetchStream(call, await request);
  }

  $async.Future<$0.FetchResponse> fetch_Pre($grpc.ServiceCall call, $async.Future<$0.FetchRequest> request) async {
    return fetch(call, await request);
  }

  $async.Future<$0.SaveResponse> save_Pre($grpc.ServiceCall call, $async.Future<$0.SaveRequest> request) async {
    return save(call, await request);
  }

  $async.Future<$0.GetStorageStateResponse> getStorageState_Pre($grpc.ServiceCall call, $async.Future<$0.GetStorageStateRequest> request) async {
    return getStorageState(call, await request);
  }

  $async.Stream<$0.FetchStreamResponse> fetchStream($grpc.ServiceCall call, $0.FetchStreamRequest request);
  $async.Future<$0.FetchResponse> fetch($grpc.ServiceCall call, $0.FetchRequest request);
  $async.Future<$0.SaveResponse> save($grpc.ServiceCall call, $0.SaveRequest request);
  $async.Future<$0.GetStorageStateResponse> getStorageState($grpc.ServiceCall call, $0.GetStorageStateRequest request);
}
