// This is a generated file - do not edit.
//
// Generated from src/proto/videostore.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, unused_import

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
final $typed_data.Uint8List saveResponseDescriptor = $convert
    .base64Decode('CgxTYXZlUmVzcG9uc2USGgoIZmlsZW5hbWUYASABKAlSCGZpbGVuYW1l');

const $core.Map<$core.String, $core.dynamic> videostoreServiceBase$json = {
  '1': 'videostoreService',
  '2': [
    {
      '1': 'FetchStream',
      '2': '.viammodules.service.videostore.v1.FetchStreamRequest',
      '3': '.viammodules.service.videostore.v1.FetchStreamResponse',
      '6': true
    },
    {
      '1': 'Fetch',
      '2': '.viammodules.service.videostore.v1.FetchRequest',
      '3': '.viammodules.service.videostore.v1.FetchResponse'
    },
    {
      '1': 'Save',
      '2': '.viammodules.service.videostore.v1.SaveRequest',
      '3': '.viammodules.service.videostore.v1.SaveResponse'
    },
  ],
};

@$core.Deprecated('Use videostoreServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    videostoreServiceBase$messageJson = {
  '.viammodules.service.videostore.v1.FetchStreamRequest':
      FetchStreamRequest$json,
  '.viammodules.service.videostore.v1.FetchStreamResponse':
      FetchStreamResponse$json,
  '.viammodules.service.videostore.v1.FetchRequest': FetchRequest$json,
  '.viammodules.service.videostore.v1.FetchResponse': FetchResponse$json,
  '.viammodules.service.videostore.v1.SaveRequest': SaveRequest$json,
  '.viammodules.service.videostore.v1.SaveResponse': SaveResponse$json,
};

/// Descriptor for `videostoreService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List videostoreServiceDescriptor = $convert.base64Decode(
    'ChF2aWRlb3N0b3JlU2VydmljZRJ+CgtGZXRjaFN0cmVhbRI1LnZpYW1tb2R1bGVzLnNlcnZpY2'
    'UudmlkZW9zdG9yZS52MS5GZXRjaFN0cmVhbVJlcXVlc3QaNi52aWFtbW9kdWxlcy5zZXJ2aWNl'
    'LnZpZGVvc3RvcmUudjEuRmV0Y2hTdHJlYW1SZXNwb25zZTABEmoKBUZldGNoEi8udmlhbW1vZH'
    'VsZXMuc2VydmljZS52aWRlb3N0b3JlLnYxLkZldGNoUmVxdWVzdBowLnZpYW1tb2R1bGVzLnNl'
    'cnZpY2UudmlkZW9zdG9yZS52MS5GZXRjaFJlc3BvbnNlEmcKBFNhdmUSLi52aWFtbW9kdWxlcy'
    '5zZXJ2aWNlLnZpZGVvc3RvcmUudjEuU2F2ZVJlcXVlc3QaLy52aWFtbW9kdWxlcy5zZXJ2aWNl'
    'LnZpZGVvc3RvcmUudjEuU2F2ZVJlc3BvbnNl');
