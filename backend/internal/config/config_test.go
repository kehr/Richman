package config

import "testing"

// TestIsProduction verifies the fail-closed semantics of the environment
// guard used by dev-only feature flags (e.g. the onboarding reset endpoint).
// Any unrecognized value must be treated as production.
func TestIsProduction(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"dev", false},
		{"DEV", false},
		{"Dev", false},
		{"test", false},
		{"TEST", false},
		{"staging", false},
		{"Staging", false},
		{"prod", true},
		{"production", true},
		{"PRODUCTION", true},
		{"", true},
		{"unknown", true},
		{"dev-branch", true},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			c := &Config{App: AppConfig{Env: tc.env}}
			if got := c.IsProduction(); got != tc.want {
				t.Errorf("IsProduction(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

// TestIsDev is the symmetric case for IsDev; the function should be
// case-insensitive but should not accept staging or test as "dev".
func TestIsDev(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"dev", true},
		{"DEV", true},
		{"Dev", true},
		{"test", false},
		{"staging", false},
		{"prod", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			c := &Config{App: AppConfig{Env: tc.env}}
			if got := c.IsDev(); got != tc.want {
				t.Errorf("IsDev(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}
