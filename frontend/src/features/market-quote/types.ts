// Market quote types for the real-time price panel.

export interface QuoteCurrentDTO {
	price: number;
	date: string;
	changeAbs: number;
	changePct: number;
}

export interface QuoteHistoryPoint {
	date: string;
	open: number;
	high: number;
	low: number;
	close: number;
	volume: number;
}

export interface AssetQuoteDTO {
	assetCode: string;
	assetType: string;
	source: string;
	fetchedAt: string;
	current: QuoteCurrentDTO | null;
	history: QuoteHistoryPoint[];
}

// Chart rendering primitives (business-agnostic).

export interface PriceLine {
	price: number;
	color: string;
	lineStyle: "solid" | "dashed";
	label: string;
}

export interface TimeMarker {
	time: string;
	label: string;
	color: string;
}
