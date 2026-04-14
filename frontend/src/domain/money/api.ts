import { requestV1 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { DisplayCurrency } from "@/features/user-settings";

export interface ExchangeRatesData {
	rates: Partial<Record<DisplayCurrency, number>>;
	updatedAt: string;
}

export function getExchangeRates(): Promise<ApiResponse<ExchangeRatesData>> {
	return requestV1<ApiResponse<ExchangeRatesData>>("/exchange-rates");
}
