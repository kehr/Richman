package recommendation

import "testing"

func TestActionLevel(t *testing.T) {
	cases := []struct {
		name   string
		action Action
		want   int
	}{
		{"aggressive add maps to +2", ActionAggressiveAdd, 2},
		{"small add maps to +1", ActionSmallAdd, 1},
		{"hold maps to 0", ActionHold, 0},
		{"gradual reduce maps to -1", ActionGradualReduce, -1},
		{"control position maps to -2", ActionControl, -2},
		{"unknown action falls back to 0", Action("unknown"), 0},
		{"empty action falls back to 0", Action(""), 0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.action.Level(); got != tc.want {
				t.Fatalf("Level() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if ConfidenceShiftThreshold != 10.0 {
		t.Fatalf("ConfidenceShiftThreshold = %v, want 10.0", ConfidenceShiftThreshold)
	}
	if ValidityDefaultDays != 7 {
		t.Fatalf("ValidityDefaultDays = %v, want 7", ValidityDefaultDays)
	}
}
