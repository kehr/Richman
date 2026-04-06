package position

import (
	"fmt"
	"sort"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/datasource"
)

// Calculator computes the position (valuation) dimension.
type Calculator struct{}

// NewCalculator creates a new position calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Calculate analyzes valuation data and price history to produce a PositionResult.
func (c *Calculator) Calculate(
	assetType string,
	current *datasource.ValuationData,
	history []datasource.ValuationData,
	prices []datasource.PriceData,
) (analysis.PositionResult, error) {
	switch assetType {
	case "a_share_broad", "a_share_industry":
		return c.calcASharePosition(current, history)
	case "us_stock":
		return c.calcUSStockPosition(current, history)
	case "gold_etf":
		return c.calcGoldPosition(prices)
	default:
		return analysis.PositionResult{}, fmt.Errorf("unsupported asset type: %s", assetType)
	}
}

// calcASharePosition uses PE/PB historical percentile for A-share assets.
func (c *Calculator) calcASharePosition(
	current *datasource.ValuationData,
	history []datasource.ValuationData,
) (analysis.PositionResult, error) {
	if current == nil {
		return analysis.PositionResult{},
			fmt.Errorf("current valuation data is required for A-share position")
	}
	if len(history) < 5 {
		return analysis.PositionResult{}, fmt.Errorf(
			"insufficient valuation history: need at least 5 points, got %d",
			len(history),
		)
	}

	metrics := make(map[string]float64)
	metrics["pe"] = current.PE
	metrics["pb"] = current.PB

	pePercentile := calcPercentile(current.PE, extractPE(history))
	pbPercentile := calcPercentile(current.PB, extractPB(history))

	metrics["pe_percentile"] = pePercentile
	metrics["pb_percentile"] = pbPercentile

	// Weighted average: PE 60%, PB 40%
	percentile := pePercentile*0.6 + pbPercentile*0.4

	return buildPositionResult(percentile, metrics), nil
}

// calcUSStockPosition uses CAPE if available, fallback to PE.
func (c *Calculator) calcUSStockPosition(
	current *datasource.ValuationData,
	history []datasource.ValuationData,
) (analysis.PositionResult, error) {
	if current == nil {
		return analysis.PositionResult{},
			fmt.Errorf("current valuation data is required for US stock position")
	}
	if len(history) < 5 {
		return analysis.PositionResult{}, fmt.Errorf(
			"insufficient valuation history: need at least 5 points, got %d",
			len(history),
		)
	}

	metrics := make(map[string]float64)

	var percentile float64
	if current.CAPE > 0 {
		capePercentile := calcPercentile(current.CAPE, extractCAPE(history))
		metrics["cape"] = current.CAPE
		metrics["cape_percentile"] = capePercentile
		percentile = capePercentile
	} else {
		pePercentile := calcPercentile(current.PE, extractPE(history))
		metrics["pe"] = current.PE
		metrics["pe_percentile"] = pePercentile
		percentile = pePercentile
	}

	return buildPositionResult(percentile, metrics), nil
}

// calcGoldPosition uses price percentile in the available price range.
func (c *Calculator) calcGoldPosition(prices []datasource.PriceData) (analysis.PositionResult, error) {
	if len(prices) < 5 {
		return analysis.PositionResult{}, fmt.Errorf(
			"insufficient price data for gold position: need at least 5 points, got %d",
			len(prices),
		)
	}

	currentPrice := prices[len(prices)-1].Close
	allPrices := make([]float64, len(prices))
	for i, p := range prices {
		allPrices[i] = p.Close
	}

	percentile := calcPercentile(currentPrice, allPrices)

	metrics := make(map[string]float64)
	metrics["current_price"] = currentPrice
	metrics["price_percentile"] = percentile

	return buildPositionResult(percentile, metrics), nil
}

// calcPercentile returns the percentile rank of value within the dataset (0.0-1.0).
func calcPercentile(value float64, dataset []float64) float64 {
	if len(dataset) == 0 {
		return 0.5
	}

	sorted := make([]float64, len(dataset))
	copy(sorted, dataset)
	sort.Float64s(sorted)

	count := 0
	for _, v := range sorted {
		if v < value {
			count++
		}
	}

	return float64(count) / float64(len(sorted))
}

func buildPositionResult(
	percentile float64,
	metrics map[string]float64,
) analysis.PositionResult {
	var assessment analysis.Direction
	var summary string

	switch {
	case percentile < 0.3:
		assessment = analysis.DirectionBullish
		summary = fmt.Sprintf(
			"Undervalued at %.0f%% historical percentile, favorable entry point.",
			percentile*100,
		)
	case percentile > 0.7:
		assessment = analysis.DirectionBearish
		summary = fmt.Sprintf(
			"Overvalued at %.0f%% historical percentile, elevated risk.",
			percentile*100,
		)
	default:
		assessment = analysis.DirectionNeutral
		summary = fmt.Sprintf(
			"Fair valuation at %.0f%% historical percentile.",
			percentile*100,
		)
	}

	return analysis.PositionResult{
		Assessment: assessment,
		Percentile: percentile,
		Summary:    summary,
		Metrics:    metrics,
	}
}

func extractPE(data []datasource.ValuationData) []float64 {
	vals := make([]float64, 0, len(data))
	for _, d := range data {
		if d.PE > 0 {
			vals = append(vals, d.PE)
		}
	}
	return vals
}

func extractPB(data []datasource.ValuationData) []float64 {
	vals := make([]float64, 0, len(data))
	for _, d := range data {
		if d.PB > 0 {
			vals = append(vals, d.PB)
		}
	}
	return vals
}

func extractCAPE(data []datasource.ValuationData) []float64 {
	vals := make([]float64, 0, len(data))
	for _, d := range data {
		if d.CAPE > 0 {
			vals = append(vals, d.CAPE)
		}
	}
	return vals
}
