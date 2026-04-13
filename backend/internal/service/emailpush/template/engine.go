package template

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// DailyBriefingData holds template variables for daily_briefing_{zh,en}.html.
type DailyBriefingData struct {
	RegimeLabel    string
	GoldScore      float64
	GoldScoreDelta float64
	Events         []EventSummary
	Holdings       []HoldingSummary
	HasHoldings    bool
	UnsubscribeURL string
	Disclaimer     string
}

// WeeklyInsightData holds template variables for weekly_insight_{zh,en}.html.
type WeeklyInsightData struct {
	Title          string
	Sections       []InsightSection
	UnsubscribeURL string
	Disclaimer     string
}

// MarketAlertData holds template variables for market_alert_{zh,en}.html.
type MarketAlertData struct {
	EventTitle      string
	PrevProbability float64
	CurrProbability float64
	Delta           float64
	GoldDirection   string
	UnsubscribeURL  string
	Disclaimer      string
}

// HoldingSuggestionData holds template variables for holding_suggestion_{zh,en}.html.
type HoldingSuggestionData struct {
	AssetName      string
	AssetCode      string
	ActionAdvice   string
	DetailedAdvice string
	RiskWarnings   []string
	Confidence     float64
	UnsubscribeURL string
	Disclaimer     string
}

// EventSummary is a single market/macro event shown in the daily briefing.
type EventSummary struct {
	Title         string
	GoldDirection string
	Impact        string
}

// HoldingSummary is a single holding entry shown in the daily briefing.
type HoldingSummary struct {
	AssetName    string
	AssetCode    string
	ActionAdvice string
	Confidence   float64
}

// InsightSection is a single section within the weekly insight email.
type InsightSection struct {
	Heading string
	Content string
}

// Engine loads HTML templates from disk and renders them on demand.
type Engine struct {
	templates map[string]*htmltemplate.Template
	logger    *zap.Logger
}

// NewEngine loads all *.html files from templateDir and returns an Engine.
// It returns an error if any template file fails to parse.
func NewEngine(templateDir string, logger *zap.Logger) (*Engine, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, fmt.Errorf("read template dir %s: %w", templateDir, err)
	}

	templates := make(map[string]*htmltemplate.Template)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".html") {
			continue
		}

		path := filepath.Join(templateDir, name)
		tmpl, err := htmltemplate.ParseFiles(path)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}

		key := strings.TrimSuffix(name, ".html")
		templates[key] = tmpl
		logger.Debug("loaded email template", zap.String("name", key))
	}

	if len(templates) == 0 {
		logger.Warn("no email templates loaded", zap.String("dir", templateDir))
	}

	return &Engine{
		templates: templates,
		logger:    logger,
	}, nil
}

// Render executes the named template with data and returns the rendered HTML.
// templateName is the filename without the .html extension (e.g. "daily_briefing_zh").
func (e *Engine) Render(templateName string, data any) (string, error) {
	tmpl, ok := e.templates[templateName]
	if !ok {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", templateName, err)
	}

	return buf.String(), nil
}
