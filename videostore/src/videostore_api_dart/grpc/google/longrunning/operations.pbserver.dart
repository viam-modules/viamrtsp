// This is a generated file - do not edit.
//
// Generated from google/longrunning/operations.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import '../protobuf/empty.pb.dart' as $3;
import 'operations.pb.dart' as $4;
import 'operations.pbjson.dart';

export 'operations.pb.dart';

abstract class OperationsServiceBase extends $pb.GeneratedService {
  $async.Future<$4.ListOperationsResponse> listOperations(
      $pb.ServerContext ctx, $4.ListOperationsRequest request);
  $async.Future<$4.Operation> getOperation(
      $pb.ServerContext ctx, $4.GetOperationRequest request);
  $async.Future<$3.Empty> deleteOperation(
      $pb.ServerContext ctx, $4.DeleteOperationRequest request);
  $async.Future<$3.Empty> cancelOperation(
      $pb.ServerContext ctx, $4.CancelOperationRequest request);
  $async.Future<$4.Operation> waitOperation(
      $pb.ServerContext ctx, $4.WaitOperationRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'ListOperations':
        return $4.ListOperationsRequest();
      case 'GetOperation':
        return $4.GetOperationRequest();
      case 'DeleteOperation':
        return $4.DeleteOperationRequest();
      case 'CancelOperation':
        return $4.CancelOperationRequest();
      case 'WaitOperation':
        return $4.WaitOperationRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'ListOperations':
        return listOperations(ctx, request as $4.ListOperationsRequest);
      case 'GetOperation':
        return getOperation(ctx, request as $4.GetOperationRequest);
      case 'DeleteOperation':
        return deleteOperation(ctx, request as $4.DeleteOperationRequest);
      case 'CancelOperation':
        return cancelOperation(ctx, request as $4.CancelOperationRequest);
      case 'WaitOperation':
        return waitOperation(ctx, request as $4.WaitOperationRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      OperationsServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => OperationsServiceBase$messageJson;
}
