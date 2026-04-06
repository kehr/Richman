package portfolio

import (
	"testing"
	"time"

	"github.com/richman/backend/internal/model"
	"github.com/shopspring/decimal"
)

func d(v string) decimal.Decimal {
	return decimal.RequireFromString(v)
}

func TestRecalculateCost_BuyOnly(t *testing.T) {
	trades := []model.Trade{
		{Direction: "buy", Price: d("10"), Quantity: d("100"), TradedAt: time.Now()},
		{Direction: "buy", Price: d("12"), Quantity: d("100"), TradedAt: time.Now()},
	}

	result := RecalculateCost(decimal.Zero, decimal.Zero, trades)

	// (0 + 10*100 + 12*100) / (0+100+100) = 2200 / 200 = 11
	if !result.CostPrice.Equal(d("11")) {
		t.Errorf("expected cost 11, got %s", result.CostPrice)
	}
	if !result.TotalQuantity.Equal(d("200")) {
		t.Errorf("expected quantity 200, got %s", result.TotalQuantity)
	}
}

func TestRecalculateCost_BuySellMixed(t *testing.T) {
	trades := []model.Trade{
		{Direction: "buy", Price: d("10"), Quantity: d("100"), TradedAt: time.Now()},
		{Direction: "buy", Price: d("20"), Quantity: d("100"), TradedAt: time.Now()},
		{Direction: "sell", Price: d("25"), Quantity: d("50"), TradedAt: time.Now()},
	}

	result := RecalculateCost(decimal.Zero, decimal.Zero, trades)

	// After 2 buys: cost=15, qty=200
	// After sell 50: cost=15, qty=150
	if !result.CostPrice.Equal(d("15")) {
		t.Errorf("expected cost 15, got %s", result.CostPrice)
	}
	if !result.TotalQuantity.Equal(d("150")) {
		t.Errorf("expected quantity 150, got %s", result.TotalQuantity)
	}
}

func TestRecalculateCost_BaseCostWithTrades(t *testing.T) {
	// Simulate quick mode: user set base cost and quantity, then adds trades.
	baseCost := d("10")
	baseQty := d("100")

	trades := []model.Trade{
		{Direction: "buy", Price: d("20"), Quantity: d("100"), TradedAt: time.Now()},
	}

	result := RecalculateCost(baseCost, baseQty, trades)

	// (10*100 + 20*100) / (100+100) = 3000/200 = 15
	if !result.CostPrice.Equal(d("15")) {
		t.Errorf("expected cost 15, got %s", result.CostPrice)
	}
	if !result.TotalQuantity.Equal(d("200")) {
		t.Errorf("expected quantity 200, got %s", result.TotalQuantity)
	}
}

func TestRecalculateCost_SellAllShares(t *testing.T) {
	trades := []model.Trade{
		{Direction: "buy", Price: d("10"), Quantity: d("100"), TradedAt: time.Now()},
		{Direction: "sell", Price: d("15"), Quantity: d("100"), TradedAt: time.Now()},
	}

	result := RecalculateCost(decimal.Zero, decimal.Zero, trades)

	if !result.TotalQuantity.Equal(decimal.Zero) {
		t.Errorf("expected quantity 0, got %s", result.TotalQuantity)
	}
}

func TestRecalculateCost_SellMoreThanOwned(t *testing.T) {
	trades := []model.Trade{
		{Direction: "buy", Price: d("10"), Quantity: d("50"), TradedAt: time.Now()},
		{Direction: "sell", Price: d("15"), Quantity: d("100"), TradedAt: time.Now()},
	}

	result := RecalculateCost(decimal.Zero, decimal.Zero, trades)

	// Quantity should clamp to zero
	if !result.TotalQuantity.Equal(decimal.Zero) {
		t.Errorf("expected quantity 0, got %s", result.TotalQuantity)
	}
}
