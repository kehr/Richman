import { useQuery } from "@tanstack/react-query";
import { fetchCardById, fetchCardHistory, fetchLatestCards } from "./api";

export function useLatestCards() {
	return useQuery({
		queryKey: ["decision-cards", "latest"],
		queryFn: fetchLatestCards,
		select: (res) => res.data,
	});
}

export function useCardById(id: number) {
	return useQuery({
		queryKey: ["decision-cards", id],
		queryFn: () => fetchCardById(id),
		select: (res) => res.data,
		enabled: id > 0,
	});
}

export function useCardHistory() {
	return useQuery({
		queryKey: ["decision-cards", "history"],
		queryFn: fetchCardHistory,
		select: (res) => res.data,
	});
}
