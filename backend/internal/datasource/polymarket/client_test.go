package polymarket

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestFetchMarketProbabilitiesMock(t *testing.T) {
	logger := zap.NewNop()
	client := New(logger)

	events, err := client.FetchMarketProbabilities(context.Background(), []string{"fed", "recession"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected non-empty event data")
	}

	for i, e := range events {
		if e.MarketID == "" {
			t.Errorf("event[%d]: MarketID is empty", i)
		}
		if e.Question == "" {
			t.Errorf("event[%d]: Question is empty", i)
		}
		if e.Probability < 0 || e.Probability > 1 {
			t.Errorf("event[%d]: Probability should be in [0, 1], got %f", i, e.Probability)
		}
		if e.Volume <= 0 {
			t.Errorf("event[%d]: Volume should be positive, got %f", i, e.Volume)
		}
		if e.UpdatedAt.IsZero() {
			t.Errorf("event[%d]: UpdatedAt is zero", i)
		}
	}
}

func TestFetchMarketByIDMock(t *testing.T) {
	logger := zap.NewNop()
	client := New(logger)

	event, err := client.FetchMarketByID(context.Background(), "test-market-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if event.MarketID != "test-market-123" {
		t.Errorf("MarketID: got %s, want test-market-123", event.MarketID)
	}
	if event.Probability < 0 || event.Probability > 1 {
		t.Errorf("Probability should be in [0, 1], got %f", event.Probability)
	}
}
