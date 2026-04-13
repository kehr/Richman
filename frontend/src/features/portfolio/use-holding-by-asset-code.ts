// useHoldingByAssetCode finds the first holding matching a given asset code.
// Returns undefined when the user is not logged in or has no matching holding.
// Depends on the holdings query being pre-populated; does not issue a separate request.

import type { HoldingDto } from "./api";
import { useHoldings } from "./usePortfolio";

export function useHoldingByAssetCode(assetCode: string): {
	data: HoldingDto | undefined;
	isLoading: boolean;
} {
	const { data: holdings, isLoading } = useHoldings();
	const holding = holdings?.find((h) => h.assetCode === assetCode);
	return { data: holding, isLoading };
}
