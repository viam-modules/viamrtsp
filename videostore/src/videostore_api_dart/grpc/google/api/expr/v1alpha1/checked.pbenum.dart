// This is a generated file - do not edit.
//
// Generated from google/api/expr/v1alpha1/checked.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// CEL primitive types.
class Type_PrimitiveType extends $pb.ProtobufEnum {
  /// Unspecified type.
  static const Type_PrimitiveType PRIMITIVE_TYPE_UNSPECIFIED =
      Type_PrimitiveType._(
          0, _omitEnumNames ? '' : 'PRIMITIVE_TYPE_UNSPECIFIED');

  /// Boolean type.
  static const Type_PrimitiveType BOOL =
      Type_PrimitiveType._(1, _omitEnumNames ? '' : 'BOOL');

  /// Int64 type.
  ///
  /// Proto-based integer values are widened to int64.
  static const Type_PrimitiveType INT64 =
      Type_PrimitiveType._(2, _omitEnumNames ? '' : 'INT64');

  /// Uint64 type.
  ///
  /// Proto-based unsigned integer values are widened to uint64.
  static const Type_PrimitiveType UINT64 =
      Type_PrimitiveType._(3, _omitEnumNames ? '' : 'UINT64');

  /// Double type.
  ///
  /// Proto-based float values are widened to double values.
  static const Type_PrimitiveType DOUBLE =
      Type_PrimitiveType._(4, _omitEnumNames ? '' : 'DOUBLE');

  /// String type.
  static const Type_PrimitiveType STRING =
      Type_PrimitiveType._(5, _omitEnumNames ? '' : 'STRING');

  /// Bytes type.
  static const Type_PrimitiveType BYTES =
      Type_PrimitiveType._(6, _omitEnumNames ? '' : 'BYTES');

  static const $core.List<Type_PrimitiveType> values = <Type_PrimitiveType>[
    PRIMITIVE_TYPE_UNSPECIFIED,
    BOOL,
    INT64,
    UINT64,
    DOUBLE,
    STRING,
    BYTES,
  ];

  static final $core.List<Type_PrimitiveType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static Type_PrimitiveType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const Type_PrimitiveType._(super.value, super.name);
}

/// Well-known protobuf types treated with first-class support in CEL.
class Type_WellKnownType extends $pb.ProtobufEnum {
  /// Unspecified type.
  static const Type_WellKnownType WELL_KNOWN_TYPE_UNSPECIFIED =
      Type_WellKnownType._(
          0, _omitEnumNames ? '' : 'WELL_KNOWN_TYPE_UNSPECIFIED');

  /// Well-known protobuf.Any type.
  ///
  /// Any types are a polymorphic message type. During type-checking they are
  /// treated like `DYN` types, but at runtime they are resolved to a specific
  /// message type specified at evaluation time.
  static const Type_WellKnownType ANY =
      Type_WellKnownType._(1, _omitEnumNames ? '' : 'ANY');

  /// Well-known protobuf.Timestamp type, internally referenced as `timestamp`.
  static const Type_WellKnownType TIMESTAMP =
      Type_WellKnownType._(2, _omitEnumNames ? '' : 'TIMESTAMP');

  /// Well-known protobuf.Duration type, internally referenced as `duration`.
  static const Type_WellKnownType DURATION =
      Type_WellKnownType._(3, _omitEnumNames ? '' : 'DURATION');

  static const $core.List<Type_WellKnownType> values = <Type_WellKnownType>[
    WELL_KNOWN_TYPE_UNSPECIFIED,
    ANY,
    TIMESTAMP,
    DURATION,
  ];

  static final $core.List<Type_WellKnownType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static Type_WellKnownType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const Type_WellKnownType._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
