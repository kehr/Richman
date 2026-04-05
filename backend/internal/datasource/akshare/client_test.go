package akshare

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestFetchPriceHistoryMock(t *testing.T) {
	logger := zap.NewNop()
	client := New("http://localhost:8888", logger)

	prices, err := client.FetchPriceHistory(context.Background(), "000300", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(prices) == 0 {
		t.Fatal("expected non-empty price data")
	}

	// Verify data structure integrity.
	for i, p := range prices {
		if p.Date.IsZero() {
			t.Errorf("price[%d]: date is zero", i)
		}
		if p.High < p.Low {
			t.Errorf("price[%d]: high (%f) < low (%f)", i, p.High, p.Low)
		}
		if p.Volume <= 0 {
			t.Errorf("price[%d]: volume should be positive, got %f", i, p.Volume)
		}
	}

	// Verify chronological order.
	for i := 1; i < len(prices); i++ {
		if prices[i].Date.Before(prices[i-1].Date) {
			t.Errorf("prices not in chronological order at index %d", i)
		}
	}
}

func TestFetchValuationMock(t *testing.T) {
	logger := zap.NewNop()
	client := New("http://localhost:8888", logger)

	val, err := client.FetchValuation(context.Background(), "000300")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val == nil {
		t.Fatal("expected non-nil valuation")
	}

	if val.PE <= 0 {
		t.Errorf("PE should be positive, got %f", val.PE)
	}
	if val.PB <= 0 {
		t.Errorf("PB should be positive, got %f", val.PB)
	}
	if val.CAPE <= 0 {
		t.Errorf("CAPE should be positive, got %f", val.CAPE)
	}
	if val.DividendYield <= 0 || val.DividendYield >= 1 {
		t.Errorf("DividendYield should be between 0 and 1, got %f", val.DividendYield)
	}
}

func TestFetchPriceHistoryDefaultDays(t *testing.T) {
	logger := zap.NewNop()
	client := New("http://localhost:8888", logger)

	// Pass 0 days, should default to 30.
	prices, err := client.FetchPriceHistory(context.Background(), "000300", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prices) == 0 {
		t.Fatal("expected non-empty price data for default days")
	}
}
