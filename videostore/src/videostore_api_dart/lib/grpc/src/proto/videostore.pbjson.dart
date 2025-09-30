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
  ],
};

/// Descriptor for `FetchStreamRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchStreamRequestDescriptor = $convert.base64Decode(
    'ChJGZXRjaFN0cmVhbVJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgRmcm9tGAIgASgJUg'
    'Rmcm9tEg4KAnRvGAMgASgJUgJ0bw==');

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
  ],
};

/// Descriptor for `FetchRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchRequestDescriptor = $convert.base64Decode(
    'CgxGZXRjaFJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgRmcm9tGAIgASgJUgRmcm9tEg'
    '4KAnRvGAMgASgJUgJ0bw==');

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
  ],
};

/// Descriptor for `SaveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List saveRequestDescriptor = $convert.base64Decode(
    'CgtTYXZlUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEhIKBGZyb20YAiABKAlSBGZyb20SDg'
    'oCdG8YAyABKAlSAnRv');

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

