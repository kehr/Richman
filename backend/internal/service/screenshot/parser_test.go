package screenshot

import (
	"errors"
	"strings"
	"testing"
)

func TestParse_ValidHighConfidence(t *testing.T) {
	raw := `{
		"holdings": [
			{
				"assetName":   {"value": "贵州茅台", "confidence": 0.95},
				"assetCode":   {"value": "600519",   "confidence": 0.9},
				"costPrice":   {"value": "1800.00",  "confidence": 0.88},
				"positionPct": {"value": "40.0",     "confidence": 0.87},
				"assetTypeGuess": "a_share"
			}
		]
	}`

	resp, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if resp.OverallStatus != StatusOK {
		t.Errorf("want status %q, got %q", StatusOK, resp.OverallStatus)
	}
	if len(resp.Holdings) != 1 {
		t.Fatalf("want 1 holding, got %d", len(resp.Holdings))
	}
	h := resp.Holdings[0]
	if h.AssetName.Value != "贵州茅台" || h.AssetName.Confidence != 0.95 {
		t.Errorf("assetName mismatch: %+v", h.AssetName)
	}
	if h.AssetTypeGuess != "a_share" {
		t.Errorf("assetTypeGuess mismatch: %q", h.AssetTypeGuess)
	}
}

func TestParse_AllFieldsLowConfidence(t *testing.T) {
	raw := `{
		"holdings": [
			{
				"assetName":   {"value": "?", "confidence": 0.2},
				"assetCode":   {"value": "",  "confidence": 0.0},
				"costPrice":   {"value": "",  "confidence": 0.0},
				"positionPct": {"value": "",  "confidence": 0.0},
				"assetTypeGuess": ""
			}
		]
	}`

	resp, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if resp.OverallStatus != StatusLowQuality {
		t.Errorf("want status %q, got %q", StatusLowQuality, resp.OverallStatus)
	}
}

func TestParse_EmptyHoldingsIsLowQuality(t *testing.T) {
	resp, err := Parse(`{"holdings": []}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if resp.OverallStatus != StatusLowQuality {
		t.Errorf("want status %q, got %q", StatusLowQuality, resp.OverallStatus)
	}
	if resp.Holdings == nil {
		t.Errorf("holdings should be a non-nil empty slice")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	cases := []string{
		"",
		"not json at all",
		"{",
		`{"holdings": "not an array"}`,
	}
	for _, in := range cases {
		_, err := Parse(in)
		if !errors.Is(err, ErrInvalidJSON) {
			t.Errorf("input %q: want ErrInvalidJSON, got %v", in, err)
		}
	}
}

func TestParse_StripsMarkdownCodeFence(t *testing.T) {
	raw := "```json\n" + `{"holdings": [{"assetName": {"value": "A", "confidence": 0.9}, "assetCode": {"value": "", "confidence": 0}, "costPrice": {"value": "", "confidence": 0}, "positionPct": {"value": "", "confidence": 0}, "assetTypeGuess": ""}]}` + "\n```"
	resp, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if resp.OverallStatus != StatusOK {
		t.Errorf("want status %q, got %q", StatusOK, resp.OverallStatus)
	}
}

func TestParse_MixedConfidenceMarksOK(t *testing.T) {
	// One field above the threshold is enough to grade the batch OK.
	raw := `{
		"holdings": [
			{
				"assetName":   {"value": "AAPL", "confidence": 0.7},
				"assetCode":   {"value": "",      "confidence": 0.1},
				"costPrice":   {"value": "",      "confidence": 0.1},
				"positionPct": {"value": "",      "confidence": 0.1},
				"assetTypeGuess": "us_stock"
			}
		]
	}`
	resp, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if resp.OverallStatus != StatusOK {
		t.Errorf("want status %q, got %q", StatusOK, resp.OverallStatus)
	}
}

func TestStripCodeFence_HandlesBareFence(t *testing.T) {
	// A stand-alone fence without JSON should collapse to empty.
	got := stripCodeFence("```")
	if strings.TrimSpace(got) != "" {
		t.Errorf("want empty, got %q", got)
	}
}
