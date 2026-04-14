import { requestV1 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { AssetQuoteDTO } from "./types";

// fetchAssetQuote loads the latest quote and 45-day history for an asset.
export async function fetchAssetQuote(
	assetType: string,
	assetCode: string,
): Promise<ApiResponse<AssetQuoteDTO>> {
	return requestV1<ApiResponse<AssetQuoteDTO>>(`/quotes/${assetType}/${assetCode}`);
}
