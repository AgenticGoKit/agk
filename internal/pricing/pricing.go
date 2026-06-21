// Package pricing provides approximate USD cost estimates for LLM token usage.
//
// Prices are published list prices per 1,000,000 tokens and are intended for
// rough cost reporting in traces, not billing. They drift over time; update the
// table as providers change pricing. Local models (Ollama, etc.) are treated as
// free and intentionally absent from the table.
package pricing

import "strings"

// ModelPrice is the USD price per 1,000,000 tokens, split by input and output.
type ModelPrice struct {
	InputPer1M  float64
	OutputPer1M float64
}

// table maps a normalized model key to its price. Lookups use exact match first,
// then the longest matching prefix, so dated/variant model ids (e.g.
// "gpt-4o-2024-08-06", "claude-sonnet-4-20250514") resolve to their base model.
var table = map[string]ModelPrice{
	// OpenAI
	"gpt-4o":        {InputPer1M: 2.50, OutputPer1M: 10.00},
	"gpt-4o-mini":   {InputPer1M: 0.15, OutputPer1M: 0.60},
	"gpt-4-turbo":   {InputPer1M: 10.00, OutputPer1M: 30.00},
	"gpt-4":         {InputPer1M: 30.00, OutputPer1M: 60.00},
	"gpt-3.5-turbo": {InputPer1M: 0.50, OutputPer1M: 1.50},
	"o1":            {InputPer1M: 15.00, OutputPer1M: 60.00},
	"o1-mini":       {InputPer1M: 1.10, OutputPer1M: 4.40},
	"o3-mini":       {InputPer1M: 1.10, OutputPer1M: 4.40},

	// Anthropic
	"claude-3-5-sonnet": {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-5-haiku":  {InputPer1M: 0.80, OutputPer1M: 4.00},
	"claude-3-opus":     {InputPer1M: 15.00, OutputPer1M: 75.00},
	"claude-3-sonnet":   {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-3-haiku":    {InputPer1M: 0.25, OutputPer1M: 1.25},
	"claude-sonnet-4":   {InputPer1M: 3.00, OutputPer1M: 15.00},
	"claude-opus-4":     {InputPer1M: 15.00, OutputPer1M: 75.00},
	"claude-haiku-4":    {InputPer1M: 1.00, OutputPer1M: 5.00},
}

// Estimate returns the estimated USD cost for the given token usage and whether
// the model was found in the price table. Unknown or local models return (0, false),
// letting callers decide how to report an unpriced run.
func Estimate(model string, inputTokens, outputTokens int) (float64, bool) {
	p, ok := lookup(model)
	if !ok {
		return 0, false
	}
	cost := float64(inputTokens)/1e6*p.InputPer1M + float64(outputTokens)/1e6*p.OutputPer1M
	return cost, true
}

func lookup(model string) (ModelPrice, bool) {
	key := normalize(model)
	if key == "" {
		return ModelPrice{}, false
	}
	if p, ok := table[key]; ok {
		return p, true
	}
	// Longest-prefix match handles dated/variant ids.
	var best string
	for k := range table {
		if strings.HasPrefix(key, k) && len(k) > len(best) {
			best = k
		}
	}
	if best != "" {
		return table[best], true
	}
	return ModelPrice{}, false
}

// normalize lowercases the model id and strips any provider prefix
// (e.g. "openai/gpt-4o" -> "gpt-4o").
func normalize(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if i := strings.LastIndex(m, "/"); i >= 0 {
		m = m[i+1:]
	}
	return m
}
