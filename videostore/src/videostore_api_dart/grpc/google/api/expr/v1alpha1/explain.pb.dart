// This is a generated file - do not edit.
//
// Generated from google/api/expr/v1alpha1/explain.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'value.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// ID and value index of one step.
class Explain_ExprStep extends $pb.GeneratedMessage {
  factory Explain_ExprStep({
    $fixnum.Int64? id,
    $core.int? valueIndex,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (valueIndex != null) result.valueIndex = valueIndex;
    return result;
  }

  Explain_ExprStep._();

  factory Explain_ExprStep.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Explain_ExprStep.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Explain.ExprStep',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'google.api.expr.v1alpha1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'id')
    ..a<$core.int>(2, _omitFieldNames ? '' : 'valueIndex', $pb.PbFieldType.O3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Explain_ExprStep clone() => Explain_ExprStep()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Explain_ExprStep copyWith(void Function(Explain_ExprStep) updates) =>
      super.copyWith((message) => updates(message as Explain_ExprStep))
          as Explain_ExprStep;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Explain_ExprStep create() => Explain_ExprStep._();
  @$core.override
  Explain_ExprStep createEmptyInstance() => create();
  static $pb.PbList<Explain_ExprStep> createRepeated() =>
      $pb.PbList<Explain_ExprStep>();
  @$core.pragma('dart2js:noInline')
  static Explain_ExprStep getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Explain_ExprStep>(create);
  static Explain_ExprStep? _defaultInstance;

  /// ID of corresponding Expr node.
  @$pb.TagNumber(1)
  $fixnum.Int64 get id => $_getI64(0);
  @$pb.TagNumber(1)
  set id($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  /// Index of the value in the values list.
  @$pb.TagNumber(2)
  $core.int get valueIndex => $_getIZ(1);
  @$pb.TagNumber(2)
  set valueIndex($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasValueIndex() => $_has(1);
  @$pb.TagNumber(2)
  void clearValueIndex() => $_clearField(2);
}

/// Values of intermediate expressions produced when evaluating expression.
/// Deprecated, use `EvalState` instead.
@$core.Deprecated('This message is deprecated')
class Explain extends $pb.GeneratedMessage {
  factory Explain({
    $core.Iterable<$0.Value>? values,
    $core.Iterable<Explain_ExprStep>? exprSteps,
  }) {
    final result = create();
    if (values != null) result.values.addAll(values);
    if (exprSteps != null) result.exprSteps.addAll(exprSteps);
    return result;
  }

  Explain._();

  factory Explain.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Explain.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Explain',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'google.api.expr.v1alpha1'),
      createEmptyInstance: create)
    ..pc<$0.Value>(1, _omitFieldNames ? '' : 'values', $pb.PbFieldType.PM,
        subBuilder: $0.Value.create)
    ..pc<Explain_ExprStep>(
        2, _omitFieldNames ? '' : 'exprSteps', $pb.PbFieldType.PM,
        subBuilder: Explain_ExprStep.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Explain clone() => Explain()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Explain copyWith(void Function(Explain) updates) =>
      super.copyWith((message) => updates(message as Explain)) as Explain;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Explain create() => Explain._();
  @$core.override
  Explain createEmptyInstance() => create();
  static $pb.PbList<Explain> createRepeated() => $pb.PbList<Explain>();
  @$core.pragma('dart2js:noInline')
  static Explain getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Explain>(create);
  static Explain? _defaultInstance;

  /// All of the observed values.
  ///
  /// The field value_index is an index in the values list.
  /// Separating values from steps is needed to remove redundant values.
  @$pb.TagNumber(1)
  $pb.PbList<$0.Value> get values => $_getList(0);

  /// List of steps.
  ///
  /// Repeated evaluations of the same expression generate new ExprStep
  /// instances. The order of such ExprStep instances matches the order of
  /// elements returned by Comprehension.iter_range.
  @$pb.TagNumber(2)
  $pb.PbList<Explain_ExprStep> get exprSteps => $_getList(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
