package screenshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// Confidence thresholds used to grade the overall recognition quality.
// See TRD §4.4.
const (
	// ConfidenceHigh marks a field as "trustworthy enough to auto-fill".
	ConfidenceHigh = 0.85
	// ConfidenceLow is the minimum confidence considered usable.
	// Below this value the frontend forces the user to re-enter the field.
	ConfidenceLow = 0.60
)

// Overall status values returned to the client.
const (
	StatusOK         = "ok"
	StatusLowQuality = "low_quality"
	StatusFailed     = "failed"
)

// ErrInvalidJSON is returned when the LLM response cannot be decoded into
// the expected schema.
var ErrInvalidJSON = errors.New("screenshot: llm response is not valid json")

// llmPayload mirrors the JSON schema defined in prompts.go.
type llmPayload struct {
	Holdings []RecognizedHolding `json:"holdings"`
}

// Parse decodes a raw LLM JSON response into the structured holdings list
// and grades the overall recognition status by aggregating per-field
// confidences. It tolerates the model wrapping its reply in whitespace or
// markdown code fences but rejects anything that is not a JSON object.
//
// Grading rules:
//   - JSON decode failure                  -> StatusFailed + ErrInvalidJSON
//   - empty holdings list                  -> StatusLowQuality
//   - any holding has a field >= low       -> StatusOK
//   - otherwise                            -> StatusLowQuality
func Parse(raw string) (*RecognizeResponse, error) {
	trimmed := stripCodeFence(strings.TrimSpace(raw))
	if trimmed == "" {
		return nil, ErrInvalidJSON
	}

	// Strict decoding: reject unknown fields so prompt drift (e.g. the LLM
	// starts emitting "items" instead of "holdings") surfaces as an explicit
	// failure rather than silently producing an empty holdings list.
	dec := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	dec.DisallowUnknownFields()
	var payload llmPayload
	if err := dec.Decode(&payload); err != nil {
		return nil, ErrInvalidJSON
	}

	resp := &RecognizeResponse{
		Holdings: payload.Holdings,
	}
	if resp.Holdings == nil {
		resp.Holdings = []RecognizedHolding{}
	}

	resp.OverallStatus = gradeStatus(resp.Holdings)
	return resp, nil
}

// gradeStatus decides the overall status from the recognized holdings.
// Any single field whose confidence reaches ConfidenceLow is enough to
// mark the batch as usable ("ok"); the frontend still highlights the
// low-confidence fields individually.
func gradeStatus(holdings []RecognizedHolding) string {
	if len(holdings) == 0 {
		return StatusLowQuality
	}
	for _, h := range holdings {
		for _, f := range []Field{h.AssetName, h.AssetCode, h.CostPrice, h.PositionPct} {
			if f.Confidence >= ConfidenceLow {
				return StatusOK
			}
		}
	}
	return StatusLowQuality
}

// stripCodeFence removes a leading/trailing ``` or ```json fence if the
// model wrapped its JSON in a markdown code block despite instructions.
func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Drop the opening fence line.
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[idx+1:]
	} else {
		return ""
	}
	// Drop a trailing closing fence.
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
