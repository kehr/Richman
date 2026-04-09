package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/richman/backend/internal/model"
)

// TestLLMConfigRepo_Upsert_NilGuard locks the defensive nil-cfg branch so a
// future refactor cannot silently remove the early return and start feeding
// nil pointers into the pgx call path.
func TestLLMConfigRepo_Upsert_NilGuard(t *testing.T) {
	// pool is nil — we never reach it because the early return fires first.
	r := &LLMConfigRepo{pool: nil}

	err := r.Upsert(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on nil cfg, got nil")
	}
	if !containsErr(err, "cfg is nil") {
		t.Errorf("expected nil cfg error, got %q", err.Error())
	}
}

// TestLLMConfigRepo_Upsert_MissingUserIDGuard locks the invariant that
// Upsert refuses to persist a config without a user_id so an orphaned row
// cannot slip past the NOT NULL + partial index contract at the DB level.
func TestLLMConfigRepo_Upsert_MissingUserIDGuard(t *testing.T) {
	r := &LLMConfigRepo{pool: nil}

	cfg := &model.LLMConfig{
		// UserID intentionally left zero.
		ProviderType: model.ProviderClaude,
		Model:        "claude-sonnet-4-6",
	}
	err := r.Upsert(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error on missing user_id, got nil")
	}
	if !containsErr(err, "user_id is required") {
		t.Errorf("expected missing user_id error, got %q", err.Error())
	}
}

// containsErr walks the errors.Unwrap chain and reports whether any level
// contains the given substring. Kept local so the repo test file does not
// drag in a test helper package.
func containsErr(err error, substr string) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}
