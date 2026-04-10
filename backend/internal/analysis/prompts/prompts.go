// Package prompts embeds and exposes LLM prompt templates for the analysis pipeline.
// All templates are embedded at compile time via go:embed so the binary is self-contained.
// Templates use standard Go text/template syntax; data structs are defined below.
package prompts

import (
	_ "embed"
	"fmt"
	"sort"
	"text/template"

	"bytes"
)

//go:embed catalyst_system.txt
var catalystSystemSrc string

//go:embed catalyst_user.tmpl
var catalystUserSrc string

//go:embed synthesis_system.tmpl
var synthesisSystemSrc string

//go:embed synthesis_user.tmpl
var synthesisUserSrc string

//go:embed synthesis_recommendation.tmpl
var synthesisRecommendationSrc string

var (
	catalystUserTmpl       *template.Template
	synthesisSystemTmpl    *template.Template
	synthesisUserTmpl      *template.Template
	synthesisRecommendTmpl *template.Template
)

func init() {
	// All templates are parsed at package initialization. template.Must panics on
	// syntax errors so misconfigured templates are caught immediately at startup.
	parse := func(name, src string) *template.Template {
		return template.Must(template.New(name).Parse(src))
	}
	catalystUserTmpl = parse("catalyst_user", catalystUserSrc)
	synthesisSystemTmpl = parse("synthesis_system", synthesisSystemSrc)
	synthesisUserTmpl = parse("synthesis_user", synthesisUserSrc)
	synthesisRecommendTmpl = parse("synthesis_recommendation", synthesisRecommendationSrc)
}

// render executes a template with data and returns the rendered string.
// A render error is a programming mistake (nil data, wrong field names), not a
// runtime condition — it returns an error so callers can fall back gracefully.
func render(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render prompt %q: %w", tmpl.Name(), err)
	}
	return buf.String(), nil
}

// NamedValue is a key-value pair for signal/metric maps, used in templates
// to produce deterministic ordering (maps have non-deterministic iteration).
type NamedValue struct {
	Name  string
	Value float64
}

// SortedPairs converts a map[string]float64 to a sorted []NamedValue so
// templates iterate over signals in a consistent, deterministic order.
func SortedPairs(m map[string]float64) []NamedValue {
	pairs := make([]NamedValue, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, NamedValue{Name: k, Value: v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Name < pairs[j].Name })
	return pairs
}

// CatalystEventData is one event entry for catalyst prompt templates.
type CatalystEventData struct {
	Impact      string
	Probability float64
	Title       string
}

// CatalystData is the template data for catalyst prompts.
type CatalystData struct {
	AssetCode string
	AssetType string
	Direction string
	Score     float64
	MinScore  float64
	MaxScore  float64
	Events    []CatalystEventData
}

// CatalystSystem returns the static catalyst system prompt string.
func CatalystSystem() string {
	return catalystSystemSrc
}

// CatalystUser renders the catalyst user prompt template.
func CatalystUser(d *CatalystData) (string, error) {
	return render(catalystUserTmpl, d)
}

// SynthesisSystemData is the template data for the synthesis system prompt.
type SynthesisSystemData struct {
	LangInstruction string
}

// SynthesisSystem renders the synthesis system prompt template.
func SynthesisSystem(d SynthesisSystemData) (string, error) {
	return render(synthesisSystemTmpl, d)
}

// SynthesisUserData is the template data for the synthesis user prompt.
type SynthesisUserData struct {
	// Asset
	AssetName   string
	AssetCode   string
	AssetType   string
	CostPrice   float64
	PositionPct float64 // in percent (e.g. 20.0 means 20%)

	// Trend dimension
	TrendWeightPct float64
	TrendDirection string
	TrendStrength  float64
	TrendSignals   []NamedValue
	TrendSummary   string

	// Position/valuation dimension
	PosWeightPct  float64
	PosAssessment string
	PosPercentile float64
	PosMetrics    []NamedValue
	PosSummary    string

	// Catalyst dimension
	CatWeightPct float64
	CatDirection string
	CatScore     float64
	CatEvents    []CatalystEventData
	CatSummary   string

	// Matrix output
	Recommendation string
	Confidence     float64
}

// SynthesisUser renders the synthesis user prompt template.
func SynthesisUser(d *SynthesisUserData) (string, error) {
	return render(synthesisUserTmpl, d)
}

// SynthesisRecommendationData is the template data for the recommendation sub-prompt.
type SynthesisRecommendationData struct {
	Action          string
	CurrentPct      float64
	ExecTypeHint    string
	DeltaConstraint string
	StopGuidance    string
}

// SynthesisRecommendation renders the recommendation sub-prompt template.
func SynthesisRecommendation(d SynthesisRecommendationData) (string, error) {
	return render(synthesisRecommendTmpl, d)
}
