// This is a generated file - do not edit.
//
// Generated from google/type/calendar_period.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// A `CalendarPeriod` represents the abstract concept of a time period that has
/// a canonical start. Grammatically, "the start of the current
/// `CalendarPeriod`." All calendar times begin at midnight UTC.
class CalendarPeriod extends $pb.ProtobufEnum {
  /// Undefined period, raises an error.
  static const CalendarPeriod CALENDAR_PERIOD_UNSPECIFIED =
      CalendarPeriod._(0, _omitEnumNames ? '' : 'CALENDAR_PERIOD_UNSPECIFIED');

  /// A day.
  static const CalendarPeriod DAY =
      CalendarPeriod._(1, _omitEnumNames ? '' : 'DAY');

  /// A week. Weeks begin on Monday, following
  /// [ISO 8601](https://en.wikipedia.org/wiki/ISO_week_date).
  static const CalendarPeriod WEEK =
      CalendarPeriod._(2, _omitEnumNames ? '' : 'WEEK');

  /// A fortnight. The first calendar fortnight of the year begins at the start
  /// of week 1 according to
  /// [ISO 8601](https://en.wikipedia.org/wiki/ISO_week_date).
  static const CalendarPeriod FORTNIGHT =
      CalendarPeriod._(3, _omitEnumNames ? '' : 'FORTNIGHT');

  /// A month.
  static const CalendarPeriod MONTH =
      CalendarPeriod._(4, _omitEnumNames ? '' : 'MONTH');

  /// A quarter. Quarters start on dates 1-Jan, 1-Apr, 1-Jul, and 1-Oct of each
  /// year.
  static const CalendarPeriod QUARTER =
      CalendarPeriod._(5, _omitEnumNames ? '' : 'QUARTER');

  /// A half-year. Half-years start on dates 1-Jan and 1-Jul.
  static const CalendarPeriod HALF =
      CalendarPeriod._(6, _omitEnumNames ? '' : 'HALF');

  /// A year.
  static const CalendarPeriod YEAR =
      CalendarPeriod._(7, _omitEnumNames ? '' : 'YEAR');

  static const $core.List<CalendarPeriod> values = <CalendarPeriod>[
    CALENDAR_PERIOD_UNSPECIFIED,
    DAY,
    WEEK,
    FORTNIGHT,
    MONTH,
    QUARTER,
    HALF,
    YEAR,
  ];

  static final $core.List<CalendarPeriod?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 7);
  static CalendarPeriod? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CalendarPeriod._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
