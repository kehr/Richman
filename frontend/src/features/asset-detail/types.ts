// types.ts — data contracts for the asset detail feature.
// All shapes mirror the backend richson API v2 response bodies.

export interface DimensionSubIndicator {
	name: string;
	rawValue: number | string;
	percentile: number | null;
	normalizedScore: number;
	weight: number;
}

export interface DimensionDetailDto {
	id: string; // "d1" | "d2" | "d3" | "d4"
	name: string;
	score: number;
	quantScore: number | null; // base quantitative score before LLM adjustment
	llmAdjustment: number | null; // delta applied by LLM
	signal: string; // "bullish" | "neutral" | "bearish"
	weight: number;
	summary: string;
	llmReason: string | null;
	subIndicators: DimensionSubIndicator[];
}

export interface MajorChangeRecapDto {
	text: string;
	scoreDelta: number;
	previousScore: number;
	currentScore: number;
}

export interface RiskFactorDto {
	id: string;
	description: string;
	severity: "high" | "medium" | "low";
}

export interface KeyPriceLevelDto {
	type: "support" | "resistance";
	price: number;
	distancePct: number;
	currency: string;
}

export interface DrawdownReferenceDto {
	currentBullMaxDrawdown: number;
	currentBullMaxDrawdownDate: string;
	historicalAvgDrawdown: number;
}

export interface ExecutionScenarioDto {
	id: string;
	priority: number;
	condition: string;
	action: string;
	rationale: string;
}

export interface ExecutionPlanDto {
	recommendation: string;
	defaultAdvice: string;
	stopLoss: number | null;
	takeProfit: number | null;
	validDays: number;
	concentrationWarning: string | null;
	scenarios: ExecutionScenarioDto[];
	disclaimer: string;
}

export interface AssetDetailDto {
	code: string;
	name: string;
	nameEn: string;
	assetType: string;
	exchange: string;
	currency: "USD" | "CNY";
	usdExchangeRate: number | null;
	// current price from latest data
	currentPrice: number;
	priceChangePercent: number;
	priceAtAnalysis: number;
	// analysis fields
	overallScore: number;
	scoreBandLow: number;
	scoreBandHigh: number;
	signalLevel: string; // "strong_bullish" | "bullish" | "neutral" | "bearish" | "strong_bearish"
	percentileLabel: string; // "veryHigh" | "high" | "mid" | "low" | "veryLow"
	marketInterpretation: string;
	scoreDelta: number;
	changeSummary: string | null;
	majorChangeRecap: MajorChangeRecapDto | null;
	conflictType: string | null;
	conflictMessage: string | null;
	analyzedAt: string;
	validDays: number;
	dimensions: DimensionDetailDto[];
	riskFactors: RiskFactorDto[];
	keyPriceLevels: KeyPriceLevelDto[];
	drawdownReference: DrawdownReferenceDto | null;
	executionPlan: ExecutionPlanDto | null;
	supports: number[];
	resistances: number[];
	sma200: number | null;
}

export interface OhlcvBarDto {
	time: string; // "YYYY-MM-DD"
	open: number;
	high: number;
	low: number;
	close: number;
	volume: number;
}

export interface OhlcvDto {
	code: string;
	period: string;
	bars: OhlcvBarDto[];
}

export interface ScoreHistoryPointDto {
	date: string;
	score: number;
	versionChange: boolean;
	versionLabel: string | null;
}

export interface ScoreHistoryDto {
	code: string;
	days: number;
	points: ScoreHistoryPointDto[];
}

export interface DemoPlanDto {
	holdingCode: string;
	disclaimer: string;
	executionPlan: ExecutionPlanDto;
}

export interface TriggerAnalysisResponseDto {
	jobId: string;
}

export type AnalysisJobStatus = "pending" | "running" | "completed" | "failed";

export interface AnalysisJobDto {
	jobId: string;
	status: AnalysisJobStatus;
	progress: number;
	currentStep: string | null;
	errorMessage: string | null;
	completedAt: string | null;
}
