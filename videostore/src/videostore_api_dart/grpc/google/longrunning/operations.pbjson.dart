// This is a generated file - do not edit.
//
// Generated from google/longrunning/operations.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

import '../protobuf/any.pbjson.dart' as $0;
import '../protobuf/duration.pbjson.dart' as $2;
import '../protobuf/empty.pbjson.dart' as $3;
import '../rpc/status.pbjson.dart' as $1;

@$core.Deprecated('Use operationDescriptor instead')
const Operation$json = {
  '1': 'Operation',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'metadata',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Any',
      '10': 'metadata'
    },
    {'1': 'done', '3': 3, '4': 1, '5': 8, '10': 'done'},
    {
      '1': 'error',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.google.rpc.Status',
      '9': 0,
      '10': 'error'
    },
    {
      '1': 'response',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Any',
      '9': 0,
      '10': 'response'
    },
  ],
  '8': [
    {'1': 'result'},
  ],
};

/// Descriptor for `Operation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operationDescriptor = $convert.base64Decode(
    'CglPcGVyYXRpb24SEgoEbmFtZRgBIAEoCVIEbmFtZRIwCghtZXRhZGF0YRgCIAEoCzIULmdvb2'
    'dsZS5wcm90b2J1Zi5BbnlSCG1ldGFkYXRhEhIKBGRvbmUYAyABKAhSBGRvbmUSKgoFZXJyb3IY'
    'BCABKAsyEi5nb29nbGUucnBjLlN0YXR1c0gAUgVlcnJvchIyCghyZXNwb25zZRgFIAEoCzIULm'
    'dvb2dsZS5wcm90b2J1Zi5BbnlIAFIIcmVzcG9uc2VCCAoGcmVzdWx0');

@$core.Deprecated('Use getOperationRequestDescriptor instead')
const GetOperationRequest$json = {
  '1': 'GetOperationRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetOperationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getOperationRequestDescriptor = $convert
    .base64Decode('ChNHZXRPcGVyYXRpb25SZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use listOperationsRequestDescriptor instead')
const ListOperationsRequest$json = {
  '1': 'ListOperationsRequest',
  '2': [
    {'1': 'name', '3': 4, '4': 1, '5': 9, '10': 'name'},
    {'1': 'filter', '3': 1, '4': 1, '5': 9, '10': 'filter'},
    {'1': 'page_size', '3': 2, '4': 1, '5': 5, '10': 'pageSize'},
    {'1': 'page_token', '3': 3, '4': 1, '5': 9, '10': 'pageToken'},
  ],
};

/// Descriptor for `ListOperationsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperationsRequestDescriptor = $convert.base64Decode(
    'ChVMaXN0T3BlcmF0aW9uc1JlcXVlc3QSEgoEbmFtZRgEIAEoCVIEbmFtZRIWCgZmaWx0ZXIYAS'
    'ABKAlSBmZpbHRlchIbCglwYWdlX3NpemUYAiABKAVSCHBhZ2VTaXplEh0KCnBhZ2VfdG9rZW4Y'
    'AyABKAlSCXBhZ2VUb2tlbg==');

@$core.Deprecated('Use listOperationsResponseDescriptor instead')
const ListOperationsResponse$json = {
  '1': 'ListOperationsResponse',
  '2': [
    {
      '1': 'operations',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.google.longrunning.Operation',
      '10': 'operations'
    },
    {'1': 'next_page_token', '3': 2, '4': 1, '5': 9, '10': 'nextPageToken'},
  ],
};

/// Descriptor for `ListOperationsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOperationsResponseDescriptor = $convert.base64Decode(
    'ChZMaXN0T3BlcmF0aW9uc1Jlc3BvbnNlEj0KCm9wZXJhdGlvbnMYASADKAsyHS5nb29nbGUubG'
    '9uZ3J1bm5pbmcuT3BlcmF0aW9uUgpvcGVyYXRpb25zEiYKD25leHRfcGFnZV90b2tlbhgCIAEo'
    'CVINbmV4dFBhZ2VUb2tlbg==');

@$core.Deprecated('Use cancelOperationRequestDescriptor instead')
const CancelOperationRequest$json = {
  '1': 'CancelOperationRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `CancelOperationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cancelOperationRequestDescriptor =
    $convert.base64Decode(
        'ChZDYW5jZWxPcGVyYXRpb25SZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use deleteOperationRequestDescriptor instead')
const DeleteOperationRequest$json = {
  '1': 'DeleteOperationRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `DeleteOperationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deleteOperationRequestDescriptor =
    $convert.base64Decode(
        'ChZEZWxldGVPcGVyYXRpb25SZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use waitOperationRequestDescriptor instead')
const WaitOperationRequest$json = {
  '1': 'WaitOperationRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'timeout',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.google.protobuf.Duration',
      '10': 'timeout'
    },
  ],
};

/// Descriptor for `WaitOperationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List waitOperationRequestDescriptor = $convert.base64Decode(
    'ChRXYWl0T3BlcmF0aW9uUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEjMKB3RpbWVvdXQYAi'
    'ABKAsyGS5nb29nbGUucHJvdG9idWYuRHVyYXRpb25SB3RpbWVvdXQ=');

