import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	createHolding,
	createTrade,
	deleteHolding,
	fetchHoldings,
	fetchTrades,
	updateHolding,
} from "./api";
import type { CreateHoldingInput } from "./api";
import type { CreateTradeInput, Trade, TradeDirection } from "./trade-types";

export function useHoldings() {
	return useQuery({
		queryKey: ["holdings"],
		queryFn: fetchHoldings,
		select: (res) => res.data,
	});
}

export function useCreateHolding() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateHoldingInput) => createHolding(data),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["holdings"] });
			qc.invalidateQueries({ queryKey: ["dashboard"] });
		},
	});
}

export function useUpdateHolding() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({ id, data }: { id: number; data: Partial<CreateHoldingInput> }) =>
			updateHolding(id, data),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["holdings"] });
		},
	});
}

export function useDeleteHolding() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: number) => deleteHolding(id),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["holdings"] });
			qc.invalidateQueries({ queryKey: ["dashboard"] });
		},
	});
}

// useTrades fetches the trade history for a holding. The raw backend
// direction string is narrowed to the TradeDirection union here so consumers
// can rely on a discriminated union without revalidating each row.
export function useTrades(holdingId: number) {
	return useQuery({
		queryKey: ["trades", holdingId],
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

export function useCreateTrade(holdingId: number) {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateTradeInput) => createTrade(holdingId, data),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["trades", holdingId] });
			qc.invalidateQueries({ queryKey: ["holdings"] });
		},
	});
}
