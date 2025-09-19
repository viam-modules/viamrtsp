//
//  Generated code. Do not modify.
//  source: google/longrunning/operations.proto
//

import "package:connectrpc/connect.dart" as connect;
import "operations.pb.dart" as googlelongrunningoperations;
import "../protobuf/empty.pb.dart" as googleprotobufempty;

/// Manages long-running operations with an API service.
/// When an API method normally takes long time to complete, it can be designed
/// to return [Operation][google.longrunning.Operation] to the client, and the
/// client can use this interface to receive the real response asynchronously by
/// polling the operation resource, or pass the operation resource to another API
/// (such as Pub/Sub API) to receive the response.  Any API service that returns
/// long-running operations should implement the `Operations` interface so
/// developers can have a consistent client experience.
abstract final class Operations {
  /// Fully-qualified name of the Operations service.
  static const name = 'google.longrunning.Operations';

  /// Lists operations that match the specified filter in the request. If the
  /// server doesn't support this method, it returns `UNIMPLEMENTED`.
  static const listOperations = connect.Spec(
    '/$name/ListOperations',
    connect.StreamType.unary,
    googlelongrunningoperations.ListOperationsRequest.new,
    googlelongrunningoperations.ListOperationsResponse.new,
  );

  /// Gets the latest state of a long-running operation.  Clients can use this
  /// method to poll the operation result at intervals as recommended by the API
  /// service.
  static const getOperation = connect.Spec(
    '/$name/GetOperation',
    connect.StreamType.unary,
    googlelongrunningoperations.GetOperationRequest.new,
    googlelongrunningoperations.Operation.new,
  );

  /// Deletes a long-running operation. This method indicates that the client is
  /// no longer interested in the operation result. It does not cancel the
  /// operation. If the server doesn't support this method, it returns
  /// `google.rpc.Code.UNIMPLEMENTED`.
  static const deleteOperation = connect.Spec(
    '/$name/DeleteOperation',
    connect.StreamType.unary,
    googlelongrunningoperations.DeleteOperationRequest.new,
    googleprotobufempty.Empty.new,
  );

  /// Starts asynchronous cancellation on a long-running operation.  The server
  /// makes a best effort to cancel the operation, but success is not
  /// guaranteed.  If the server doesn't support this method, it returns
  /// `google.rpc.Code.UNIMPLEMENTED`.  Clients can use
  /// [Operations.GetOperation][google.longrunning.Operations.GetOperation] or
  /// other methods to check whether the cancellation succeeded or whether the
  /// operation completed despite cancellation. On successful cancellation,
  /// the operation is not deleted; instead, it becomes an operation with
  /// an [Operation.error][google.longrunning.Operation.error] value with a
  /// [google.rpc.Status.code][google.rpc.Status.code] of `1`, corresponding to
  /// `Code.CANCELLED`.
  static const cancelOperation = connect.Spec(
    '/$name/CancelOperation',
    connect.StreamType.unary,
    googlelongrunningoperations.CancelOperationRequest.new,
    googleprotobufempty.Empty.new,
  );

  /// Waits until the specified long-running operation is done or reaches at most
  /// a specified timeout, returning the latest state.  If the operation is
  /// already done, the latest state is immediately returned.  If the timeout
  /// specified is greater than the default HTTP/RPC timeout, the HTTP/RPC
  /// timeout is used.  If the server does not support this method, it returns
  /// `google.rpc.Code.UNIMPLEMENTED`.
  /// Note that this method is on a best-effort basis.  It may return the latest
  /// state before the specified timeout (including immediately), meaning even an
  /// immediate response is no guarantee that the operation is done.
  static const waitOperation = connect.Spec(
    '/$name/WaitOperation',
    connect.StreamType.unary,
    googlelongrunningoperations.WaitOperationRequest.new,
    googlelongrunningoperations.Operation.new,
  );
}
