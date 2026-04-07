import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createTrade, fetchTrades } from "./api";
import type { CreateTradeInput, Trade, TradeDirection } from "./trade-types";

// useHoldingTrades fetches the list of trades for a given holding. The
// generic Trade shape mirrors backend/internal/model/trade.go and is
// re-exported from the feature barrel. We coerce the raw direction string
// into the TradeDirection union so consumers can rely on a discriminated
// union without revalidating the response.
export function useHoldingTrades(holdingId: number) {
	return useQuery({
		queryKey: ["holding-trades", holdingId],
		queryFn: () => fetchTrades(holdingId),
		select: (res): Trade[] =>
			(res.data ?? []).map((t) => ({
				tradeId: t.tradeId,
				holdingId: t.holdingId,
				direction: t.direction as TradeDirection,
				price: t.price,
				quantity: t.quantity,
				tradedAt: t.tradedAt,
			})),
		enabled: holdingId > 0,
	});
}

export function useCreateHoldingTrade(holdingId: number) {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateTradeInput) => createTrade(holdingId, data),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["holding-trades", holdingId] });
			qc.invalidateQueries({ queryKey: ["holdings"] });
		},
	});
}
