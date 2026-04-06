import { useQuery } from "@tanstack/react-query";
import { fetchDashboardStats } from "./api";

export function useStats() {
	return useQuery({
		queryKey: ["dashboard", "stats"],
		queryFn: fetchDashboardStats,
		select: (res) => res.data,
	});
}
