package recommendation

import (
	"bytes"
	"crypto/sha1" // #nosec G505 -- fingerprint is non-cryptographic, collision domain is tiny
	"encoding/hex"
	"fmt"
	"sort"
)

// Fingerprint computes a stable SHA-1 hex digest over the load-bearing
// fields of an execution plan. The digest is used by the badge diff
// algorithm to detect "plan adjusted" transitions (TRD §3.3).
//
// Stable fields, in fixed order:
//
//  1. execution.Type
//  2. targetPositionPct
//  3. execution.StopLoss   (or "nil" placeholder)
//  4. execution.TakeProfit (or "nil" placeholder)
//  5. each Step (sorted by Order ascending):
//     TriggerType | TriggerValue | DeltaPct
//
// Step.Rationale is intentionally excluded: the LLM regenerates the
// natural-language rationale on every call so including it would force a
// fingerprint change on every analysis even when the actionable plan is
// identical.
func Fingerprint(targetPositionPct float64, exec Execution) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "type=%s\n", exec.Type)
	fmt.Fprintf(&buf, "target=%.6f\n", targetPositionPct)
	fmt.Fprintf(&buf, "stopLoss=%s\n", floatPtrToken(exec.StopLoss))
	fmt.Fprintf(&buf, "takeProfit=%s\n", floatPtrToken(exec.TakeProfit))

	steps := make([]Step, len(exec.Steps))
	copy(steps, exec.Steps)
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].Order < steps[j].Order
	})

	for _, s := range steps {
		fmt.Fprintf(&buf, "step|%d|%s|%s|%.6f\n",
			s.Order, s.TriggerType, s.TriggerValue, s.DeltaPct)
	}

	sum := sha1.Sum(buf.Bytes()) // #nosec G401
	return hex.EncodeToString(sum[:])
}

// floatPtrToken renders a *float64 as either its fixed-precision decimal
// representation or the literal string "nil" so that the absence of a
// guard rail produces a stable, distinguishable fingerprint input.
func floatPtrToken(v *float64) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%.6f", *v)
}
