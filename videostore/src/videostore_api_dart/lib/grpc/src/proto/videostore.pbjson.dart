//
//  Generated code. Do not modify.
//  source: src/proto/videostore.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use fetchStreamRequestDescriptor instead')
const FetchStreamRequest$json = {
  '1': 'FetchStreamRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'from', '3': 2, '4': 1, '5': 9, '10': 'from'},
    {'1': 'to', '3': 3, '4': 1, '5': 9, '10': 'to'},
    {'1': 'container', '3': 4, '4': 1, '5': 9, '10': 'container'},
  ],
};

/// Descriptor for `FetchStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchStreamRequestDescriptor = $convert.base64Decode(
    'ChJGZXRjaFN0cmVhbVJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgRmcm9tGAIgASgJUg'
    'Rmcm9tEg4KAnRvGAMgASgJUgJ0bxIcCgljb250YWluZXIYBCABKAlSCWNvbnRhaW5lcg==');

@$core.Deprecated('Use fetchStreamResponseDescriptor instead')
const FetchStreamResponse$json = {
  '1': 'FetchStreamResponse',
  '2': [
    {'1': 'video_data', '3': 1, '4': 1, '5': 12, '10': 'videoData'},
  ],
};

/// Descriptor for `FetchStreamResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchStreamResponseDescriptor = $convert.base64Decode(
    'ChNGZXRjaFN0cmVhbVJlc3BvbnNlEh0KCnZpZGVvX2RhdGEYASABKAxSCXZpZGVvRGF0YQ==');

@$core.Deprecated('Use fetchRequestDescriptor instead')
const FetchRequest$json = {
  '1': 'FetchRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'from', '3': 2, '4': 1, '5': 9, '10': 'from'},
    {'1': 'to', '3': 3, '4': 1, '5': 9, '10': 'to'},
    {'1': 'container', '3': 4, '4': 1, '5': 9, '10': 'container'},
  ],
};

/// Descriptor for `FetchRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchRequestDescriptor = $convert.base64Decode(
    'CgxGZXRjaFJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgRmcm9tGAIgASgJUgRmcm9tEg'
    '4KAnRvGAMgASgJUgJ0bxIcCgljb250YWluZXIYBCABKAlSCWNvbnRhaW5lcg==');

@$core.Deprecated('Use fetchResponseDescriptor instead')
const FetchResponse$json = {
  '1': 'FetchResponse',
  '2': [
    {'1': 'video_data', '3': 1, '4': 1, '5': 12, '10': 'videoData'},
  ],
};

/// Descriptor for `FetchResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchResponseDescriptor = $convert.base64Decode(
    'Cg1GZXRjaFJlc3BvbnNlEh0KCnZpZGVvX2RhdGEYASABKAxSCXZpZGVvRGF0YQ==');

@$core.Deprecated('Use saveRequestDescriptor instead')
const SaveRequest$json = {
  '1': 'SaveRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'from', '3': 2, '4': 1, '5': 9, '10': 'from'},
    {'1': 'to', '3': 3, '4': 1, '5': 9, '10': 'to'},
    {'1': 'container', '3': 4, '4': 1, '5': 9, '10': 'container'},
  ],
};

/// Descriptor for `SaveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List saveRequestDescriptor = $convert.base64Decode(
    'CgtTYXZlUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEhIKBGZyb20YAiABKAlSBGZyb20SDg'
    'oCdG8YAyABKAlSAnRvEhwKCWNvbnRhaW5lchgEIAEoCVIJY29udGFpbmVy');

@$core.Deprecated('Use saveResponseDescriptor instead')
const SaveResponse$json = {
  '1': 'SaveResponse',
  '2': [
    {'1': 'filename', '3': 1, '4': 1, '5': 9, '10': 'filename'},
  ],
};

/// Descriptor for `SaveResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List saveResponseDescriptor = $convert.base64Decode(
    'CgxTYXZlUmVzcG9uc2USGgoIZmlsZW5hbWUYASABKAlSCGZpbGVuYW1l');

