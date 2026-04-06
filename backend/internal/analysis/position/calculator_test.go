package position

import (
	"testing"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/datasource"
)

func makeValuationHistory(peRange, pbRange [2]float64, count int) []datasource.ValuationData {
	history := make([]datasource.ValuationData, count)
	peStep := (peRange[1] - peRange[0]) / float64(count-1)
	pbStep := (pbRange[1] - pbRange[0]) / float64(count-1)
	for i := 0; i < count; i++ {
		history[i] = datasource.ValuationData{
			Date: time.Now().AddDate(0, 0, -count+i),
			PE:   peRange[0] + float64(i)*peStep,
			PB:   pbRange[0] + float64(i)*pbStep,
		}
	}
	return history
}

func TestAShareLowValuation(t *testing.T) {
	calc := NewCalculator()
	// History PE ranges from 10 to 30, current PE=12 should be low percentile
	history := makeValuationHistory([2]float64{10, 30}, [2]float64{1.0, 3.0}, 20)
	current := &datasource.ValuationData{PE: 12, PB: 1.2}

	result, err := calc.Calculate("a_share_broad", current, history, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Assessment != analysis.DirectionBullish {
		t.Errorf("expected bullish (undervalued), got %s", result.Assessment)
	}
	if result.Percentile >= 0.3 {
		t.Errorf("expected percentile < 0.3 for low valuation, got %f", result.Percentile)
	}
}

func TestAShareHighValuation(t *testing.T) {
	calc := NewCalculator()
	history := makeValuationHistory([2]float64{10, 30}, [2]float64{1.0, 3.0}, 20)
	current := &datasource.ValuationData{PE: 28, PB: 2.8}

	result, err := calc.Calculate("a_share_broad", current, history, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Assessment != analysis.DirectionBearish {
		t.Errorf("expected bearish (overvalued), got %s", result.Assessment)
	}
	if result.Percentile <= 0.7 {
		t.Errorf("expected percentile > 0.7 for high valuation, got %f", result.Percentile)
	}
}

func TestAShareFairValuation(t *testing.T) {
	calc := NewCalculator()
	history := makeValuationHistory([2]float64{10, 30}, [2]float64{1.0, 3.0}, 20)
	current := &datasource.ValuationData{PE: 20, PB: 2.0}

	result, err := calc.Calculate("a_share_broad", current, history, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Assessment != analysis.DirectionNeutral {
		t.Errorf("expected neutral (fair), got %s", result.Assessment)
	}
}

func TestUSStockWithCAPE(t *testing.T) {
	calc := NewCalculator()
	history := make([]datasource.ValuationData, 20)
	for i := 0; i < 20; i++ {
		history[i] = datasource.ValuationData{
			Date: time.Now().AddDate(0, 0, -20+i),
			CAPE: 15 + float64(i),
			PE:   10 + float64(i),
		}
	}
	// High CAPE should yield bearish
	current := &datasource.ValuationData{CAPE: 33, PE: 25}

	result, err := calc.Calculate("us_stock", current, history, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Metrics["cape"] != 33 {
		t.Error("expected CAPE metric in result")
	}
	if result.Assessment != analysis.DirectionBearish {
		t.Errorf("expected bearish for high CAPE, got %s", result.Assessment)
	}
}

func TestUSStockFallbackToPE(t *testing.T) {
	calc := NewCalculator()
	history := make([]datasource.ValuationData, 20)
	for i := 0; i < 20; i++ {
		history[i] = datasource.ValuationData{
			Date: time.Now().AddDate(0, 0, -20+i),
			PE:   10 + float64(i),
		}
	}
	// No CAPE, low PE should yield bullish
	current := &datasource.ValuationData{PE: 11}

	result, err := calc.Calculate("us_stock", current, history, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result.Metrics["pe_percentile"]; !ok {
		t.Error("expected pe_percentile metric when CAPE unavailable")
	}
}

func TestGoldPosition(t *testing.T) {
	calc := NewCalculator()
	prices := make([]datasource.PriceData, 100)
	for i := 0; i < 100; i++ {
		prices[i] = datasource.PriceData{
			Date:  time.Now().AddDate(0, 0, -100+i),
			Close: 1800 + float64(i)*2,
		}
	}

	// Current price is the last one (highest), should be high percentile
	result, err := calc.Calculate("gold_etf", nil, nil, prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Percentile < 0.9 {
		t.Errorf("expected high percentile for max price, got %f", result.Percentile)
	}
}

func TestInsufficientData(t *testing.T) {
	calc := NewCalculator()

	_, err := calc.Calculate("a_share_broad", &datasource.ValuationData{PE: 15}, nil, nil)
	if err == nil {
		t.Error("expected error for insufficient history")
	}

	_, err = calc.Calculate("gold_etf", nil, nil, []datasource.PriceData{{Close: 1800}})
	if err == nil {
		t.Error("expected error for insufficient gold price data")
	}
}

func TestUnsupportedAssetType(t *testing.T) {
	calc := NewCalculator()
	_, err := calc.Calculate("crypto", nil, nil, nil)
	if err == nil {
		t.Error("expected error for unsupported asset type")
	}
}

func TestPercentile(t *testing.T) {
	dataset := []float64{10, 20, 30, 40, 50}

	tests := []struct {
		value float64
		want  float64
	}{
		{5, 0.0},  // below all
		{55, 1.0}, // above all
		{30, 0.4}, // 2 out of 5 below
		{10, 0.0}, // none below
		{50, 0.8}, // 4 out of 5 below
	}

	for _, tt := range tests {
		got := calcPercentile(tt.value, dataset)
		if got != tt.want {
			t.Errorf("calcPercentile(%f, dataset) = %f, want %f", tt.value, got, tt.want)
		}
	}
}