@$core.Deprecated('Use operationInfoDescriptor instead')
const OperationInfo$json = {
  '1': 'OperationInfo',
  '2': [
    {'1': 'response_type', '3': 1, '4': 1, '5': 9, '10': 'responseType'},
    {'1': 'metadata_type', '3': 2, '4': 1, '5': 9, '10': 'metadataType'},
  ],
};

/// Descriptor for `OperationInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List operationInfoDescriptor = $convert.base64Decode(
    'Cg1PcGVyYXRpb25JbmZvEiMKDXJlc3BvbnNlX3R5cGUYASABKAlSDHJlc3BvbnNlVHlwZRIjCg'
    '1tZXRhZGF0YV90eXBlGAIgASgJUgxtZXRhZGF0YVR5cGU=');

const $core.Map<$core.String, $core.dynamic> OperationsServiceBase$json = {
  '1': 'Operations',
  '2': [
    {
      '1': 'ListOperations',
      '2': '.google.longrunning.ListOperationsRequest',
      '3': '.google.longrunning.ListOperationsResponse',
      '4': {
        '1051': ['name,filter'],
      },
    },
    {
      '1': 'GetOperation',
      '2': '.google.longrunning.GetOperationRequest',
      '3': '.google.longrunning.Operation',
      '4': {
        '1051': ['name'],
      },
    },
    {
      '1': 'DeleteOperation',
      '2': '.google.longrunning.DeleteOperationRequest',
      '3': '.google.protobuf.Empty',
      '4': {
        '1051': ['name'],
      },
    },
    {
      '1': 'CancelOperation',
      '2': '.google.longrunning.CancelOperationRequest',
      '3': '.google.protobuf.Empty',
      '4': {
        '1051': ['name'],
      },
    },
    {
      '1': 'WaitOperation',
      '2': '.google.longrunning.WaitOperationRequest',
      '3': '.google.longrunning.Operation',
      '4': {}
    },
  ],
  '3': {'1049': 'longrunning.googleapis.com'},
};

@$core.Deprecated('Use operationsServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    OperationsServiceBase$messageJson = {
  '.google.longrunning.ListOperationsRequest': ListOperationsRequest$json,
  '.google.longrunning.ListOperationsResponse': ListOperationsResponse$json,
  '.google.longrunning.Operation': Operation$json,
  '.google.protobuf.Any': $0.Any$json,
  '.google.rpc.Status': $1.Status$json,
  '.google.longrunning.GetOperationRequest': GetOperationRequest$json,
  '.google.longrunning.DeleteOperationRequest': DeleteOperationRequest$json,
  '.google.protobuf.Empty': $3.Empty$json,
  '.google.longrunning.CancelOperationRequest': CancelOperationRequest$json,
  '.google.longrunning.WaitOperationRequest': WaitOperationRequest$json,
  '.google.protobuf.Duration': $2.Duration$json,
};

/// Descriptor for `Operations`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List operationsServiceDescriptor = $convert.base64Decode(
    'CgpPcGVyYXRpb25zEpQBCg5MaXN0T3BlcmF0aW9ucxIpLmdvb2dsZS5sb25ncnVubmluZy5MaX'
    'N0T3BlcmF0aW9uc1JlcXVlc3QaKi5nb29nbGUubG9uZ3J1bm5pbmcuTGlzdE9wZXJhdGlvbnNS'
    'ZXNwb25zZSIr2kELbmFtZSxmaWx0ZXKC0+STAhcSFS92MS97bmFtZT1vcGVyYXRpb25zfRJ/Cg'
    'xHZXRPcGVyYXRpb24SJy5nb29nbGUubG9uZ3J1bm5pbmcuR2V0T3BlcmF0aW9uUmVxdWVzdBod'
    'Lmdvb2dsZS5sb25ncnVubmluZy5PcGVyYXRpb24iJ9pBBG5hbWWC0+STAhoSGC92MS97bmFtZT'
    '1vcGVyYXRpb25zLyoqfRJ+Cg9EZWxldGVPcGVyYXRpb24SKi5nb29nbGUubG9uZ3J1bm5pbmcu'
    'RGVsZXRlT3BlcmF0aW9uUmVxdWVzdBoWLmdvb2dsZS5wcm90b2J1Zi5FbXB0eSIn2kEEbmFtZY'
    'LT5JMCGioYL3YxL3tuYW1lPW9wZXJhdGlvbnMvKip9EogBCg9DYW5jZWxPcGVyYXRpb24SKi5n'
    'b29nbGUubG9uZ3J1bm5pbmcuQ2FuY2VsT3BlcmF0aW9uUmVxdWVzdBoWLmdvb2dsZS5wcm90b2'
    'J1Zi5FbXB0eSIx2kEEbmFtZYLT5JMCJDoBKiIfL3YxL3tuYW1lPW9wZXJhdGlvbnMvKip9OmNh'
    'bmNlbBJaCg1XYWl0T3BlcmF0aW9uEiguZ29vZ2xlLmxvbmdydW5uaW5nLldhaXRPcGVyYXRpb2'
    '5SZXF1ZXN0Gh0uZ29vZ2xlLmxvbmdydW5uaW5nLk9wZXJhdGlvbiIAGh3KQRpsb25ncnVubmlu'
    'Zy5nb29nbGVhcGlzLmNvbQ==');
