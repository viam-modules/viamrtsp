// This is a generated file - do not edit.
//
// Generated from google/api/expr/v1alpha1/syntax.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// CEL component specifier.
class SourceInfo_Extension_Component extends $pb.ProtobufEnum {
  /// Unspecified, default.
  static const SourceInfo_Extension_Component COMPONENT_UNSPECIFIED =
      SourceInfo_Extension_Component._(
          0, _omitEnumNames ? '' : 'COMPONENT_UNSPECIFIED');

  /// Parser. Converts a CEL string to an AST.
  static const SourceInfo_Extension_Component COMPONENT_PARSER =
      SourceInfo_Extension_Component._(
          1, _omitEnumNames ? '' : 'COMPONENT_PARSER');

  /// Type checker. Checks that references in an AST are defined and types
  /// agree.
  static const SourceInfo_Extension_Component COMPONENT_TYPE_CHECKER =
      SourceInfo_Extension_Component._(
          2, _omitEnumNames ? '' : 'COMPONENT_TYPE_CHECKER');

  /// Runtime. Evaluates a parsed and optionally checked CEL AST against a
  /// context.
  static const SourceInfo_Extension_Component COMPONENT_RUNTIME =
      SourceInfo_Extension_Component._(
          3, _omitEnumNames ? '' : 'COMPONENT_RUNTIME');

  static const $core.List<SourceInfo_Extension_Component> values =
      <SourceInfo_Extension_Component>[
    COMPONENT_UNSPECIFIED,
    COMPONENT_PARSER,
    COMPONENT_TYPE_CHECKER,
    COMPONENT_RUNTIME,
  ];

  static final $core.List<SourceInfo_Extension_Component?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static SourceInfo_Extension_Component? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const SourceInfo_Extension_Component._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
