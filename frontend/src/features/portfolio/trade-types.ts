// Trade types for the holding transactions sub-page (PRD §4.4).
// Mirrors backend/internal/model/trade.go.

export type TradeDirection = "buy" | "sell";

export interface Trade {
	tradeId: number;
	holdingId: number;
	direction: TradeDirection;
	price: number;
	quantity: number;
	tradedAt: string;
}

export interface CreateTradeInput {
	direction: TradeDirection;
	price: number;
	quantity: number;
	tradedAt: string;
}