@$core.Deprecated('Use getStorageStateRequestDescriptor instead')
const GetStorageStateRequest$json = {
  '1': 'GetStorageStateRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetStorageStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getStorageStateRequestDescriptor = $convert.base64Decode(
    'ChZHZXRTdG9yYWdlU3RhdGVSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use getStorageStateResponseDescriptor instead')
const GetStorageStateResponse$json = {
  '1': 'GetStorageStateResponse',
  '2': [
    {'1': 'state', '3': 1, '4': 1, '5': 11, '6': '.viammodules.service.videostore.v1.StorageState', '10': 'state'},
  ],
};

/// Descriptor for `GetStorageStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getStorageStateResponseDescriptor = $convert.base64Decode(
    'ChdHZXRTdG9yYWdlU3RhdGVSZXNwb25zZRJFCgVzdGF0ZRgBIAEoCzIvLnZpYW1tb2R1bGVzLn'
    'NlcnZpY2UudmlkZW9zdG9yZS52MS5TdG9yYWdlU3RhdGVSBXN0YXRl');

@$core.Deprecated('Use storageStateDescriptor instead')
const StorageState$json = {
  '1': 'StorageState',
  '2': [
    {'1': 'storage_used_bytes', '3': 1, '4': 1, '5': 3, '10': 'storageUsedBytes'},
    {'1': 'total_duration_ms', '3': 2, '4': 1, '5': 3, '10': 'totalDurationMs'},
    {'1': 'video_count', '3': 3, '4': 1, '5': 5, '10': 'videoCount'},
    {'1': 'ranges', '3': 4, '4': 3, '5': 11, '6': '.viammodules.service.videostore.v1.VideoRange', '10': 'ranges'},
    {'1': 'storage_limit_gb', '3': 5, '4': 1, '5': 5, '10': 'storageLimitGb'},
    {'1': 'device_storage_remaining_gb', '3': 6, '4': 1, '5': 1, '10': 'deviceStorageRemainingGb'},
    {'1': 'storage_path', '3': 7, '4': 1, '5': 9, '10': 'storagePath'},
  ],
};

/// Descriptor for `StorageState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storageStateDescriptor = $convert.base64Decode(
    'CgxTdG9yYWdlU3RhdGUSLAoSc3RvcmFnZV91c2VkX2J5dGVzGAEgASgDUhBzdG9yYWdlVXNlZE'
    'J5dGVzEioKEXRvdGFsX2R1cmF0aW9uX21zGAIgASgDUg90b3RhbER1cmF0aW9uTXMSHwoLdmlk'
    'ZW9fY291bnQYAyABKAVSCnZpZGVvQ291bnQSRQoGcmFuZ2VzGAQgAygLMi0udmlhbW1vZHVsZX'
    'Muc2VydmljZS52aWRlb3N0b3JlLnYxLlZpZGVvUmFuZ2VSBnJhbmdlcxIoChBzdG9yYWdlX2xp'
    'bWl0X2diGAUgASgFUg5zdG9yYWdlTGltaXRHYhI9ChtkZXZpY2Vfc3RvcmFnZV9yZW1haW5pbm'
    'dfZ2IYBiABKAFSGGRldmljZVN0b3JhZ2VSZW1haW5pbmdHYhIhCgxzdG9yYWdlX3BhdGgYByAB'
    'KAlSC3N0b3JhZ2VQYXRo');

@$core.Deprecated('Use videoRangeDescriptor instead')
const VideoRange$json = {
  '1': 'VideoRange',
  '2': [
    {'1': 'from', '3': 1, '4': 1, '5': 9, '10': 'from'},
    {'1': 'to', '3': 2, '4': 1, '5': 9, '10': 'to'},
  ],
};

/// Descriptor for `VideoRange`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List videoRangeDescriptor = $convert.base64Decode(
    'CgpWaWRlb1JhbmdlEhIKBGZyb20YASABKAlSBGZyb20SDgoCdG8YAiABKAlSAnRv');

