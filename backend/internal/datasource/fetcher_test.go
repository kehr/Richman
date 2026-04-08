package datasource

import "testing"

func TestIsChineseNumericCode(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"", false},
		{"GLD", false},
		{"IAU", false},
		{"SGOL", false},
		{"AAPL", false},
		// Shanghai gold ETF.
		{"518880", true},
		// Shenzhen gold ETF (Huaan Gold) — the bug regressed this code to
		// Yahoo before the routing fix.
		{"159934", true},
		// Shanghai 51xxxx range.
		{"510300", true},
		// Shanghai 588xxx STAR Market.
		{"588000", true},
		// Shenzhen 160xxx LOF.
		{"160106", true},
		// Mixed alphanumeric (e.g. a typo) → not pure numeric.
		{"159A34", false},
		// Whitespace padded → not pure numeric.
		{" 159934", false},
	}
	for _, tc := range cases {
		if got := isChineseNumericCode(tc.code); got != tc.want {
			t.Errorf("isChineseNumericCode(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}
