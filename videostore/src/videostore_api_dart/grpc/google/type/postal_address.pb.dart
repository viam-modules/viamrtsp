// This is a generated file - do not edit.
//
// Generated from google/type/postal_address.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// Represents a postal address, e.g. for postal delivery or payments addresses.
/// Given a postal address, a postal service can deliver items to a premise, P.O.
/// Box or similar.
/// It is not intended to model geographical locations (roads, towns,
/// mountains).
///
/// In typical usage an address would be created via user input or from importing
/// existing data, depending on the type of process.
///
/// Advice on address input / editing:
///  - Use an i18n-ready address widget such as
///    https://github.com/google/libaddressinput)
/// - Users should not be presented with UI elements for input or editing of
///   fields outside countries where that field is used.
///
/// For more guidance on how to use this schema, please see:
/// https://support.google.com/business/answer/6397478
class PostalAddress extends $pb.GeneratedMessage {
  factory PostalAddress({
    $core.int? revision,
    $core.String? regionCode,
    $core.String? languageCode,
    $core.String? postalCode,
    $core.String? sortingCode,
    $core.String? administrativeArea,
    $core.String? locality,
    $core.String? sublocality,
    $core.Iterable<$core.String>? addressLines,
    $core.Iterable<$core.String>? recipients,
    $core.String? organization,
  }) {
    final result = create();
    if (revision != null) result.revision = revision;
    if (regionCode != null) result.regionCode = regionCode;
    if (languageCode != null) result.languageCode = languageCode;
    if (postalCode != null) result.postalCode = postalCode;
    if (sortingCode != null) result.sortingCode = sortingCode;
    if (administrativeArea != null)
      result.administrativeArea = administrativeArea;
    if (locality != null) result.locality = locality;
    if (sublocality != null) result.sublocality = sublocality;
    if (addressLines != null) result.addressLines.addAll(addressLines);
    if (recipients != null) result.recipients.addAll(recipients);
    if (organization != null) result.organization = organization;
    return result;
  }

  PostalAddress._();

