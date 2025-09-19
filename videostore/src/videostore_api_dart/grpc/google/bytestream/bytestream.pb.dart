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

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// Request object for ByteStream.Read.
class ReadRequest extends $pb.GeneratedMessage {
  factory ReadRequest({
    $core.String? resourceName,
    $fixnum.Int64? readOffset,
    $fixnum.Int64? readLimit,
  }) {
    final result = create();
    if (resourceName != null) result.resourceName = resourceName;
    if (readOffset != null) result.readOffset = readOffset;
    if (readLimit != null) result.readLimit = readLimit;
    return result;
  }

  ReadRequest._();

  factory ReadRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReadRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReadRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.bytestream'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'resourceName')
    ..aInt64(2, _omitFieldNames ? '' : 'readOffset')
    ..aInt64(3, _omitFieldNames ? '' : 'readLimit')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadRequest clone() => ReadRequest()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadRequest copyWith(void Function(ReadRequest) updates) =>
      super.copyWith((message) => updates(message as ReadRequest))
          as ReadRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReadRequest create() => ReadRequest._();
  @$core.override
  ReadRequest createEmptyInstance() => create();
  static $pb.PbList<ReadRequest> createRepeated() => $pb.PbList<ReadRequest>();
  @$core.pragma('dart2js:noInline')
  static ReadRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReadRequest>(create);
  static ReadRequest? _defaultInstance;

  /// The name of the resource to read.
  @$pb.TagNumber(1)
  $core.String get resourceName => $_getSZ(0);
  @$pb.TagNumber(1)
  set resourceName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasResourceName() => $_has(0);
  @$pb.TagNumber(1)
  void clearResourceName() => $_clearField(1);

