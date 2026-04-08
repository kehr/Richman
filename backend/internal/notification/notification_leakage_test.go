package notification

import (
	"testing"

	"github.com/richman/backend/internal/notification/adapter"
	usersettings "github.com/richman/backend/internal/service/user_settings"
)

// TestAdapterMessage_NoCapitalLeakage enforces TRD §5.2 compile-time
// constraint #3: notification adapter inputs must not carry absolute
// capital/amount fields. The dispatcher feeds adapter.Message into every
// channel adapter, so guarding that single type covers the entire push
// surface.
//
// If a future change adds a field with a forbidden json tag (e.g. embedding
// a DecisionCard payload directly), this test will fail and the offending
// field must either be renamed, marked json:"-", or projected via
// PublicCardSummary before reaching the adapter.
func TestAdapterMessage_NoCapitalLeakage(t *testing.T) {
	if err := usersettings.AssertNoCapitalLeakage(adapter.Message{}); err != nil {
		t.Fatalf("adapter.Message leaks capital info: %v", err)
	}
}
