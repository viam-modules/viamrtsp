// This is a generated file - do not edit.
//
// Generated from google/type/datetime.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import '../protobuf/duration.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

enum DateTime_TimeOffset { utcOffset, timeZone, notSet }

/// Represents civil time (or occasionally physical time).
///
/// This type can represent a civil time in one of a few possible ways:
///
///  * When utc_offset is set and time_zone is unset: a civil time on a calendar
///    day with a particular offset from UTC.
///  * When time_zone is set and utc_offset is unset: a civil time on a calendar
///    day in a particular time zone.
///  * When neither time_zone nor utc_offset is set: a civil time on a calendar
///    day in local time.
///
/// The date is relative to the Proleptic Gregorian Calendar.
///
/// If year is 0, the DateTime is considered not to have a specific year. month
/// and day must have valid, non-zero values.
///
/// This type may also be used to represent a physical time if all the date and
/// time fields are set and either case of the `time_offset` oneof is set.
/// Consider using `Timestamp` message for physical time instead. If your use
/// case also would like to store the user's timezone, that can be done in
/// another field.
///
/// This type is more flexible than some applications may want. Make sure to
/// document and validate your application's limitations.
class DateTime extends $pb.GeneratedMessage {
  factory DateTime({
    $core.int? year,
    $core.int? month,
    $core.int? day,
    $core.int? hours,
    $core.int? minutes,
    $core.int? seconds,
    $core.int? nanos,
    $0.Duration? utcOffset,
    TimeZone? timeZone,
  }) {
    final result = create();
    if (year != null) result.year = year;
    if (month != null) result.month = month;
    if (day != null) result.day = day;
    if (hours != null) result.hours = hours;
    if (minutes != null) result.minutes = minutes;
    if (seconds != null) result.seconds = seconds;
    if (nanos != null) result.nanos = nanos;
    if (utcOffset != null) result.utcOffset = utcOffset;
    if (timeZone != null) result.timeZone = timeZone;
    return result;
  }

  DateTime._();

