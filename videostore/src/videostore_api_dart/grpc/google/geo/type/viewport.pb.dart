// This is a generated file - do not edit.
//
// Generated from google/geo/type/viewport.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import '../../type/latlng.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// A latitude-longitude viewport, represented as two diagonally opposite `low`
/// and `high` points. A viewport is considered a closed region, i.e. it includes
/// its boundary. The latitude bounds must range between -90 to 90 degrees
/// inclusive, and the longitude bounds must range between -180 to 180 degrees
/// inclusive. Various cases include:
///
///  - If `low` = `high`, the viewport consists of that single point.
///
///  - If `low.longitude` > `high.longitude`, the longitude range is inverted
///    (the viewport crosses the 180 degree longitude line).
///
///  - If `low.longitude` = -180 degrees and `high.longitude` = 180 degrees,
///    the viewport includes all longitudes.
///
///  - If `low.longitude` = 180 degrees and `high.longitude` = -180 degrees,
///    the longitude range is empty.
///
///  - If `low.latitude` > `high.latitude`, the latitude range is empty.
///
/// Both `low` and `high` must be populated, and the represented box cannot be
/// empty (as specified by the definitions above). An empty viewport will result
/// in an error.
///
/// For example, this viewport fully encloses New York City:
///
/// {
///     "low": {
///         "latitude": 40.477398,
///         "longitude": -74.259087
///     },
///     "high": {
///         "latitude": 40.91618,
///         "longitude": -73.70018
///     }
/// }
class Viewport extends $pb.GeneratedMessage {
  factory Viewport({
    $0.LatLng? low,
    $0.LatLng? high,
  }) {
    final result = create();
    if (low != null) result.low = low;
    if (high != null) result.high = high;
    return result;
  }

  Viewport._();

  factory Viewport.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Viewport.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Viewport',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'google.geo.type'),
      createEmptyInstance: create)
    ..aOM<$0.LatLng>(1, _omitFieldNames ? '' : 'low',
        subBuilder: $0.LatLng.create)
    ..aOM<$0.LatLng>(2, _omitFieldNames ? '' : 'high',
        subBuilder: $0.LatLng.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Viewport clone() => Viewport()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Viewport copyWith(void Function(Viewport) updates) =>
      super.copyWith((message) => updates(message as Viewport)) as Viewport;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Viewport create() => Viewport._();
  @$core.override
  Viewport createEmptyInstance() => create();
  static $pb.PbList<Viewport> createRepeated() => $pb.PbList<Viewport>();
  @$core.pragma('dart2js:noInline')
  static Viewport getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Viewport>(create);
  static Viewport? _defaultInstance;

  /// Required. The low point of the viewport.
  @$pb.TagNumber(1)
  $0.LatLng get low => $_getN(0);
  @$pb.TagNumber(1)
  set low($0.LatLng value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasLow() => $_has(0);
  @$pb.TagNumber(1)
  void clearLow() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.LatLng ensureLow() => $_ensure(0);

  /// Required. The high point of the viewport.
  @$pb.TagNumber(2)
  $0.LatLng get high => $_getN(1);
  @$pb.TagNumber(2)
  set high($0.LatLng value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasHigh() => $_has(1);
  @$pb.TagNumber(2)
  void clearHigh() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.LatLng ensureHigh() => $_ensure(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
