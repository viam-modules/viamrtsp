//
//  Generated code. Do not modify.
//  source: src/proto/videostore.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

class SaveRequest extends $pb.GeneratedMessage {
  factory SaveRequest({
    $core.String? name,
    $core.String? from,
    $core.String? to,
    $core.String? container,
    $core.String? metadata,
    $core.bool? async,
    $core.String? requestId,
  }) {
    final $result = create();
    if (name != null) {
      $result.name = name;
    }
    if (from != null) {
      $result.from = from;
    }
    if (to != null) {
      $result.to = to;
    }
    if (container != null) {
      $result.container = container;
    }
    if (metadata != null) {
      $result.metadata = metadata;
    }
    if (async != null) {
      $result.async = async;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  SaveRequest._() : super();
  factory SaveRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SaveRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SaveRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'from')
    ..aOS(3, _omitFieldNames ? '' : 'to')
    ..aOS(4, _omitFieldNames ? '' : 'container')
    ..aOS(5, _omitFieldNames ? '' : 'metadata')
    ..aOB(6, _omitFieldNames ? '' : 'async')
    ..aOS(7, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SaveRequest clone() => SaveRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SaveRequest copyWith(void Function(SaveRequest) updates) => super.copyWith((message) => updates(message as SaveRequest)) as SaveRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SaveRequest create() => SaveRequest._();
  SaveRequest createEmptyInstance() => create();
  static $pb.PbList<SaveRequest> createRepeated() => $pb.PbList<SaveRequest>();
  @$core.pragma('dart2js:noInline')
  static SaveRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SaveRequest>(create);
  static SaveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get from => $_getSZ(1);
  @$pb.TagNumber(2)
  set from($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasFrom() => $_has(1);
  @$pb.TagNumber(2)
  void clearFrom() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get to => $_getSZ(2);
  @$pb.TagNumber(3)
  set to($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasTo() => $_has(2);
  @$pb.TagNumber(3)
  void clearTo() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get container => $_getSZ(3);
  @$pb.TagNumber(4)
  set container($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasContainer() => $_has(3);
  @$pb.TagNumber(4)
  void clearContainer() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get metadata => $_getSZ(4);
  @$pb.TagNumber(5)
  set metadata($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasMetadata() => $_has(4);
  @$pb.TagNumber(5)
  void clearMetadata() => clearField(5);

  @$pb.TagNumber(6)
  $core.bool get async => $_getBF(5);
  @$pb.TagNumber(6)
  set async($core.bool v) { $_setBool(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasAsync() => $_has(5);
  @$pb.TagNumber(6)
  void clearAsync() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get requestId => $_getSZ(6);
  @$pb.TagNumber(7)
  set requestId($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasRequestId() => $_has(6);
  @$pb.TagNumber(7)
  void clearRequestId() => clearField(7);
}

class SaveResponse extends $pb.GeneratedMessage {
  factory SaveResponse({
    $core.String? filename,
    $core.String? requestId,
  }) {
    final $result = create();
    if (filename != null) {
      $result.filename = filename;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  SaveResponse._() : super();
  factory SaveResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SaveResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SaveResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'filename')
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SaveResponse clone() => SaveResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SaveResponse copyWith(void Function(SaveResponse) updates) => super.copyWith((message) => updates(message as SaveResponse)) as SaveResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SaveResponse create() => SaveResponse._();
  SaveResponse createEmptyInstance() => create();
  static $pb.PbList<SaveResponse> createRepeated() => $pb.PbList<SaveResponse>();
  @$core.pragma('dart2js:noInline')
  static SaveResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SaveResponse>(create);
  static SaveResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get filename => $_getSZ(0);
  @$pb.TagNumber(1)
  set filename($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasFilename() => $_has(0);
  @$pb.TagNumber(1)
  void clearFilename() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => clearField(2);
}

class FetchRequest extends $pb.GeneratedMessage {
  factory FetchRequest({
    $core.String? name,
    $core.String? from,
    $core.String? to,
    $core.String? container,
    $core.String? requestId,
  }) {
    final $result = create();
    if (name != null) {
      $result.name = name;
    }
    if (from != null) {
      $result.from = from;
    }
    if (to != null) {
      $result.to = to;
    }
    if (container != null) {
      $result.container = container;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  FetchRequest._() : super();
  factory FetchRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory FetchRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'FetchRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'from')
    ..aOS(3, _omitFieldNames ? '' : 'to')
    ..aOS(4, _omitFieldNames ? '' : 'container')
    ..aOS(5, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  FetchRequest clone() => FetchRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  FetchRequest copyWith(void Function(FetchRequest) updates) => super.copyWith((message) => updates(message as FetchRequest)) as FetchRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchRequest create() => FetchRequest._();
  FetchRequest createEmptyInstance() => create();
  static $pb.PbList<FetchRequest> createRepeated() => $pb.PbList<FetchRequest>();
  @$core.pragma('dart2js:noInline')
  static FetchRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FetchRequest>(create);
  static FetchRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get from => $_getSZ(1);
  @$pb.TagNumber(2)
  set from($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasFrom() => $_has(1);
  @$pb.TagNumber(2)
  void clearFrom() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get to => $_getSZ(2);
  @$pb.TagNumber(3)
  set to($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasTo() => $_has(2);
  @$pb.TagNumber(3)
  void clearTo() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get container => $_getSZ(3);
  @$pb.TagNumber(4)
  set container($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasContainer() => $_has(3);
  @$pb.TagNumber(4)
  void clearContainer() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get requestId => $_getSZ(4);
  @$pb.TagNumber(5)
  set requestId($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasRequestId() => $_has(4);
  @$pb.TagNumber(5)
  void clearRequestId() => clearField(5);
}

class FetchResponse extends $pb.GeneratedMessage {
  factory FetchResponse({
    $core.List<$core.int>? videoData,
    $core.String? requestId,
  }) {
    final $result = create();
    if (videoData != null) {
      $result.videoData = videoData;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  FetchResponse._() : super();
  factory FetchResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory FetchResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'FetchResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..a<$core.List<$core.int>>(1, _omitFieldNames ? '' : 'videoData', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  FetchResponse clone() => FetchResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  FetchResponse copyWith(void Function(FetchResponse) updates) => super.copyWith((message) => updates(message as FetchResponse)) as FetchResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchResponse create() => FetchResponse._();
  FetchResponse createEmptyInstance() => create();
  static $pb.PbList<FetchResponse> createRepeated() => $pb.PbList<FetchResponse>();
  @$core.pragma('dart2js:noInline')
  static FetchResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FetchResponse>(create);
  static FetchResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get videoData => $_getN(0);
  @$pb.TagNumber(1)
  set videoData($core.List<$core.int> v) { $_setBytes(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasVideoData() => $_has(0);
  @$pb.TagNumber(1)
  void clearVideoData() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => clearField(2);
}

class FetchStreamRequest extends $pb.GeneratedMessage {
  factory FetchStreamRequest({
    $core.String? name,
    $core.String? from,
    $core.String? to,
    $core.String? container,
    $core.String? requestId,
  }) {
    final $result = create();
    if (name != null) {
      $result.name = name;
    }
    if (from != null) {
      $result.from = from;
    }
    if (to != null) {
      $result.to = to;
    }
    if (container != null) {
      $result.container = container;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  FetchStreamRequest._() : super();
  factory FetchStreamRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory FetchStreamRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'FetchStreamRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'from')
    ..aOS(3, _omitFieldNames ? '' : 'to')
    ..aOS(4, _omitFieldNames ? '' : 'container')
    ..aOS(5, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  FetchStreamRequest clone() => FetchStreamRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  FetchStreamRequest copyWith(void Function(FetchStreamRequest) updates) => super.copyWith((message) => updates(message as FetchStreamRequest)) as FetchStreamRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchStreamRequest create() => FetchStreamRequest._();
  FetchStreamRequest createEmptyInstance() => create();
  static $pb.PbList<FetchStreamRequest> createRepeated() => $pb.PbList<FetchStreamRequest>();
  @$core.pragma('dart2js:noInline')
  static FetchStreamRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FetchStreamRequest>(create);
  static FetchStreamRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get from => $_getSZ(1);
  @$pb.TagNumber(2)
  set from($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasFrom() => $_has(1);
  @$pb.TagNumber(2)
  void clearFrom() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get to => $_getSZ(2);
  @$pb.TagNumber(3)
  set to($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasTo() => $_has(2);
  @$pb.TagNumber(3)
  void clearTo() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get container => $_getSZ(3);
  @$pb.TagNumber(4)
  set container($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasContainer() => $_has(3);
  @$pb.TagNumber(4)
  void clearContainer() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get requestId => $_getSZ(4);
  @$pb.TagNumber(5)
  set requestId($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasRequestId() => $_has(4);
  @$pb.TagNumber(5)
  void clearRequestId() => clearField(5);
}

class FetchStreamResponse extends $pb.GeneratedMessage {
  factory FetchStreamResponse({
    $core.List<$core.int>? videoData,
    $core.String? requestId,
  }) {
    final $result = create();
    if (videoData != null) {
      $result.videoData = videoData;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  FetchStreamResponse._() : super();
  factory FetchStreamResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory FetchStreamResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'FetchStreamResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..a<$core.List<$core.int>>(1, _omitFieldNames ? '' : 'videoData', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  FetchStreamResponse clone() => FetchStreamResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  FetchStreamResponse copyWith(void Function(FetchStreamResponse) updates) => super.copyWith((message) => updates(message as FetchStreamResponse)) as FetchStreamResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FetchStreamResponse create() => FetchStreamResponse._();
  FetchStreamResponse createEmptyInstance() => create();
  static $pb.PbList<FetchStreamResponse> createRepeated() => $pb.PbList<FetchStreamResponse>();
  @$core.pragma('dart2js:noInline')
  static FetchStreamResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FetchStreamResponse>(create);
  static FetchStreamResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get videoData => $_getN(0);
  @$pb.TagNumber(1)
  set videoData($core.List<$core.int> v) { $_setBytes(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasVideoData() => $_has(0);
  @$pb.TagNumber(1)
  void clearVideoData() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => clearField(2);
}

class GetStorageStateRequest extends $pb.GeneratedMessage {
  factory GetStorageStateRequest({
    $core.String? name,
    $core.String? requestId,
  }) {
    final $result = create();
    if (name != null) {
      $result.name = name;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  GetStorageStateRequest._() : super();
  factory GetStorageStateRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetStorageStateRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetStorageStateRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetStorageStateRequest clone() => GetStorageStateRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetStorageStateRequest copyWith(void Function(GetStorageStateRequest) updates) => super.copyWith((message) => updates(message as GetStorageStateRequest)) as GetStorageStateRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetStorageStateRequest create() => GetStorageStateRequest._();
  GetStorageStateRequest createEmptyInstance() => create();
  static $pb.PbList<GetStorageStateRequest> createRepeated() => $pb.PbList<GetStorageStateRequest>();
  @$core.pragma('dart2js:noInline')
  static GetStorageStateRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetStorageStateRequest>(create);
  static GetStorageStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => clearField(2);
}

class GetStorageStateResponse extends $pb.GeneratedMessage {
  factory GetStorageStateResponse({
    StorageState? state,
    $core.String? requestId,
  }) {
    final $result = create();
    if (state != null) {
      $result.state = state;
    }
    if (requestId != null) {
      $result.requestId = requestId;
    }
    return $result;
  }
  GetStorageStateResponse._() : super();
  factory GetStorageStateResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetStorageStateResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetStorageStateResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOM<StorageState>(1, _omitFieldNames ? '' : 'state', subBuilder: StorageState.create)
    ..aOS(2, _omitFieldNames ? '' : 'requestId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetStorageStateResponse clone() => GetStorageStateResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetStorageStateResponse copyWith(void Function(GetStorageStateResponse) updates) => super.copyWith((message) => updates(message as GetStorageStateResponse)) as GetStorageStateResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetStorageStateResponse create() => GetStorageStateResponse._();
  GetStorageStateResponse createEmptyInstance() => create();
  static $pb.PbList<GetStorageStateResponse> createRepeated() => $pb.PbList<GetStorageStateResponse>();
  @$core.pragma('dart2js:noInline')
  static GetStorageStateResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetStorageStateResponse>(create);
  static GetStorageStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  StorageState get state => $_getN(0);
  @$pb.TagNumber(1)
  set state(StorageState v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasState() => $_has(0);
  @$pb.TagNumber(1)
  void clearState() => clearField(1);
  @$pb.TagNumber(1)
  StorageState ensureState() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get requestId => $_getSZ(1);
  @$pb.TagNumber(2)
  set requestId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => clearField(2);
}

class StorageState extends $pb.GeneratedMessage {
  factory StorageState({
    $fixnum.Int64? storageUsedBytes,
    $fixnum.Int64? totalDurationMs,
    $core.int? videoCount,
    $core.Iterable<VideoRange>? ranges,
    $core.int? storageLimitGb,
    $core.double? deviceStorageRemainingGb,
    $core.String? storagePath,
  }) {
    final $result = create();
    if (storageUsedBytes != null) {
      $result.storageUsedBytes = storageUsedBytes;
    }
    if (totalDurationMs != null) {
      $result.totalDurationMs = totalDurationMs;
    }
    if (videoCount != null) {
      $result.videoCount = videoCount;
    }
    if (ranges != null) {
      $result.ranges.addAll(ranges);
    }
    if (storageLimitGb != null) {
      $result.storageLimitGb = storageLimitGb;
    }
    if (deviceStorageRemainingGb != null) {
      $result.deviceStorageRemainingGb = deviceStorageRemainingGb;
    }
    if (storagePath != null) {
      $result.storagePath = storagePath;
    }
    return $result;
  }
  StorageState._() : super();
  factory StorageState.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory StorageState.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'StorageState', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'storageUsedBytes')
    ..aInt64(2, _omitFieldNames ? '' : 'totalDurationMs')
    ..a<$core.int>(3, _omitFieldNames ? '' : 'videoCount', $pb.PbFieldType.O3)
    ..pc<VideoRange>(4, _omitFieldNames ? '' : 'ranges', $pb.PbFieldType.PM, subBuilder: VideoRange.create)
    ..a<$core.int>(5, _omitFieldNames ? '' : 'storageLimitGb', $pb.PbFieldType.O3)
    ..a<$core.double>(6, _omitFieldNames ? '' : 'deviceStorageRemainingGb', $pb.PbFieldType.OD)
    ..aOS(7, _omitFieldNames ? '' : 'storagePath')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  StorageState clone() => StorageState()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  StorageState copyWith(void Function(StorageState) updates) => super.copyWith((message) => updates(message as StorageState)) as StorageState;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StorageState create() => StorageState._();
  StorageState createEmptyInstance() => create();
  static $pb.PbList<StorageState> createRepeated() => $pb.PbList<StorageState>();
  @$core.pragma('dart2js:noInline')
  static StorageState getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StorageState>(create);
  static StorageState? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get storageUsedBytes => $_getI64(0);
  @$pb.TagNumber(1)
  set storageUsedBytes($fixnum.Int64 v) { $_setInt64(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasStorageUsedBytes() => $_has(0);
  @$pb.TagNumber(1)
  void clearStorageUsedBytes() => clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get totalDurationMs => $_getI64(1);
  @$pb.TagNumber(2)
  set totalDurationMs($fixnum.Int64 v) { $_setInt64(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTotalDurationMs() => $_has(1);
  @$pb.TagNumber(2)
  void clearTotalDurationMs() => clearField(2);

  @$pb.TagNumber(3)
  $core.int get videoCount => $_getIZ(2);
  @$pb.TagNumber(3)
  set videoCount($core.int v) { $_setSignedInt32(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasVideoCount() => $_has(2);
  @$pb.TagNumber(3)
  void clearVideoCount() => clearField(3);

  @$pb.TagNumber(4)
  $core.List<VideoRange> get ranges => $_getList(3);

  @$pb.TagNumber(5)
  $core.int get storageLimitGb => $_getIZ(4);
  @$pb.TagNumber(5)
  set storageLimitGb($core.int v) { $_setSignedInt32(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasStorageLimitGb() => $_has(4);
  @$pb.TagNumber(5)
  void clearStorageLimitGb() => clearField(5);

  @$pb.TagNumber(6)
  $core.double get deviceStorageRemainingGb => $_getN(5);
  @$pb.TagNumber(6)
  set deviceStorageRemainingGb($core.double v) { $_setDouble(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasDeviceStorageRemainingGb() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeviceStorageRemainingGb() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get storagePath => $_getSZ(6);
  @$pb.TagNumber(7)
  set storagePath($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasStoragePath() => $_has(6);
  @$pb.TagNumber(7)
  void clearStoragePath() => clearField(7);
}

class VideoRange extends $pb.GeneratedMessage {
  factory VideoRange({
    $core.String? from,
    $core.String? to,
  }) {
    final $result = create();
    if (from != null) {
      $result.from = from;
    }
    if (to != null) {
      $result.to = to;
    }
    return $result;
  }
  VideoRange._() : super();
  factory VideoRange.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory VideoRange.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'VideoRange', package: const $pb.PackageName(_omitMessageNames ? '' : 'viammodules.service.videostore.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'from')
    ..aOS(2, _omitFieldNames ? '' : 'to')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  VideoRange clone() => VideoRange()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  VideoRange copyWith(void Function(VideoRange) updates) => super.copyWith((message) => updates(message as VideoRange)) as VideoRange;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static VideoRange create() => VideoRange._();
  VideoRange createEmptyInstance() => create();
  static $pb.PbList<VideoRange> createRepeated() => $pb.PbList<VideoRange>();
  @$core.pragma('dart2js:noInline')
  static VideoRange getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<VideoRange>(create);
  static VideoRange? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get from => $_getSZ(0);
  @$pb.TagNumber(1)
  set from($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasFrom() => $_has(0);
  @$pb.TagNumber(1)
  void clearFrom() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get to => $_getSZ(1);
  @$pb.TagNumber(2)
  set to($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasTo() => $_has(1);
  @$pb.TagNumber(2)
  void clearTo() => clearField(2);
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
