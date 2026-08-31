---
page_title: "API Assertions v2 - UptimeRobot Provider"
description: |-
  Configure API Assertions v2, understand validation and upgrade behavior, and plan safe provider rollout.
---

# API Assertions v2

API Assertions v2 extends the existing `uptimerobot_monitor.config.api_assertions` shape. It does not add a Terraform state-shape version or expose an internal semantics-version setting.

## Sources and comparisons

| Property source | Property form | Supported comparisons |
| --- | --- | --- |
| Parsed JSON body | `$...` | `equals`, `not_equals`, `contains`, `not_contains`, `greater_than`, `less_than`, `is_null`, `is_not_null`, `is_string`, `is_number`, `is_boolean`, `is_array`, `is_object`, `exists`, `not_exists`, `is_empty`, `is_not_empty`, `length_equals`, `length_not_equals`, `length_greater_than`, `length_less_than` |
| Response header | `headers.<HTTP field name>` | `equals`, `not_equals`, `contains`, `not_contains`, `is_string`, `exists`, `not_exists` |
| HTTP status | `status_code` | `equals`, `not_equals`, `greater_than`, `less_than`, `is_number` |
| Raw response body | `body` | `equals`, `not_equals`, `contains`, `not_contains`, `is_string` |

The provider accepts between one and five checks. Checks form an order-insensitive multiset: duplicates are preserved, reordering alone is not a change, and removing one duplicate is a material change.

## Targets

The target-taking comparisons are `equals`, `not_equals`, `contains`, `not_contains`, `greater_than`, `less_than`, and the four `length_*` comparisons. Every other comparison takes no concrete target; omit `target` or use `target = jsonencode(null)`.

- For parsed JSON bodies, `equals` and `not_equals` accept any non-null scalar, array, or object target. Structured equality is recursive and type-strict; object key order is ignored and array order is significant.
- For parsed JSON bodies, `contains` and `not_contains` accept any non-null JSON target. Arrays look for an exact element, objects use recursive object-subset matching, and strings retain substring behavior.
- Header and raw-body `contains` and `not_contains` still require strings. An empty string is API-valid.
- `greater_than` and `less_than` require a JSON number.
- `is_empty` and `is_not_empty` take no target and apply to selected arrays and objects.
- `length_equals`, `length_not_equals`, `length_greater_than`, and `length_less_than` require a non-negative safe integer and count array elements or object keys.
- Header and raw-body equality require strings.
- Status equality accepts a number or a legacy unsigned numeric string.
- Structured numbers must be finite IEEE-754 binary64 values; integers must be within `-9007199254740991` through `9007199254740991`.
- Binary64-equivalent number spellings and negative zero versus zero are non-material during refresh, avoiding drift if API storage normalizes the JSON number text.
- A property can contain at most 500 Unicode characters. A compact serialized target can contain at most 2048 UTF-8 bytes, and a structured target can contain at most 16 array/object levels.
- Object keys must be unique. The provider rejects duplicates from the original `jsontypes.Normalized` JSON before map decoding can discard them.

`target` remains a JSON string in Terraform state because that is the provider's HCL/state representation. During request expansion, the provider decodes it exactly once and sends arrays and objects as native JSON values. A value such as `jsonencode("[1,2]")` therefore remains a string; it is never reparsed as an array.

The provider does not use secret-detection heuristics and does not reject an otherwise API-valid value because it resembles a credential.

## JSONPath authority

API Internal owns parsing and validation of the documented safe RFC 9535-compatible subset, including the 16-selector depth limit. Terraform validates only the JSONPath source shape (`$`, `$.`, or `$[`), the known property length, and the source/comparison/target contract. It never evaluates selectors or requests the monitored URL during validation or planning.

This division keeps Terraform from becoming a second JSONPath or assertion-evaluation authority. A selector rejected by API Internal is returned as an API validation error.

## Sensitive data and state

`target` is marked sensitive, so Terraform redacts it in plans where nested sensitivity is supported. Sensitive values are still stored in Terraform state. In addition:

- the nested checks collection can reveal its count and structure;
- `property` remains non-sensitive so diffs are useful;
- JSONPath filter literals inside `property` are visible in configuration, plans, and state.

Treat the state backend and every reader as trusted before placing confidential data in an assertion. Prefer selectors and targets that do not contain credentials.

## Upgrade and compatibility

Existing provider configurations keep the same HCL shape. Imported legacy API monitors are read without assigning semantics in Terraform. API Internal owns semantics-version assignment and strips internal markers, per-check identifiers, and diagnostics from the public contract.

An unrelated update or a canonically unchanged assertion update preserves legacy semantics. Header-name casing, default `AND`, status numeric-string normalization, and check ordering are non-material; duplicate counts and omitted-versus-explicit-null target presence remain material. A material assertion change must satisfy the v2 matrix and API Internal assigns v2 semantics.

Terraform unknown values are deferred until their dependent property, comparison, or target is known. In particular, the provider does not reject a structured or non-string target while an unknown property may still resolve to a body-JSON source.

## Implementation and release order

Implementation can start on the provider epic branch as soon as the frozen contract and the core runtime implementation are available; it does not need to wait for a service release. Before publishing the provider release, deploy and verify:

1. the frozen Core Assertions contract and Go runtime;
2. monitoring consumers using that runtime;
3. API Internal validation, readiness gate, and semantics assignment;
4. dashboard, Terraform, and QA coverage for creating, editing, transporting, and diagnosing the same contract.

Only then should the Terraform provider be pushed and released.
