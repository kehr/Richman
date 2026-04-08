package user_settings

import (
	"reflect"
	"strings"
)

// AttachAmounts walks the top-level fields of dto and, for every field whose
// name ends in "Pct" (or "Ratio" for PositionRatio / TargetPositionRatio), it
// looks for a sibling field named with the "Amount" suffix and fills it.
//
// Rules:
//
//   - "PositionRatio"        → "PositionAmount"
//   - "TargetPositionRatio"  → "TargetPositionAmount"
//   - "<X>Pct"               → "<X>Amount"
//
// The Pct/Ratio value is interpreted as a 0-100 percentage (not 0-1), matching
// the convention already used throughout the analysis/decision pipeline. The
// amount is computed as `capital * pct / 100`.
//
// If totalCapital is nil, AttachAmounts is a no-op: no Amount fields are
// written, so the DTO's optional Amount pointers remain nil and will be
// omitted from JSON via `omitempty`.
//
// The Amount target field must be of type `*float64`. If it does not exist or
// has a different type, AttachAmounts silently skips it — this matches the
// "some DTOs don't expose amounts" policy and keeps the helper safe to call
// unconditionally at the API layer.
//
// AttachAmounts only inspects top-level fields; it does not recurse into
// nested structs. Callers that need nested attachment should call it once per
// sub-struct explicitly.
//
// dto must be a non-nil pointer to a struct. Any other input is a no-op.
func AttachAmounts(dto any, totalCapital *float64) {
	if dto == nil || totalCapital == nil {
		return
	}
	v := reflect.ValueOf(dto)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}

	capital := *totalCapital
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		amountName := amountFieldFor(field.Name)
		if amountName == "" {
			continue
		}

		// Read the pct value.
		pctVal := v.Field(i)
		pct, ok := readFloat(pctVal)
		if !ok {
			continue
		}

		// Locate the sibling Amount field.
		target := v.FieldByName(amountName)
		if !target.IsValid() || !target.CanSet() {
			continue
		}
		// Must be *float64.
		if target.Kind() != reflect.Ptr || target.Type().Elem().Kind() != reflect.Float64 {
			continue
		}

		amount := capital * pct / 100.0
		target.Set(reflect.ValueOf(&amount))
	}
}

// amountFieldFor maps a percentage-style field name to the sibling amount
// field name. It returns an empty string when the field does not participate
// in the pct → amount projection.
func amountFieldFor(name string) string {
	switch {
	case strings.HasSuffix(name, "Pct"):
		return strings.TrimSuffix(name, "Pct") + "Amount"
	case name == "PositionRatio":
		return "PositionAmount"
	case name == "TargetPositionRatio":
		return "TargetPositionAmount"
	default:
		return ""
	}
}

// readFloat extracts a float64 from a reflect.Value, accepting float64 or
// *float64 (treating nil pointers as "no value").
func readFloat(v reflect.Value) (float64, bool) {
	switch v.Kind() {
	case reflect.Float64, reflect.Float32:
		return v.Float(), true
	case reflect.Ptr:
		if v.IsNil() {
			return 0, false
		}
		if v.Elem().Kind() == reflect.Float64 || v.Elem().Kind() == reflect.Float32 {
			return v.Elem().Float(), true
		}
	}
	return 0, false
}
