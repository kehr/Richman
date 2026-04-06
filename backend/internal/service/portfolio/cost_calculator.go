package portfolio

import (
	"github.com/richman/backend/internal/model"
	"github.com/shopspring/decimal"
)

// CostResult holds the result of a cost recalculation.
type CostResult struct {
	CostPrice     decimal.Decimal
	TotalQuantity decimal.Decimal
}

// RecalculateCost computes the weighted average cost price and total quantity
// from a base position plus a list of trades.
// For buy trades: new_cost = (old_cost * old_qty + price * qty) / (old_qty + qty)
// For sell trades: quantity is reduced, cost stays unchanged.
func RecalculateCost(baseCost, baseQty decimal.Decimal, trades []model.Trade) CostResult {
	cost := baseCost
	qty := baseQty

	for i := range trades {
		switch trades[i].Direction {
		case "buy":
			totalValue := cost.Mul(qty).Add(
				trades[i].Price.Mul(trades[i].Quantity),
			)
			qty = qty.Add(trades[i].Quantity)
			if qty.IsPositive() {
				cost = totalValue.Div(qty)
			}
		case "sell":
			qty = qty.Sub(trades[i].Quantity)
			if qty.IsNegative() {
				qty = decimal.Zero
			}
		}
	}

	return CostResult{
		CostPrice:     cost,
		TotalQuantity: qty,
	}
}
