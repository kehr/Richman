package trend

import (
	"fmt"
	"math"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/datasource"
)

// Calculator computes the trend dimension from historical price data.
type Calculator struct{}

// NewCalculator creates a new trend calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Calculate analyzes price data and returns a TrendResult.
// Requires at least 60 data points for full signal coverage.
func (c *Calculator) Calculate(prices []datasource.PriceData) (analysis.TrendResult, error) {
	if len(prices) < 20 {
		return analysis.TrendResult{}, fmt.Errorf(
			"insufficient price data: need at least 20 points, got %d", len(prices),
		)
	}

	closes := make([]float64, len(prices))
	for i, p := range prices {
		closes[i] = p.Close
	}

	signals := make(map[string]float64)

	// MA crossover signal
	maScore := c.calcMASignal(closes, signals)

	// RSI signal
	rsiScore := 0.0
	if len(closes) >= 15 {
		rsiScore = c.calcRSISignal(closes, signals)
	}

	// MACD signal
	macdScore := 0.0
	if len(closes) >= 35 {
		macdScore = c.calcMACDSignal(closes, signals)
	}

	// Combine scores: MA has highest weight when fewer data points available
	var totalScore float64
	var totalWeight float64

	totalScore += maScore * 0.4
	totalWeight += 0.4

	if len(closes) >= 15 {
		totalScore += rsiScore * 0.3
		totalWeight += 0.3
	}

	if len(closes) >= 35 {
		totalScore += macdScore * 0.3
		totalWeight += 0.3
	}

	normalized := totalScore / totalWeight // -1.0 to 1.0 range

	direction := analysis.DirectionSideways
	if normalized > 0.2 {
		direction = analysis.DirectionUpward
	} else if normalized < -0.2 {
		direction = analysis.DirectionDownward
	}

	strength := math.Abs(normalized)
	if strength > 1.0 {
		strength = 1.0
	}

	summary := buildSummary(direction, strength)

	return analysis.TrendResult{
		Direction: direction,
		Strength:  strength,
		Summary:   summary,
		Signals:   signals,
	}, nil
}

// calcMASignal computes the moving average crossover signal.
// Returns a score from -1.0 (bearish) to 1.0 (bullish).
func (c *Calculator) calcMASignal(closes []float64, signals map[string]float64) float64 {
	n := len(closes)
	ma5 := sma(closes, 5)
	ma20 := sma(closes, 20)

	signals["ma5"] = ma5
	signals["ma20"] = ma20

	if n >= 60 {
		ma60 := sma(closes, 60)
		signals["ma60"] = ma60

		// Full MA alignment: MA5 > MA20 > MA60 = strong bullish
		if ma5 > ma20 && ma20 > ma60 {
			return 1.0
		}
		if ma5 < ma20 && ma20 < ma60 {
			return -1.0
		}
		// Partial alignment
		if ma5 > ma20 {
			return 0.5
		}
		if ma5 < ma20 {
			return -0.5
		}
		return 0.0
	}

	// Only MA5/MA20 available
	if ma5 > ma20 {
		return 0.7
	}
	if ma5 < ma20 {
		return -0.7
	}
	return 0.0
}

// calcRSISignal computes the RSI(14) signal as a trend-following indicator.
// Returns a score from -1.0 (bearish momentum) to 1.0 (bullish momentum).
func (c *Calculator) calcRSISignal(closes []float64, signals map[string]float64) float64 {
	rsi := calcRSI(closes, 14)
	signals["rsi"] = rsi

	// RSI as momentum indicator:
	// > 60: bullish momentum, > 70: strong bullish
	// < 40: bearish momentum, < 30: strong bearish
	// 40-60: neutral
	if rsi > 70 {
		return 1.0
	}
	if rsi > 60 {
		return 0.5
	}
	if rsi < 30 {
		return -1.0
	}
	if rsi < 40 {
		return -0.5
	}
	// Neutral zone: linear interpolation
	return (rsi - 50) / 20.0
}

// calcMACDSignal computes the MACD signal line crossover.
// Returns a score from -1.0 (bearish) to 1.0 (bullish).
func (c *Calculator) calcMACDSignal(closes []float64, signals map[string]float64) float64 {
	macdLine, signalLine, histogram := calcMACD(closes)
	signals["macd"] = macdLine
	signals["macd_signal"] = signalLine
	signals["macd_histogram"] = histogram

	// Use both MACD line direction and histogram for signal
	score := 0.0

	// MACD line above zero = bullish bias
	if macdLine > 0 {
		score += 0.5
	} else if macdLine < 0 {
		score -= 0.5
	}

	// Histogram positive = MACD above signal = bullish momentum
	if histogram > 0 {
		score += 0.5
	} else if histogram < 0 {
		score -= 0.5
	}

	return math.Max(-1.0, math.Min(1.0, score))
}

// sma calculates the simple moving average of the last n values.
func sma(data []float64, period int) float64 {
	if len(data) < period {
		return 0
	}
	sum := 0.0
	start := len(data) - period
	for i := start; i < len(data); i++ {
		sum += data[i]
	}
	return sum / float64(period)
}

// ema calculates the exponential moving average.
func ema(data []float64, period int) float64 {
	if len(data) < period {
		return 0
	}
	multiplier := 2.0 / float64(period+1)
	result := sma(data[:period], period)
	for i := period; i < len(data); i++ {
		result = (data[i]-result)*multiplier + result
	}
	return result
}

// calcRSI computes the Relative Strength Index.
func calcRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50 // neutral default
	}

	gains := 0.0
	losses := 0.0

	for i := 1; i <= period; i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// Smoothed RSI using Wilder's method
	for i := period + 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) - change) / float64(period)
		}
	}

	if avgLoss == 0 {
		if avgGain == 0 {
			return 50 // no movement, neutral
		}
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// calcMACD computes the MACD line, signal line, and histogram.
func calcMACD(closes []float64) (macdLine, signalLine, histogram float64) {
	ema12 := ema(closes, 12)
	ema26 := ema(closes, 26)
	macdLine = ema12 - ema26

	// For signal line, compute MACD series then take EMA(9)
	if len(closes) < 35 {
		return macdLine, 0, macdLine
	}

	macdSeries := make([]float64, 0, len(closes)-26)
	for i := 26; i <= len(closes); i++ {
		slice := closes[:i]
		e12 := ema(slice, 12)
		e26 := ema(slice, 26)
		macdSeries = append(macdSeries, e12-e26)
	}

	if len(macdSeries) >= 9 {
		signalLine = ema(macdSeries, 9)
	}

	histogram = macdLine - signalLine
	return macdLine, signalLine, histogram
}

func buildSummary(dir analysis.Direction, strength float64) string {
	strengthDesc := "weak"
	if strength > 0.7 {
		strengthDesc = "strong"
	} else if strength > 0.4 {
		strengthDesc = "moderate"
	}

	switch dir {
	case analysis.DirectionUpward:
		return fmt.Sprintf("Price trend is %s upward based on technical indicators.", strengthDesc)
	case analysis.DirectionDownward:
		return fmt.Sprintf("Price trend is %s downward based on technical indicators.", strengthDesc)
	default:
		return "Price is moving sideways with no clear directional trend."
	}
}
