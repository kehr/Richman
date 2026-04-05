package yahoo

import (
	"testing"
	"time"

	"github.com/richman/backend/internal/datasource"
)

// sampleChartJSON is a realistic Yahoo Finance v8 chart API response for testing.
const sampleChartJSON = `{
	"chart": {
		"result": [{
			"timestamp": [1704067200, 1704153600, 1704240000],
			"indicators": {
				"quote": [{
					"open":   [185.50, 186.20, 184.90],
					"high":   [186.80, 187.10, 186.50],
					"low":    [184.20, 185.00, 183.80],
					"close":  [186.00, 185.50, 186.30],
					"volume": [55000000, 48000000, 52000000]
				}]
			}
		}],
		"error": null
	}
}`

func TestParseChartResponse(t *testing.T) {
	prices, err := ParseChartResponse([]byte(sampleChartJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}

	// Verify first data point.
	p := prices[0]
	expectedDate := time.Unix(1704067200, 0).UTC().Truncate(24 * time.Hour)
	if !p.Date.Equal(expectedDate) {
		t.Errorf("date: got %v, want %v", p.Date, expectedDate)
	}
	if p.Open != 185.50 {
		t.Errorf("open: got %f, want 185.50", p.Open)
	}
	if p.Close != 186.00 {
		t.Errorf("close: got %f, want 186.00", p.Close)
	}
	if p.Volume != 55000000 {
		t.Errorf("volume: got %f, want 55000000", p.Volume)
	}
}

func TestParseChartResponseWithNulls(t *testing.T) {
	jsonWithNulls := `{
		"chart": {
			"result": [{
				"timestamp": [1704067200, 1704153600],
				"indicators": {
					"quote": [{
						"open":   [185.50, null],
						"high":   [186.80, null],
						"low":    [184.20, null],
						"close":  [186.00, null],
						"volume": [55000000, null]
					}]
				}
			}],
			"error": null
		}
	}`

	prices, err := ParseChartResponse([]byte(jsonWithNulls))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Null entries should be skipped.
	if len(prices) != 1 {
		t.Fatalf("expected 1 price (null skipped), got %d", len(prices))
	}
}

func TestParseChartResponseError(t *testing.T) {
	errorJSON := `{
		"chart": {
			"result": null,
			"error": {
				"code": "Not Found",
				"description": "No data found for symbol INVALID"
			}
		}
	}`

	_, err := ParseChartResponse([]byte(errorJSON))
	if err == nil {
		t.Fatal("expected error for error response, got nil")
	}
	if !containsError(err, datasource.ErrInvalidResponse) {
		t.Errorf("expected ErrInvalidResponse, got: %v", err)
	}
}

func TestParseChartResponseEmptyResult(t *testing.T) {
	emptyJSON := `{"chart": {"result": [], "error": null}}`

	_, err := ParseChartResponse([]byte(emptyJSON))
	if err == nil {
		t.Fatal("expected error for empty result, got nil")
	}
}

func TestParseChartResponseInvalidJSON(t *testing.T) {
	_, err := ParseChartResponse([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func containsError(err, target error) bool {
	for e := err; e != nil; {
		if e.Error() == target.Error() {
			return true
		}
		// Walk the chain manually since errors.Is checks identity.
		if unwrapped, ok := e.(interface{ Unwrap() error }); ok {
			e = unwrapped.Unwrap()
		} else {
			break
		}
	}
	return false
}
