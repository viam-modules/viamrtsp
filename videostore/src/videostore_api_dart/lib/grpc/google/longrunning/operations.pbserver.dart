///
//  Generated code. Do not modify.
//  source: google/longrunning/operations.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,constant_identifier_names,deprecated_member_use_from_same_package,directives_ordering,library_prefixes,non_constant_identifier_names,prefer_final_fields,return_of_invalid_type,unnecessary_const,unnecessary_import,unnecessary_this,unused_import,unused_shown_name

import 'dart:async' as $async;

import 'package:protobuf/protobuf.dart' as $pb;

import 'dart:core' as $core;
import 'operations.pb.dart' as $15;
import '../protobuf/empty.pb.dart' as $6;
import 'operations.pbjson.dart';

export 'operations.pb.dart';

abstract class OperationsServiceBase extends $pb.GeneratedService {
  $async.Future<$15.ListOperationsResponse> listOperations($pb.ServerContext ctx, $15.ListOperationsRequest request);
  $async.Future<$15.Operation> getOperation($pb.ServerContext ctx, $15.GetOperationRequest request);
  $async.Future<$6.Empty> deleteOperation($pb.ServerContext ctx, $15.DeleteOperationRequest request);
  $async.Future<$6.Empty> cancelOperation($pb.ServerContext ctx, $15.CancelOperationRequest request);
  $async.Future<$15.Operation> waitOperation($pb.ServerContext ctx, $15.WaitOperationRequest request);

  $pb.GeneratedMessage createRequest($core.String method) {
    switch (method) {
      case 'ListOperations': return $15.ListOperationsRequest();
      case 'GetOperation': return $15.GetOperationRequest();
      case 'DeleteOperation': return $15.DeleteOperationRequest();
      case 'CancelOperation': return $15.CancelOperationRequest();
      case 'WaitOperation': return $15.WaitOperationRequest();
      default: throw $core.ArgumentError('Unknown method: $method');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String method, $pb.GeneratedMessage request) {
    switch (method) {
      case 'ListOperations': return this.listOperations(ctx, request as $15.ListOperationsRequest);
      case 'GetOperation': return this.getOperation(ctx, request as $15.GetOperationRequest);
      case 'DeleteOperation': return this.deleteOperation(ctx, request as $15.DeleteOperationRequest);
      case 'CancelOperation': return this.cancelOperation(ctx, request as $15.CancelOperationRequest);
      case 'WaitOperation': return this.waitOperation(ctx, request as $15.WaitOperationRequest);
      default: throw $core.ArgumentError('Unknown method: $method');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => OperationsServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => OperationsServiceBase$messageJson;
}

