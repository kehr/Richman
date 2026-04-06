package trend

import (
	"testing"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/datasource"
)

func generatePrices(basePrice float64, count int, trend string) []datasource.PriceData {
	prices := make([]datasource.PriceData, count)
	price := basePrice
	for i := 0; i < count; i++ {
		switch trend {
		case "up":
			price *= 1.02
		case "down":
			price *= 0.98
		case "sideways":
			// Stay flat at base price
			price = basePrice
		}
		prices[i] = datasource.PriceData{
			Date:   time.Now().AddDate(0, 0, -count+i),
			Open:   price * 0.999,
			High:   price * 1.002,
			Low:    price * 0.998,
			Close:  price,
			Volume: 1000000,
		}
	}
	return prices
}

func TestCalculateUptrend(t *testing.T) {
	calc := NewCalculator()
	prices := generatePrices(100, 70, "up")

	result, err := calc.Calculate(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Direction != analysis.DirectionUpward {
		t.Errorf("expected upward direction, got %s", result.Direction)
	}
	if result.Strength < 0.3 {
		t.Errorf("expected strength > 0.3 for clear uptrend, got %f", result.Strength)
	}
	if result.Signals["ma5"] <= result.Signals["ma20"] {
		t.Errorf("expected MA5 > MA20 in uptrend, got MA5=%f MA20=%f", result.Signals["ma5"], result.Signals["ma20"])
	}
}

func TestCalculateDowntrend(t *testing.T) {
	calc := NewCalculator()
	prices := generatePrices(200, 70, "down")

	result, err := calc.Calculate(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Direction != analysis.DirectionDownward {
		t.Errorf("expected downward direction, got %s", result.Direction)
	}
	if result.Strength < 0.3 {
		t.Errorf("expected strength > 0.3 for clear downtrend, got %f", result.Strength)
	}
}

func TestCalculateSideways(t *testing.T) {
	calc := NewCalculator()
	prices := generatePrices(100, 70, "sideways")

	result, err := calc.Calculate(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Direction != analysis.DirectionSideways {
		t.Errorf("expected sideways direction, got %s", result.Direction)
	}
}

func TestCalculateInsufficientData(t *testing.T) {
	calc := NewCalculator()
	prices := generatePrices(100, 10, "up")

	_, err := calc.Calculate(prices)
	if err == nil {
		t.Error("expected error for insufficient data")
	}
}

func TestCalculateMinimalData(t *testing.T) {
	calc := NewCalculator()
	prices := generatePrices(100, 20, "up")

	result, err := calc.Calculate(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still produce a result with at least MA signals
	if _, ok := result.Signals["ma5"]; !ok {
		t.Error("expected ma5 signal in result")
	}
	if _, ok := result.Signals["ma20"]; !ok {
		t.Error("expected ma20 signal in result")
	}
}

func TestSMA(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	got := sma(data, 5)
	want := 3.0
	if got != want {
		t.Errorf("sma([1,2,3,4,5], 5) = %f, want %f", got, want)
	}
}

func TestRSI(t *testing.T) {
	// Generate a simple rising sequence; RSI should be > 50
	data := make([]float64, 30)
	for i := range data {
		data[i] = float64(100 + i)
	}
	rsi := calcRSI(data, 14)
	if rsi <= 50 {
		t.Errorf("RSI for rising sequence should be > 50, got %f", rsi)
	}

	// Generate a falling sequence; RSI should be < 50
	for i := range data {
		data[i] = float64(200 - i)
	}
	rsi = calcRSI(data, 14)
	if rsi >= 50 {
		t.Errorf("RSI for falling sequence should be < 50, got %f", rsi)
	}
}
