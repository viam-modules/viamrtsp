// This is a generated file - do not edit.
//
// Generated from google/api/expr/v1beta1/decl.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'expr.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

enum Decl_Kind { ident, function, notSet }

/// A declaration.
class Decl extends $pb.GeneratedMessage {
  factory Decl({
    $core.int? id,
    $core.String? name,
    $core.String? doc,
    IdentDecl? ident,
    FunctionDecl? function,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (doc != null) result.doc = doc;
    if (ident != null) result.ident = ident;
    if (function != null) result.function = function;
    return result;
  }

  Decl._();

  factory Decl.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Decl.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, Decl_Kind> _Decl_KindByTag = {
    4: Decl_Kind.ident,
    5: Decl_Kind.function,
    0: Decl_Kind.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Decl',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'google.api.expr.v1beta1'),
      createEmptyInstance: create)
    ..oo(0, [4, 5])
    ..a<$core.int>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.O3)
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'doc')
    ..aOM<IdentDecl>(4, _omitFieldNames ? '' : 'ident',
        subBuilder: IdentDecl.create)
    ..aOM<FunctionDecl>(5, _omitFieldNames ? '' : 'function',
        subBuilder: FunctionDecl.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Decl clone() => Decl()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Decl copyWith(void Function(Decl) updates) =>
      super.copyWith((message) => updates(message as Decl)) as Decl;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Decl create() => Decl._();
  @$core.override
  Decl createEmptyInstance() => create();
  static $pb.PbList<Decl> createRepeated() => $pb.PbList<Decl>();
  @$core.pragma('dart2js:noInline')
  static Decl getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Decl>(create);
  static Decl? _defaultInstance;

  Decl_Kind whichKind() => _Decl_KindByTag[$_whichOneof(0)]!;
  void clearKind() => $_clearField($_whichOneof(0));

  /// The id of the declaration.
  @$pb.TagNumber(1)
  $core.int get id => $_getIZ(0);
  @$pb.TagNumber(1)
  set id($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  /// The name of the declaration.
  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  /// The documentation string for the declaration.
  @$pb.TagNumber(3)
  $core.String get doc => $_getSZ(2);
  @$pb.TagNumber(3)
  set doc($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDoc() => $_has(2);
  @$pb.TagNumber(3)
  void clearDoc() => $_clearField(3);

  /// An identifier declaration.
  @$pb.TagNumber(4)
  IdentDecl get ident => $_getN(3);
  @$pb.TagNumber(4)
  set ident(IdentDecl value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasIdent() => $_has(3);
  @$pb.TagNumber(4)
  void clearIdent() => $_clearField(4);
  @$pb.TagNumber(4)
  IdentDecl ensureIdent() => $_ensure(3);

  /// A function declaration.
  @$pb.TagNumber(5)
  FunctionDecl get function => $_getN(4);
  @$pb.TagNumber(5)
  set function(FunctionDecl value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasFunction() => $_has(4);
  @$pb.TagNumber(5)
  void clearFunction() => $_clearField(5);
  @$pb.TagNumber(5)
  FunctionDecl ensureFunction() => $_ensure(4);
}

/// The declared type of a variable.
///
/// Extends runtime type values with extra information used for type checking
/// and dispatching.
class DeclType extends $pb.GeneratedMessage {
  factory DeclType({
    $core.int? id,
    $core.String? type,
    $core.Iterable<DeclType>? typeParams,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (type != null) result.type = type;
    if (typeParams != null) result.typeParams.addAll(typeParams);
    return result;
  }

  DeclType._();

  factory DeclType.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeclType.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeclType',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'google.api.expr.v1beta1'),
      createEmptyInstance: create)
    ..a<$core.int>(1, _omitFieldNames ? '' : 'id', $pb.PbFieldType.O3)
    ..aOS(2, _omitFieldNames ? '' : 'type')
    ..pc<DeclType>(4, _omitFieldNames ? '' : 'typeParams', $pb.PbFieldType.PM,
        subBuilder: DeclType.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeclType clone() => DeclType()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeclType copyWith(void Function(DeclType) updates) =>
      super.copyWith((message) => updates(message as DeclType)) as DeclType;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeclType create() => DeclType._();
  @$core.override
  DeclType createEmptyInstance() => create();
  static $pb.PbList<DeclType> createRepeated() => $pb.PbList<DeclType>();
  @$core.pragma('dart2js:noInline')
  static DeclType getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DeclType>(create);
  static DeclType? _defaultInstance;

  /// The expression id of the declared type, if applicable.
  @$pb.TagNumber(1)
  $core.int get id => $_getIZ(0);
  @$pb.TagNumber(1)
  set id($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  /// The type name, e.g. 'int', 'my.type.Type' or 'T'
  @$pb.TagNumber(2)
  $core.String get type => $_getSZ(1);
  @$pb.TagNumber(2)
  set type($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);

  /// An ordered list of type parameters, e.g. `<string, int>`.
  /// Only applies to a subset of types, e.g. `map`, `list`.
  @$pb.TagNumber(4)
  $pb.PbList<DeclType> get typeParams => $_getList(2);
}

/// An identifier declaration.
class IdentDecl extends $pb.GeneratedMessage {
  factory IdentDecl({
    DeclType? type,
    $0.Expr? value,
  }) {
    final result = create();
    if (type != null) result.type = type;
    if (value != null) result.value = value;
    return result;
  }

  IdentDecl._();

  factory IdentDecl.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory IdentDecl.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IdentDecl',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'google.api.expr.v1beta1'),
      createEmptyInstance: create)
    ..aOM<DeclType>(3, _omitFieldNames ? '' : 'type',
        subBuilder: DeclType.create)
    ..aOM<$0.Expr>(4, _omitFieldNames ? '' : 'value',
        subBuilder: $0.Expr.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IdentDecl clone() => IdentDecl()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IdentDecl copyWith(void Function(IdentDecl) updates) =>
      super.copyWith((message) => updates(message as IdentDecl)) as IdentDecl;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static IdentDecl create() => IdentDecl._();
  @$core.override
  IdentDecl createEmptyInstance() => create();
  static $pb.PbList<IdentDecl> createRepeated() => $pb.PbList<IdentDecl>();
  @$core.pragma('dart2js:noInline')
  static IdentDecl getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<IdentDecl>(create);
  static IdentDecl? _defaultInstance;

  /// Optional type of the identifier.
  @$pb.TagNumber(3)
  DeclType get type => $_getN(0);
  @$pb.TagNumber(3)
  set type(DeclType value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(3)
  void clearType() => $_clearField(3);
  @$pb.TagNumber(3)
  DeclType ensureType() => $_ensure(0);

  /// Optional value of the identifier.
  @$pb.TagNumber(4)
  $0.Expr get value => $_getN(1);
  @$pb.TagNumber(4)
  set value($0.Expr value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(4)
  void clearValue() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Expr ensureValue() => $_ensure(1);
}

/// A function declaration.
class FunctionDecl extends $pb.GeneratedMessage {
  factory FunctionDecl({
    $core.Iterable<IdentDecl>? args,
    DeclType? returnType,
    $core.bool? receiverFunction,
  }) {
    final result = create();
    if (args != null) result.args.addAll(args);
    if (returnType != null) result.returnType = returnType;
    if (receiverFunction != null) result.receiverFunction = receiverFunction;
    return result;
  }

  FunctionDecl._();

  factory FunctionDecl.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FunctionDecl.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FunctionDecl',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'google.api.expr.v1beta1'),
      createEmptyInstance: create)
    ..pc<IdentDecl>(1, _omitFieldNames ? '' : 'args', $pb.PbFieldType.PM,
        subBuilder: IdentDecl.create)
    ..aOM<DeclType>(2, _omitFieldNames ? '' : 'returnType',
        subBuilder: DeclType.create)
    ..aOB(3, _omitFieldNames ? '' : 'receiverFunction')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FunctionDecl clone() => FunctionDecl()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FunctionDecl copyWith(void Function(FunctionDecl) updates) =>
      super.copyWith((message) => updates(message as FunctionDecl))
          as FunctionDecl;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FunctionDecl create() => FunctionDecl._();
  @$core.override
  FunctionDecl createEmptyInstance() => create();
  static $pb.PbList<FunctionDecl> createRepeated() =>
      $pb.PbList<FunctionDecl>();
  @$core.pragma('dart2js:noInline')
  static FunctionDecl getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FunctionDecl>(create);
  static FunctionDecl? _defaultInstance;

  /// The function arguments.
  @$pb.TagNumber(1)
  $pb.PbList<IdentDecl> get args => $_getList(0);

  /// Optional declared return type.
  @$pb.TagNumber(2)
  DeclType get returnType => $_getN(1);
  @$pb.TagNumber(2)
  set returnType(DeclType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasReturnType() => $_has(1);
  @$pb.TagNumber(2)
  void clearReturnType() => $_clearField(2);
  @$pb.TagNumber(2)
  DeclType ensureReturnType() => $_ensure(1);

  /// If the first argument of the function is the receiver.
  @$pb.TagNumber(3)
  $core.bool get receiverFunction => $_getBF(2);
  @$pb.TagNumber(3)
  set receiverFunction($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasReceiverFunction() => $_has(2);
  @$pb.TagNumber(3)
  void clearReceiverFunction() => $_clearField(3);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