  factory PostalAddress.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PostalAddress.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PostalAddress',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'google.type'),
      createEmptyInstance: create)
    ..a<$core.int>(1, _omitFieldNames ? '' : 'revision', $pb.PbFieldType.O3)
    ..aOS(2, _omitFieldNames ? '' : 'regionCode')
    ..aOS(3, _omitFieldNames ? '' : 'languageCode')
    ..aOS(4, _omitFieldNames ? '' : 'postalCode')
    ..aOS(5, _omitFieldNames ? '' : 'sortingCode')
    ..aOS(6, _omitFieldNames ? '' : 'administrativeArea')
    ..aOS(7, _omitFieldNames ? '' : 'locality')
    ..aOS(8, _omitFieldNames ? '' : 'sublocality')
    ..pPS(9, _omitFieldNames ? '' : 'addressLines')
    ..pPS(10, _omitFieldNames ? '' : 'recipients')
    ..aOS(11, _omitFieldNames ? '' : 'organization')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PostalAddress clone() => PostalAddress()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PostalAddress copyWith(void Function(PostalAddress) updates) =>
      super.copyWith((message) => updates(message as PostalAddress))
          as PostalAddress;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PostalAddress create() => PostalAddress._();
  @$core.override
  PostalAddress createEmptyInstance() => create();
  static $pb.PbList<PostalAddress> createRepeated() =>
      $pb.PbList<PostalAddress>();
  @$core.pragma('dart2js:noInline')
  static PostalAddress getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PostalAddress>(create);
  static PostalAddress? _defaultInstance;

  /// The schema revision of the `PostalAddress`. This must be set to 0, which is
  /// the latest revision.
  ///
  /// All new revisions **must** be backward compatible with old revisions.
  @$pb.TagNumber(1)
  $core.int get revision => $_getIZ(0);
  @$pb.TagNumber(1)
  set revision($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRevision() => $_has(0);
  @$pb.TagNumber(1)
  void clearRevision() => $_clearField(1);

  /// Required. CLDR region code of the country/region of the address. This
  /// is never inferred and it is up to the user to ensure the value is
  /// correct. See http://cldr.unicode.org/ and
  /// http://www.unicode.org/cldr/charts/30/supplemental/territory_information.html
  /// for details. Example: "CH" for Switzerland.
  @$pb.TagNumber(2)
  $core.String get regionCode => $_getSZ(1);
  @$pb.TagNumber(2)
  set regionCode($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRegionCode() => $_has(1);
  @$pb.TagNumber(2)
  void clearRegionCode() => $_clearField(2);

  /// Optional. BCP-47 language code of the contents of this address (if
  /// known). This is often the UI language of the input form or is expected
  /// to match one of the languages used in the address' country/region, or their
  /// transliterated equivalents.
  /// This can affect formatting in certain countries, but is not critical
  /// to the correctness of the data and will never affect any validation or
  /// other non-formatting related operations.
  ///
  /// If this value is not known, it should be omitted (rather than specifying a
  /// possibly incorrect default).
  ///
  /// Examples: "zh-Hant", "ja", "ja-Latn", "en".
  @$pb.TagNumber(3)
  $core.String get languageCode => $_getSZ(2);
  @$pb.TagNumber(3)
  set languageCode($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLanguageCode() => $_has(2);
  @$pb.TagNumber(3)
  void clearLanguageCode() => $_clearField(3);

  /// Optional. Postal code of the address. Not all countries use or require
  /// postal codes to be present, but where they are used, they may trigger
  /// additional validation with other parts of the address (e.g. state/zip
  /// validation in the U.S.A.).
  @$pb.TagNumber(4)
  $core.String get postalCode => $_getSZ(3);
  @$pb.TagNumber(4)
  set postalCode($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPostalCode() => $_has(3);
  @$pb.TagNumber(4)
  void clearPostalCode() => $_clearField(4);

  /// Optional. Additional, country-specific, sorting code. This is not used
  /// in most regions. Where it is used, the value is either a string like
  /// "CEDEX", optionally followed by a number (e.g. "CEDEX 7"), or just a number
  /// alone, representing the "sector code" (Jamaica), "delivery area indicator"
  /// (Malawi) or "post office indicator" (e.g. Côte d'Ivoire).
  @$pb.TagNumber(5)
  $core.String get sortingCode => $_getSZ(4);
  @$pb.TagNumber(5)
  set sortingCode($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSortingCode() => $_has(4);
  @$pb.TagNumber(5)
  void clearSortingCode() => $_clearField(5);

  /// Optional. Highest administrative subdivision which is used for postal
  /// addresses of a country or region.
  /// For example, this can be a state, a province, an oblast, or a prefecture.
  /// Specifically, for Spain this is the province and not the autonomous
  /// community (e.g. "Barcelona" and not "Catalonia").
  /// Many countries don't use an administrative area in postal addresses. E.g.
  /// in Switzerland this should be left unpopulated.
  @$pb.TagNumber(6)
  $core.String get administrativeArea => $_getSZ(5);
  @$pb.TagNumber(6)
  set administrativeArea($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAdministrativeArea() => $_has(5);
  @$pb.TagNumber(6)
  void clearAdministrativeArea() => $_clearField(6);

  /// Optional. Generally refers to the city/town portion of the address.
  /// Examples: US city, IT comune, UK post town.
  /// In regions of the world where localities are not well defined or do not fit
  /// into this structure well, leave locality empty and use address_lines.
  @$pb.TagNumber(7)
  $core.String get locality => $_getSZ(6);
  @$pb.TagNumber(7)
  set locality($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasLocality() => $_has(6);
  @$pb.TagNumber(7)
  void clearLocality() => $_clearField(7);

  /// Optional. Sublocality of the address.
  /// For example, this can be neighborhoods, boroughs, districts.
  @$pb.TagNumber(8)
  $core.String get sublocality => $_getSZ(7);
  @$pb.TagNumber(8)
  set sublocality($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasSublocality() => $_has(7);
  @$pb.TagNumber(8)
  void clearSublocality() => $_clearField(8);

  /// Unstructured address lines describing the lower levels of an address.
  ///
  /// Because values in address_lines do not have type information and may
  /// sometimes contain multiple values in a single field (e.g.
  /// "Austin, TX"), it is important that the line order is clear. The order of
  /// address lines should be "envelope order" for the country/region of the
  /// address. In places where this can vary (e.g. Japan), address_language is
  /// used to make it explicit (e.g. "ja" for large-to-small ordering and
  /// "ja-Latn" or "en" for small-to-large). This way, the most specific line of
  /// an address can be selected based on the language.
  ///
  /// The minimum permitted structural representation of an address consists
  /// of a region_code with all remaining information placed in the
  /// address_lines. It would be possible to format such an address very
  /// approximately without geocoding, but no semantic reasoning could be
  /// made about any of the address components until it was at least
  /// partially resolved.
  ///
  /// Creating an address only containing a region_code and address_lines, and
  /// then geocoding is the recommended way to handle completely unstructured
  /// addresses (as opposed to guessing which parts of the address should be
  /// localities or administrative areas).
  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get addressLines => $_getList(8);

  /// Optional. The recipient at the address.
  /// This field may, under certain circumstances, contain multiline information.
  /// For example, it might contain "care of" information.
  @$pb.TagNumber(10)
  $pb.PbList<$core.String> get recipients => $_getList(9);

  /// Optional. The name of the organization at the address.
  @$pb.TagNumber(11)
  $core.String get organization => $_getSZ(10);
  @$pb.TagNumber(11)
  set organization($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasOrganization() => $_has(10);
  @$pb.TagNumber(11)
  void clearOrganization() => $_clearField(11);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
