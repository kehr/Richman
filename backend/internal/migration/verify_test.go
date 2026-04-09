package migration

import (
	"reflect"
	"testing"
)

func TestDriftDetailsString(t *testing.T) {
	d := DriftDetails{Missing: []int{10, 11}}
	got := d.String()
	want := "pending versions: [10 11] (run 'make migrate-up' to apply)"
	if got != want {
		t.Errorf("DriftDetails.String() = %q, want %q", got, want)
	}
}

func TestDriftDetailsString_Empty(t *testing.T) {
	d := DriftDetails{}
	got := d.String()
	want := "pending versions: [] (run 'make migrate-up' to apply)"
	if got != want {
		t.Errorf("DriftDetails.String() empty = %q, want %q", got, want)
	}
}

func TestDiffMissingVersions(t *testing.T) {
	tests := []struct {
		name      string
		available []migrationFile
		applied   map[int]struct{}
		want      []int
	}{
		{
			name:      "all applied returns empty",
			available: []migrationFile{{Version: 1}, {Version: 2}, {Version: 3}},
			applied:   map[int]struct{}{1: {}, 2: {}, 3: {}},
			want:      []int{},
		},
		{
			name:      "missing tail version",
			available: []migrationFile{{Version: 1}, {Version: 2}, {Version: 10}},
			applied:   map[int]struct{}{1: {}, 2: {}},
			want:      []int{10},
		},
		{
			name:      "missing multiple non-contiguous",
			available: []migrationFile{{Version: 1}, {Version: 2}, {Version: 5}, {Version: 10}},
			applied:   map[int]struct{}{1: {}, 5: {}},
			want:      []int{2, 10},
		},
		{
			name:      "empty applied returns all versions sorted",
			available: []migrationFile{{Version: 3}, {Version: 1}, {Version: 2}},
			applied:   map[int]struct{}{},
			want:      []int{1, 2, 3},
		},
		{
			name:      "applied has extras returns empty",
			available: []migrationFile{{Version: 1}, {Version: 2}},
			applied:   map[int]struct{}{1: {}, 2: {}, 99: {}},
			want:      []int{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := diffMissingVersions(tc.available, tc.applied)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("diffMissingVersions = %v, want %v", got, tc.want)
			}
		})
	}
}

// Note: VerifyCurrent's full orchestration (file loading + DB query) is
// exercised at the integration level through the server startup path.
// The pure helpers above cover the set-diff and DriftDetails formatting.
// isUndefinedTableError requires a real pgx PgError fixture and is covered
// implicitly when the backend boots against an empty dev database.
