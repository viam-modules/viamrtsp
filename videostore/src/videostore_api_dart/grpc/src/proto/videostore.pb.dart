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

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class FetchStreamRequest extends $pb.GeneratedMessage {
  factory FetchStreamRequest({
    $core.String? name,
    $core.String? from,
    $core.String? to,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (from != null) result.from = from;
    if (to != null) result.to = to;
    return result;
  }

  FetchStreamRequest._();

  factory FetchStreamRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FetchStreamRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FetchStreamRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'viammodules.service.videostore.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'from')
    ..aOS(3, _omitFieldNames ? '' : 'to')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchStreamRequest clone() => FetchStreamRequest()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchStreamRequest copyWith(void Function(FetchStreamRequest) updates) =>
      super.copyWith((message) => updates(message as FetchStreamRequest))
          as FetchStreamRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchStreamRequest create() => FetchStreamRequest._();
  @$core.override
  FetchStreamRequest createEmptyInstance() => create();
  static $pb.PbList<FetchStreamRequest> createRepeated() =>
      $pb.PbList<FetchStreamRequest>();
  @$core.pragma('dart2js:noInline')
  static FetchStreamRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FetchStreamRequest>(create);
  static FetchStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  /// Date-time in format YYYY-MM-DD_HH-MM-SS
  @$pb.TagNumber(2)
  $core.String get from => $_getSZ(1);
  @$pb.TagNumber(2)
  set from($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasFrom() => $_has(1);
  @$pb.TagNumber(2)
  void clearFrom() => $_clearField(2);

  /// Date-time in format YYYY-MM-DD_HH-MM-SS
  @$pb.TagNumber(3)
  $core.String get to => $_getSZ(2);
  @$pb.TagNumber(3)
  set to($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTo() => $_has(2);
  @$pb.TagNumber(3)
  void clearTo() => $_clearField(3);
}

class FetchStreamResponse extends $pb.GeneratedMessage {
  factory FetchStreamResponse({
    $core.List<$core.int>? videoData,
  }) {
    final result = create();
    if (videoData != null) result.videoData = videoData;
    return result;
  }

  FetchStreamResponse._();

  factory FetchStreamResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FetchStreamResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FetchStreamResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'viammodules.service.videostore.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'videoData', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchStreamResponse clone() => FetchStreamResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchStreamResponse copyWith(void Function(FetchStreamResponse) updates) =>
      super.copyWith((message) => updates(message as FetchStreamResponse))
          as FetchStreamResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchStreamResponse create() => FetchStreamResponse._();
  @$core.override
  FetchStreamResponse createEmptyInstance() => create();
  static $pb.PbList<FetchStreamResponse> createRepeated() =>
      $pb.PbList<FetchStreamResponse>();
  @$core.pragma('dart2js:noInline')
  static FetchStreamResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FetchStreamResponse>(create);
  static FetchStreamResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get videoData => $_getN(0);
  @$pb.TagNumber(1)
  set videoData($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVideoData() => $_has(0);
  @$pb.TagNumber(1)
  void clearVideoData() => $_clearField(1);
}

class FetchRequest extends $pb.GeneratedMessage {
  factory FetchRequest({
    $core.String? name,
    $core.String? from,
    $core.String? to,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (from != null) result.from = from;
    if (to != null) result.to = to;
    return result;
  }

  FetchRequest._();

  factory FetchRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FetchRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FetchRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'viammodules.service.videostore.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'from')
    ..aOS(3, _omitFieldNames ? '' : 'to')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchRequest clone() => FetchRequest()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchRequest copyWith(void Function(FetchRequest) updates) =>
      super.copyWith((message) => updates(message as FetchRequest))
          as FetchRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchRequest create() => FetchRequest._();
  @$core.override
  FetchRequest createEmptyInstance() => create();
  static $pb.PbList<FetchRequest> createRepeated() =>
      $pb.PbList<FetchRequest>();
  @$core.pragma('dart2js:noInline')
  static FetchRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FetchRequest>(create);
  static FetchRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  /// Date-time in format YYYY-MM-DD_HH-MM-SS
  @$pb.TagNumber(2)
  $core.String get from => $_getSZ(1);
  @$pb.TagNumber(2)
  set from($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasFrom() => $_has(1);
  @$pb.TagNumber(2)
  void clearFrom() => $_clearField(2);

  /// Date-time in format YYYY-MM-DD_HH-MM-SS
  @$pb.TagNumber(3)
  $core.String get to => $_getSZ(2);
  @$pb.TagNumber(3)
  set to($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTo() => $_has(2);
  @$pb.TagNumber(3)
  void clearTo() => $_clearField(3);
}

class FetchResponse extends $pb.GeneratedMessage {
  factory FetchResponse({
    $core.List<$core.int>? videoData,
  }) {
    final result = create();
    if (videoData != null) result.videoData = videoData;
    return result;
  }

  FetchResponse._();

  factory FetchResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FetchResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FetchResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'viammodules.service.videostore.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'videoData', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchResponse clone() => FetchResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchResponse copyWith(void Function(FetchResponse) updates) =>
      super.copyWith((message) => updates(message as FetchResponse))
          as FetchResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchResponse create() => FetchResponse._();
  @$core.override
  FetchResponse createEmptyInstance() => create();
  static $pb.PbList<FetchResponse> createRepeated() =>
      $pb.PbList<FetchResponse>();
  @$core.pragma('dart2js:noInline')
  static FetchResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FetchResponse>(create);
  static FetchResponse? _defaultInstance;

  /// Raw video bytes for the requested interval
  @$pb.TagNumber(1)
  $core.List<$core.int> get videoData => $_getN(0);
  @$pb.TagNumber(1)
  set videoData($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVideoData() => $_has(0);
  @$pb.TagNumber(1)
  void clearVideoData() => $_clearField(1);
}

class SaveRequest extends $pb.GeneratedMessage {
  factory SaveRequest({
    $core.String? name,
    $core.String? from,
    $core.String? to,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (from != null) result.from = from;
    if (to != null) result.to = to;
    return result;
  }

  SaveRequest._();

  factory SaveRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SaveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SaveRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'viammodules.service.videostore.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'from')
    ..aOS(3, _omitFieldNames ? '' : 'to')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SaveRequest clone() => SaveRequest()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SaveRequest copyWith(void Function(SaveRequest) updates) =>
      super.copyWith((message) => updates(message as SaveRequest))
          as SaveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SaveRequest create() => SaveRequest._();
  @$core.override
  SaveRequest createEmptyInstance() => create();
  static $pb.PbList<SaveRequest> createRepeated() => $pb.PbList<SaveRequest>();
  @$core.pragma('dart2js:noInline')
  static SaveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SaveRequest>(create);
  static SaveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  /// Date-time in format YYYY-MM-DD_HH-MM-SS
  @$pb.TagNumber(2)
  $core.String get from => $_getSZ(1);
  @$pb.TagNumber(2)
  set from($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasFrom() => $_has(1);
  @$pb.TagNumber(2)
  void clearFrom() => $_clearField(2);

  /// Date-time in format YYYY-MM-DD_HH-MM-SS
  @$pb.TagNumber(3)
  $core.String get to => $_getSZ(2);
  @$pb.TagNumber(3)
  set to($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTo() => $_has(2);
  @$pb.TagNumber(3)
  void clearTo() => $_clearField(3);
}

class SaveResponse extends $pb.GeneratedMessage {
  factory SaveResponse({
    $core.String? filename,
  }) {
    final result = create();
    if (filename != null) result.filename = filename;
    return result;
  }

  SaveResponse._();

  factory SaveResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SaveResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SaveResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'viammodules.service.videostore.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'filename')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SaveResponse clone() => SaveResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SaveResponse copyWith(void Function(SaveResponse) updates) =>
      super.copyWith((message) => updates(message as SaveResponse))
          as SaveResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SaveResponse create() => SaveResponse._();
  @$core.override
  SaveResponse createEmptyInstance() => create();
  static $pb.PbList<SaveResponse> createRepeated() =>
      $pb.PbList<SaveResponse>();
  @$core.pragma('dart2js:noInline')
  static SaveResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SaveResponse>(create);
  static SaveResponse? _defaultInstance;

  /// Filename (or identifier) of the saved artifact
  @$pb.TagNumber(1)
  $core.String get filename => $_getSZ(0);
  @$pb.TagNumber(1)
  set filename($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFilename() => $_has(0);
  @$pb.TagNumber(1)
  void clearFilename() => $_clearField(1);
}

class videostoreServiceApi {
  final $pb.RpcClient _client;

  videostoreServiceApi(this._client);

  $async.Future<FetchStreamResponse> fetchStream(
          $pb.ClientContext? ctx, FetchStreamRequest request) =>
      _client.invoke<FetchStreamResponse>(ctx, 'videostoreService',
          'FetchStream', request, FetchStreamResponse());

  /// Unary fetch between [from, to]
  $async.Future<FetchResponse> fetch(
          $pb.ClientContext? ctx, FetchRequest request) =>
      _client.invoke<FetchResponse>(
          ctx, 'videostoreService', 'Fetch', request, FetchResponse());

  /// Unary save between [from, to]
  $async.Future<SaveResponse> save(
          $pb.ClientContext? ctx, SaveRequest request) =>
      _client.invoke<SaveResponse>(
          ctx, 'videostoreService', 'Save', request, SaveResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