  factory DateTime.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DateTime.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, DateTime_TimeOffset>
      _DateTime_TimeOffsetByTag = {
    8: DateTime_TimeOffset.utcOffset,
    9: DateTime_TimeOffset.timeZone,
    0: DateTime_TimeOffset.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DateTime',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'google.type'),
      createEmptyInstance: create)
    ..oo(0, [8, 9])
    ..a<$core.int>(1, _omitFieldNames ? '' : 'year', $pb.PbFieldType.O3)
    ..a<$core.int>(2, _omitFieldNames ? '' : 'month', $pb.PbFieldType.O3)
    ..a<$core.int>(3, _omitFieldNames ? '' : 'day', $pb.PbFieldType.O3)
    ..a<$core.int>(4, _omitFieldNames ? '' : 'hours', $pb.PbFieldType.O3)
    ..a<$core.int>(5, _omitFieldNames ? '' : 'minutes', $pb.PbFieldType.O3)
    ..a<$core.int>(6, _omitFieldNames ? '' : 'seconds', $pb.PbFieldType.O3)
    ..a<$core.int>(7, _omitFieldNames ? '' : 'nanos', $pb.PbFieldType.O3)
    ..aOM<$0.Duration>(8, _omitFieldNames ? '' : 'utcOffset',
        subBuilder: $0.Duration.create)
    ..aOM<TimeZone>(9, _omitFieldNames ? '' : 'timeZone',
        subBuilder: TimeZone.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DateTime clone() => DateTime()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DateTime copyWith(void Function(DateTime) updates) =>
      super.copyWith((message) => updates(message as DateTime)) as DateTime;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DateTime create() => DateTime._();
  @$core.override
  DateTime createEmptyInstance() => create();
  static $pb.PbList<DateTime> createRepeated() => $pb.PbList<DateTime>();
  @$core.pragma('dart2js:noInline')
  static DateTime getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DateTime>(create);
  static DateTime? _defaultInstance;

  DateTime_TimeOffset whichTimeOffset() =>
      _DateTime_TimeOffsetByTag[$_whichOneof(0)]!;
  void clearTimeOffset() => $_clearField($_whichOneof(0));

  /// Optional. Year of date. Must be from 1 to 9999, or 0 if specifying a
  /// datetime without a year.
  @$pb.TagNumber(1)
  $core.int get year => $_getIZ(0);
  @$pb.TagNumber(1)
  set year($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasYear() => $_has(0);
  @$pb.TagNumber(1)
  void clearYear() => $_clearField(1);

  /// Required. Month of year. Must be from 1 to 12.
  @$pb.TagNumber(2)
  $core.int get month => $_getIZ(1);
  @$pb.TagNumber(2)
  set month($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMonth() => $_has(1);
  @$pb.TagNumber(2)
  void clearMonth() => $_clearField(2);

  /// Required. Day of month. Must be from 1 to 31 and valid for the year and
  /// month.
  @$pb.TagNumber(3)
  $core.int get day => $_getIZ(2);
  @$pb.TagNumber(3)
  set day($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDay() => $_has(2);
  @$pb.TagNumber(3)
  void clearDay() => $_clearField(3);

  /// Required. Hours of day in 24 hour format. Should be from 0 to 23. An API
  /// may choose to allow the value "24:00:00" for scenarios like business
  /// closing time.
  @$pb.TagNumber(4)
  $core.int get hours => $_getIZ(3);
  @$pb.TagNumber(4)
  set hours($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasHours() => $_has(3);
  @$pb.TagNumber(4)
  void clearHours() => $_clearField(4);

  /// Required. Minutes of hour of day. Must be from 0 to 59.
  @$pb.TagNumber(5)
  $core.int get minutes => $_getIZ(4);
  @$pb.TagNumber(5)
  set minutes($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMinutes() => $_has(4);
  @$pb.TagNumber(5)
  void clearMinutes() => $_clearField(5);

  /// Required. Seconds of minutes of the time. Must normally be from 0 to 59. An
  /// API may allow the value 60 if it allows leap-seconds.
  @$pb.TagNumber(6)
  $core.int get seconds => $_getIZ(5);
  @$pb.TagNumber(6)
  set seconds($core.int value) => $_setSignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasSeconds() => $_has(5);
  @$pb.TagNumber(6)
  void clearSeconds() => $_clearField(6);

  /// Required. Fractions of seconds in nanoseconds. Must be from 0 to
  /// 999,999,999.
  @$pb.TagNumber(7)
  $core.int get nanos => $_getIZ(6);
  @$pb.TagNumber(7)
  set nanos($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasNanos() => $_has(6);
  @$pb.TagNumber(7)
  void clearNanos() => $_clearField(7);

  /// UTC offset. Must be whole seconds, between -18 hours and +18 hours.
  /// For example, a UTC offset of -4:00 would be represented as
  /// { seconds: -14400 }.
  @$pb.TagNumber(8)
  $0.Duration get utcOffset => $_getN(7);
  @$pb.TagNumber(8)
  set utcOffset($0.Duration value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasUtcOffset() => $_has(7);
  @$pb.TagNumber(8)
  void clearUtcOffset() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Duration ensureUtcOffset() => $_ensure(7);

  /// Time zone.
  @$pb.TagNumber(9)
  TimeZone get timeZone => $_getN(8);
  @$pb.TagNumber(9)
  set timeZone(TimeZone value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasTimeZone() => $_has(8);
  @$pb.TagNumber(9)
  void clearTimeZone() => $_clearField(9);
  @$pb.TagNumber(9)
  TimeZone ensureTimeZone() => $_ensure(8);
}

/// Represents a time zone from the
/// [IANA Time Zone Database](https://www.iana.org/time-zones).
class TimeZone extends $pb.GeneratedMessage {
  factory TimeZone({
    $core.String? id,
    $core.String? version,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (version != null) result.version = version;
    return result;
  }

  TimeZone._();

  factory TimeZone.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TimeZone.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TimeZone',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'google.type'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'version')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TimeZone clone() => TimeZone()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TimeZone copyWith(void Function(TimeZone) updates) =>
      super.copyWith((message) => updates(message as TimeZone)) as TimeZone;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TimeZone create() => TimeZone._();
  @$core.override
  TimeZone createEmptyInstance() => create();
  static $pb.PbList<TimeZone> createRepeated() => $pb.PbList<TimeZone>();
  @$core.pragma('dart2js:noInline')
  static TimeZone getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<TimeZone>(create);
  static TimeZone? _defaultInstance;

  /// IANA Time Zone Database time zone, e.g. "America/New_York".
  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  /// Optional. IANA Time Zone Database version number, e.g. "2019a".
  @$pb.TagNumber(2)
  $core.String get version => $_getSZ(1);
  @$pb.TagNumber(2)
  set version($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