  /// The offset for the first byte to return in the read, relative to the start
  /// of the resource.
  ///
  /// A `read_offset` that is negative or greater than the size of the resource
  /// will cause an `OUT_OF_RANGE` error.
  @$pb.TagNumber(2)
  $fixnum.Int64 get readOffset => $_getI64(1);
  @$pb.TagNumber(2)
  set readOffset($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReadOffset() => $_has(1);
  @$pb.TagNumber(2)
  void clearReadOffset() => $_clearField(2);

  /// The maximum number of `data` bytes the server is allowed to return in the
  /// sum of all `ReadResponse` messages. A `read_limit` of zero indicates that
  /// there is no limit, and a negative `read_limit` will cause an error.
  ///
  /// If the stream returns fewer bytes than allowed by the `read_limit` and no
  /// error occurred, the stream includes all data from the `read_offset` to the
  /// end of the resource.
  @$pb.TagNumber(3)
  $fixnum.Int64 get readLimit => $_getI64(2);
  @$pb.TagNumber(3)
  set readLimit($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReadLimit() => $_has(2);
  @$pb.TagNumber(3)
  void clearReadLimit() => $_clearField(3);
}

/// Response object for ByteStream.Read.
class ReadResponse extends $pb.GeneratedMessage {
  factory ReadResponse({
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (data != null) result.data = data;
    return result;
  }

  ReadResponse._();

  factory ReadResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReadResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReadResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.bytestream'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        10, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadResponse clone() => ReadResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadResponse copyWith(void Function(ReadResponse) updates) =>
      super.copyWith((message) => updates(message as ReadResponse))
          as ReadResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReadResponse create() => ReadResponse._();
  @$core.override
  ReadResponse createEmptyInstance() => create();
  static $pb.PbList<ReadResponse> createRepeated() =>
      $pb.PbList<ReadResponse>();
  @$core.pragma('dart2js:noInline')
  static ReadResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReadResponse>(create);
  static ReadResponse? _defaultInstance;

  /// A portion of the data for the resource. The service **may** leave `data`
  /// empty for any given `ReadResponse`. This enables the service to inform the
  /// client that the request is still live while it is running an operation to
  /// generate more data.
  @$pb.TagNumber(10)
  $core.List<$core.int> get data => $_getN(0);
  @$pb.TagNumber(10)
  set data($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(10)
  $core.bool hasData() => $_has(0);
  @$pb.TagNumber(10)
  void clearData() => $_clearField(10);
}

/// Request object for ByteStream.Write.
class WriteRequest extends $pb.GeneratedMessage {
  factory WriteRequest({
    $core.String? resourceName,
    $fixnum.Int64? writeOffset,
    $core.bool? finishWrite,
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (resourceName != null) result.resourceName = resourceName;
    if (writeOffset != null) result.writeOffset = writeOffset;
    if (finishWrite != null) result.finishWrite = finishWrite;
    if (data != null) result.data = data;
    return result;
  }

  WriteRequest._();

  factory WriteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WriteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WriteRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.bytestream'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'resourceName')
    ..aInt64(2, _omitFieldNames ? '' : 'writeOffset')
    ..aOB(3, _omitFieldNames ? '' : 'finishWrite')
    ..a<$core.List<$core.int>>(
        10, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteRequest clone() => WriteRequest()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteRequest copyWith(void Function(WriteRequest) updates) =>
      super.copyWith((message) => updates(message as WriteRequest))
          as WriteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WriteRequest create() => WriteRequest._();
  @$core.override
  WriteRequest createEmptyInstance() => create();
  static $pb.PbList<WriteRequest> createRepeated() =>
      $pb.PbList<WriteRequest>();
  @$core.pragma('dart2js:noInline')
  static WriteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WriteRequest>(create);
  static WriteRequest? _defaultInstance;

  /// The name of the resource to write. This **must** be set on the first
  /// `WriteRequest` of each `Write()` action. If it is set on subsequent calls,
  /// it **must** match the value of the first request.
  @$pb.TagNumber(1)
  $core.String get resourceName => $_getSZ(0);
  @$pb.TagNumber(1)
  set resourceName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasResourceName() => $_has(0);
  @$pb.TagNumber(1)
  void clearResourceName() => $_clearField(1);

  /// The offset from the beginning of the resource at which the data should be
  /// written. It is required on all `WriteRequest`s.
  ///
  /// In the first `WriteRequest` of a `Write()` action, it indicates
  /// the initial offset for the `Write()` call. The value **must** be equal to
  /// the `committed_size` that a call to `QueryWriteStatus()` would return.
  ///
  /// On subsequent calls, this value **must** be set and **must** be equal to
  /// the sum of the first `write_offset` and the sizes of all `data` bundles
  /// sent previously on this stream.
  ///
  /// An incorrect value will cause an error.
  @$pb.TagNumber(2)
  $fixnum.Int64 get writeOffset => $_getI64(1);
  @$pb.TagNumber(2)
  set writeOffset($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWriteOffset() => $_has(1);
  @$pb.TagNumber(2)
  void clearWriteOffset() => $_clearField(2);

  /// If `true`, this indicates that the write is complete. Sending any
  /// `WriteRequest`s subsequent to one in which `finish_write` is `true` will
  /// cause an error.
  @$pb.TagNumber(3)
  $core.bool get finishWrite => $_getBF(2);
  @$pb.TagNumber(3)
  set finishWrite($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFinishWrite() => $_has(2);
  @$pb.TagNumber(3)
  void clearFinishWrite() => $_clearField(3);

  /// A portion of the data for the resource. The client **may** leave `data`
  /// empty for any given `WriteRequest`. This enables the client to inform the
  /// service that the request is still live while it is running an operation to
  /// generate more data.
  @$pb.TagNumber(10)
  $core.List<$core.int> get data => $_getN(3);
  @$pb.TagNumber(10)
  set data($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(10)
  $core.bool hasData() => $_has(3);
  @$pb.TagNumber(10)
  void clearData() => $_clearField(10);
}

/// Response object for ByteStream.Write.
class WriteResponse extends $pb.GeneratedMessage {
  factory WriteResponse({
    $fixnum.Int64? committedSize,
  }) {
    final result = create();
    if (committedSize != null) result.committedSize = committedSize;
    return result;
  }

  WriteResponse._();

  factory WriteResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WriteResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WriteResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.bytestream'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'committedSize')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteResponse clone() => WriteResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteResponse copyWith(void Function(WriteResponse) updates) =>
      super.copyWith((message) => updates(message as WriteResponse))
          as WriteResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WriteResponse create() => WriteResponse._();
  @$core.override
  WriteResponse createEmptyInstance() => create();
  static $pb.PbList<WriteResponse> createRepeated() =>
      $pb.PbList<WriteResponse>();
  @$core.pragma('dart2js:noInline')
  static WriteResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WriteResponse>(create);
  static WriteResponse? _defaultInstance;

  /// The number of bytes that have been processed for the given resource.
  @$pb.TagNumber(1)
  $fixnum.Int64 get committedSize => $_getI64(0);
  @$pb.TagNumber(1)
  set committedSize($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommittedSize() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommittedSize() => $_clearField(1);
}

/// Request object for ByteStream.QueryWriteStatus.
class QueryWriteStatusRequest extends $pb.GeneratedMessage {
  factory QueryWriteStatusRequest({
    $core.String? resourceName,
  }) {
    final result = create();
    if (resourceName != null) result.resourceName = resourceName;
    return result;
  }

  QueryWriteStatusRequest._();

  factory QueryWriteStatusRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory QueryWriteStatusRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'QueryWriteStatusRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.bytestream'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'resourceName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  QueryWriteStatusRequest clone() =>
      QueryWriteStatusRequest()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  QueryWriteStatusRequest copyWith(
          void Function(QueryWriteStatusRequest) updates) =>
      super.copyWith((message) => updates(message as QueryWriteStatusRequest))
          as QueryWriteStatusRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static QueryWriteStatusRequest create() => QueryWriteStatusRequest._();
  @$core.override
  QueryWriteStatusRequest createEmptyInstance() => create();
  static $pb.PbList<QueryWriteStatusRequest> createRepeated() =>
      $pb.PbList<QueryWriteStatusRequest>();
  @$core.pragma('dart2js:noInline')
  static QueryWriteStatusRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<QueryWriteStatusRequest>(create);
  static QueryWriteStatusRequest? _defaultInstance;

  /// The name of the resource whose write status is being requested.
  @$pb.TagNumber(1)
  $core.String get resourceName => $_getSZ(0);
  @$pb.TagNumber(1)
  set resourceName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasResourceName() => $_has(0);
  @$pb.TagNumber(1)
  void clearResourceName() => $_clearField(1);
}

/// Response object for ByteStream.QueryWriteStatus.
class QueryWriteStatusResponse extends $pb.GeneratedMessage {
  factory QueryWriteStatusResponse({
    $fixnum.Int64? committedSize,
    $core.bool? complete,
  }) {
    final result = create();
    if (committedSize != null) result.committedSize = committedSize;
    if (complete != null) result.complete = complete;
    return result;
  }

  QueryWriteStatusResponse._();

  factory QueryWriteStatusResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory QueryWriteStatusResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'QueryWriteStatusResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.bytestream'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'committedSize')
    ..aOB(2, _omitFieldNames ? '' : 'complete')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  QueryWriteStatusResponse clone() =>
      QueryWriteStatusResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  QueryWriteStatusResponse copyWith(
          void Function(QueryWriteStatusResponse) updates) =>
      super.copyWith((message) => updates(message as QueryWriteStatusResponse))
          as QueryWriteStatusResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static QueryWriteStatusResponse create() => QueryWriteStatusResponse._();
  @$core.override
  QueryWriteStatusResponse createEmptyInstance() => create();
  static $pb.PbList<QueryWriteStatusResponse> createRepeated() =>
      $pb.PbList<QueryWriteStatusResponse>();
  @$core.pragma('dart2js:noInline')
  static QueryWriteStatusResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<QueryWriteStatusResponse>(create);
  static QueryWriteStatusResponse? _defaultInstance;

  /// The number of bytes that have been processed for the given resource.
  @$pb.TagNumber(1)
  $fixnum.Int64 get committedSize => $_getI64(0);
  @$pb.TagNumber(1)
  set committedSize($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommittedSize() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommittedSize() => $_clearField(1);

  /// `complete` is `true` only if the client has sent a `WriteRequest` with
  /// `finish_write` set to true, and the server has processed that request.
  @$pb.TagNumber(2)
  $core.bool get complete => $_getBF(1);
  @$pb.TagNumber(2)
  set complete($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasComplete() => $_has(1);
  @$pb.TagNumber(2)
  void clearComplete() => $_clearField(2);
}

/// #### Introduction
///
/// The Byte Stream API enables a client to read and write a stream of bytes to
/// and from a resource. Resources have names, and these names are supplied in
/// the API calls below to identify the resource that is being read from or
/// written to.
///
/// All implementations of the Byte Stream API export the interface defined here:
///
/// * `Read()`: Reads the contents of a resource.
///
/// * `Write()`: Writes the contents of a resource. The client can call `Write()`
///   multiple times with the same resource and can check the status of the write
///   by calling `QueryWriteStatus()`.
///
/// #### Service parameters and metadata
///
/// The ByteStream API provides no direct way to access/modify any metadata
/// associated with the resource.
///
/// #### Errors
///
/// The errors returned by the service are in the Google canonical error space.
class ByteStreamApi {
  final $pb.RpcClient _client;

  ByteStreamApi(this._client);

  /// `Read()` is used to retrieve the contents of a resource as a sequence
  /// of bytes. The bytes are returned in a sequence of responses, and the
  /// responses are delivered as the results of a server-side streaming RPC.
  $async.Future<ReadResponse> read(
          $pb.ClientContext? ctx, ReadRequest request) =>
      _client.invoke<ReadResponse>(
          ctx, 'ByteStream', 'Read', request, ReadResponse());

  /// `Write()` is used to send the contents of a resource as a sequence of
  /// bytes. The bytes are sent in a sequence of request protos of a client-side
  /// streaming RPC.
  ///
  /// A `Write()` action is resumable. If there is an error or the connection is
  /// broken during the `Write()`, the client should check the status of the
  /// `Write()` by calling `QueryWriteStatus()` and continue writing from the
  /// returned `committed_size`. This may be less than the amount of data the
  /// client previously sent.
  ///
  /// Calling `Write()` on a resource name that was previously written and
  /// finalized could cause an error, depending on whether the underlying service
  /// allows over-writing of previously written resources.
  ///
  /// When the client closes the request channel, the service will respond with
  /// a `WriteResponse`. The service will not view the resource as `complete`
  /// until the client has sent a `WriteRequest` with `finish_write` set to
  /// `true`. Sending any requests on a stream after sending a request with
  /// `finish_write` set to `true` will cause an error. The client **should**
  /// check the `WriteResponse` it receives to determine how much data the
  /// service was able to commit and whether the service views the resource as
  /// `complete` or not.
  $async.Future<WriteResponse> write(
          $pb.ClientContext? ctx, WriteRequest request) =>
      _client.invoke<WriteResponse>(
          ctx, 'ByteStream', 'Write', request, WriteResponse());

  /// `QueryWriteStatus()` is used to find the `committed_size` for a resource
  /// that is being written, which can then be used as the `write_offset` for
  /// the next `Write()` call.
  ///
  /// If the resource does not exist (i.e., the resource has been deleted, or the
  /// first `Write()` has not yet reached the service), this method returns the
  /// error `NOT_FOUND`.
  ///
  /// The client **may** call `QueryWriteStatus()` at any time to determine how
  /// much data has been processed for this resource. This is useful if the
  /// client is buffering data and needs to know which data can be safely
  /// evicted. For any sequence of `QueryWriteStatus()` calls for a given
  /// resource name, the sequence of returned `committed_size` values will be
  /// non-decreasing.
  $async.Future<QueryWriteStatusResponse> queryWriteStatus(
          $pb.ClientContext? ctx, QueryWriteStatusRequest request) =>
      _client.invoke<QueryWriteStatusResponse>(ctx, 'ByteStream',
          'QueryWriteStatus', request, QueryWriteStatusResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
