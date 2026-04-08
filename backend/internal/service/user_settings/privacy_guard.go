package user_settings

import (
	"fmt"
	"reflect"
	"strings"
)

// AssertNoCapitalLeakage is the runtime guard required by TRD §5.2. It walks
// the json tags of a DTO (struct, slice of struct, or pointer thereof) and
// returns an error if any field name references absolute capital/amount
// information that must never reach an analysis, LLM context, or push
// notification render path.
//
// Forbidden json-tag substrings (case-insensitive, matched against the tag
// name only — options like ",omitempty" are stripped):
//
//   - "totalcapital"         — the user's total capital should never be embedded.
//   - "positionamount"       — absolute position value (PositionPct companion).
//   - "targetpositionamount" — absolute target position value.
//   - "unrealizedamount"     — absolute unrealized gain/loss.
//   - "realizedamount"       — absolute realized gain/loss.
//
// The list is deliberately specific (rather than the broader "amount"
// substring) to avoid false positives on unrelated fields such as
// paymentAmount, minAmount, amountOfShares, etc. If a new absolute-value
// field is introduced that should be guarded, add it to forbiddenSubstrings
// explicitly.
//
// Three compile-time constraints complement this runtime guard (TRD §5.2):
//
//  1. Analysis pipeline input types must not embed total_capital.
//  2. LLM context construction functions must not accept types containing
//     total_capital.
//  3. Push notification render functions must only accept PublicCardSummary
//     DTOs (which do not carry amount fields).
//
// Those constraints are enforced by architectural convention (type isolation
// in Step 09 of the API DTO alignment) — this runtime helper exists to catch
// any accidental regression at test time.
//
// The guard recurses into nested structs and slices so it can validate an
// entire response payload in one call. Map and chan fields are not traversed.
//
// Precondition: the input must be a tree-shaped DTO (no cyclic pointers).
// The walker does not maintain a visited set, so a self-referential struct
// with a populated cycle will recurse infinitely. DTOs produced by the API
// layer are always tree-shaped.
//
// If this guard rejects a legitimate field, the recommended escape hatches
// are (a) rename the field so its json tag no longer contains a forbidden
// substring, or (b) use `json:"-"` if the field is purely internal.
func AssertNoCapitalLeakage(v any) error {
	if v == nil {
		return nil
	}
	return walkGuard(reflect.ValueOf(v), "")
}

// forbiddenSubstrings lists the lowercase substrings that must not appear in
// any json tag name of a guarded DTO. Kept specific to avoid false positives
// on benign amount-like fields.
var forbiddenSubstrings = []string{
	"totalcapital",
	"positionamount",
	"targetpositionamount",
	"unrealizedamount",
	"realizedamount",
}

func walkGuard(v reflect.Value, path string) error {
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return walkGuard(v.Elem(), path)
	case reflect.Slice, reflect.Array:
		// Validate element type by structural walk so empty slices still get
		// checked, then walk any populated items for interface slices carrying
		// heterogeneous concrete types.
		if err := walkType(v.Type().Elem(), path); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if err := walkGuard(v.Index(i), path); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		return walkStruct(v, path)
	default:
		return nil
	}
}

func walkStruct(v reflect.Value, path string) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fieldPath, tag, skip := checkFieldTag(&sf, path)
		if skip {
			continue
		}
		if err := tagViolation(tag, fieldPath); err != nil {
			return err
		}
		if err := walkGuard(v.Field(i), fieldPath); err != nil {
			return err
		}
	}
	return nil
}

// walkType performs a structural check on a type (without a concrete value)
// so empty slices still validate their element type.
func walkType(t reflect.Type, path string) error {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		fieldPath, tag, skip := checkFieldTag(&sf, path)
		if skip {
			continue
		}
		if err := tagViolation(tag, fieldPath); err != nil {
			return err
		}
		if err := walkType(sf.Type, fieldPath); err != nil {
			return err
		}
	}
	return nil
}

// checkFieldTag extracts the effective json tag name and full dotted path
// for a struct field. Returns skip=true when the field is unexported or
// json:"-" (i.e. should not participate in the guard walk).
func checkFieldTag(sf *reflect.StructField, path string) (fieldPath, tag string, skip bool) {
	if !sf.IsExported() {
		return "", "", true
	}
	tag = jsonTagName(sf.Tag.Get("json"))
	if tag == "-" {
		return "", "", true
	}
	if tag == "" {
		tag = sf.Name
	}
	fieldPath = tag
	if path != "" {
		fieldPath = path + "." + tag
	}
	return fieldPath, tag, false
}

// tagViolation returns a non-nil error when the tag name contains any
// forbidden substring. The error message includes the dotted field path,
// the raw tag, and the offending substring to make the escape-hatch path
// (rename or json:"-") obvious.
func tagViolation(tag, fieldPath string) error {
	lower := strings.ToLower(tag)
	for _, bad := range forbiddenSubstrings {
		if strings.Contains(lower, bad) {
			return fmt.Errorf(
				"privacy guard: field %q leaks capital info (json tag %q contains %q); "+
					"rename the field or use json:\"-\" if purely internal",
				fieldPath, tag, bad,
			)
		}
	}
	return nil
}

// jsonTagName extracts the name part of a json struct tag (everything before
// the first comma).
func jsonTagName(tag string) string {
	if tag == "" {
		return ""
	}
	if idx := strings.Index(tag, ","); idx >= 0 {
		return tag[:idx]
	}
	return tag
}
