package eval

import (
	"context"
	"fmt"
)

// HybridMatcher combines embedding and LLM judge strategies
type HybridMatcher struct {
	config           *SemanticConfig
	embeddingMatcher *EmbeddingMatcher
	llmMatcher       *LLMJudgeMatcher
}

// NewHybridMatcher creates a new hybrid matcher
func NewHybridMatcher(config *SemanticConfig) (*HybridMatcher, error) {
	// Validate config
	if config.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration required for hybrid strategy")
	}
	if config.LLM == nil {
		return nil, fmt.Errorf("LLM configuration required for hybrid strategy")
	}

	// Create embedding matcher
	embMatcher, err := NewEmbeddingMatcher(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding matcher: %w", err)
	}

	// Create LLM matcher
	llmMatcher, err := NewLLMJudgeMatcher(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM matcher: %w", err)
	}

	return &HybridMatcher{
		config:           config,
		embeddingMatcher: embMatcher,
		llmMatcher:       llmMatcher,
	}, nil
}

// Match evaluates using hybrid approach
// Strategy: Fast embedding filter, then LLM judge for edge cases
func (m *HybridMatcher) Match(ctx context.Context, actual string, exp Expectation) (*MatchResult, error) {
	// Step 1: Quick embedding check
	embResult, err := m.embeddingMatcher.Match(ctx, actual, exp)
	if err != nil {
		return nil, fmt.Errorf("embedding match failed: %w", err)
	}

	// If embedding confidence is very high, trust it (fast path)
	if embResult.Confidence >= 0.95 {
		embResult.Strategy = "hybrid (embedding-confident)"
		embResult.Details["decision"] = "high confidence from embedding"
		return embResult, nil
	}

	// If embedding confidence is very low, reject without LLM call (fast path)
	if embResult.Confidence <= 0.3 {
		embResult.Strategy = "hybrid (embedding-reject)"
		embResult.Details["decision"] = "low confidence from embedding"
		return embResult, nil
	}

	// Step 2: Edge case (medium confidence) - use LLM judge for final decision
	llmResult, err := m.llmMatcher.Match(ctx, actual, exp)
	if err != nil {
		// Fallback to embedding result if LLM fails
		embResult.Strategy = "hybrid (llm-failed-fallback)"
		embResult.Details["llm_error"] = err.Error()
		embResult.Details["decision"] = "fallback to embedding due to LLM error"
		return embResult, nil
	}

	// Combine results (weighted average: embedding 30%, LLM 70%)
	combinedConfidence := (embResult.Confidence * 0.3) + (llmResult.Confidence * 0.7)

	llmResult.Confidence = combinedConfidence
	llmResult.Strategy = "hybrid (embedding+llm)"
	llmResult.Details["embedding_confidence"] = embResult.Confidence
	llmResult.Details["llm_confidence"] = llmResult.Confidence
	llmResult.Details["combined_confidence"] = combinedConfidence
	llmResult.Details["decision"] = "combined embedding and LLM evaluation"

	return llmResult, nil
}

// Name returns the matcher name
func (m *HybridMatcher) Name() string {
	return MatcherStrategyHybrid
}
