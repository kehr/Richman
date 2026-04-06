package user_settings

import (
	"fmt"
	"reflect"
	"strings"
)

// AssertNoCapitalLeakage is the runtime guard required by TRD §5.2. It walks
// the json tags of a DTO (struct, slice of struct, or pointer thereof) and
// returns an error if any field name references absolute capital/amount
// information that must never reach an analysis, LLM context, or public card
// surface.
//
// Forbidden json-tag substrings (case-insensitive match on the tag name only,
// ignoring options like ",omitempty"):
//
//   - "totalcapital"  — the user's total capital should never be embedded.
//   - "amount"        — any absolute amount field (PositionAmount, etc.).
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
// in step 09 of the API DTO alignment) — this runtime helper exists to catch
// any accidental regression at test time or in debug builds.
//
// The guard recurses into nested structs and slices so it can validate an
// entire response payload in one call. Map and chan fields are not traversed.
func AssertNoCapitalLeakage(v any) error {
	if v == nil {
		return nil
	}
	return walkGuard(reflect.ValueOf(v), "")
}

// forbiddenSubstrings lists the lowercase substrings that must not appear in
// any json tag name of a guarded DTO.
var forbiddenSubstrings = []string{"totalcapital", "amount"}

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
		// Validate element type by using a zero value of the element type so
		// empty slices still get checked. Then also walk any populated items
		// to catch interface slices carrying different concrete types.
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
		if !sf.IsExported() {
			continue
		}
		tag := jsonTagName(sf.Tag.Get("json"))
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = sf.Name
		}
		fieldPath := tag
		if path != "" {
			fieldPath = path + "." + tag
		}
		lower := strings.ToLower(tag)
		for _, bad := range forbiddenSubstrings {
			if strings.Contains(lower, bad) {
				return fmt.Errorf(
					"privacy guard: field %q leaks capital info (json tag %q contains %q)",
					fieldPath, tag, bad,
				)
			}
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
		if !sf.IsExported() {
			continue
		}
		tag := jsonTagName(sf.Tag.Get("json"))
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = sf.Name
		}
		fieldPath := tag
		if path != "" {
			fieldPath = path + "." + tag
		}
		lower := strings.ToLower(tag)
		for _, bad := range forbiddenSubstrings {
			if strings.Contains(lower, bad) {
				return fmt.Errorf(
					"privacy guard: field %q leaks capital info (json tag %q contains %q)",
					fieldPath, tag, bad,
				)
			}
		}
		if err := walkType(sf.Type, fieldPath); err != nil {
			return err
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
