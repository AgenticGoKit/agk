package eval

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	agk "github.com/agenticgokit/agenticgokit/v1beta"
)

// LLMJudgeMatcher uses an LLM to evaluate semantic similarity
type LLMJudgeMatcher struct {
	config *SemanticConfig
	agent  agk.Agent
}

// NewLLMJudgeMatcher creates a new LLM judge matcher
func NewLLMJudgeMatcher(config *SemanticConfig) (*LLMJudgeMatcher, error) {
	// Validate LLM config
	if config.LLM == nil {
		return nil, fmt.Errorf("LLM configuration required for llm-judge strategy")
	}

	// Create judge agent using AgenticGoKit
	agent, err := createJudgeAgent(config.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create judge agent: %w", err)
	}

	return &LLMJudgeMatcher{
		config: config,
		agent:  agent,
	}, nil
}

// Match evaluates semantic similarity using LLM
func (m *LLMJudgeMatcher) Match(ctx context.Context, actual string, exp Expectation) (*MatchResult, error) {
	// Build judge prompt
	prompt := m.buildJudgePrompt(actual, exp)
	log.Printf("[LLM Judge] ========== PROMPT START ==========")
	log.Printf("%s", prompt)
	log.Printf("[LLM Judge] ========== PROMPT END ==========")
	log.Printf("[LLM Judge] Input actual output: %q (length: %d bytes)", actual, len(actual))

	// Initialize agent
	if err := m.agent.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize judge agent: %w", err)
	}
	defer func() {
		if err := m.agent.Cleanup(ctx); err != nil {
			log.Printf("Warning: failed to cleanup judge agent: %v", err)
		}
	}()

	// Use streaming for LLM judge evaluation
	log.Printf("[LLM Judge] Starting stream for evaluation...")
	stream, err := m.agent.RunStream(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to start judge agent stream: %w", err)
	}

	// Collect all chunks - handle both Delta and Content fields
	// Delta chunks (type="delta"): incremental text in Delta field
	// Text chunks (type="text"): complete text in Content field
	var response strings.Builder
	for chunk := range stream.Chunks() {
		// Prefer Delta for incremental streaming, fallback to Content for text chunks
		if chunk.Delta != "" {
			response.WriteString(chunk.Delta)
		} else if chunk.Content != "" {
			response.WriteString(chunk.Content)
		}
	}

	// Wait for stream completion and check for errors
	_, err = stream.Wait()
	if err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}

	// Parse response
	responseText := response.String()
	log.Printf("[LLM Judge] Final response (%d bytes): %q", len(responseText), responseText)
	matched, confidence, explanation := m.parseJudgment(responseText)

	return &MatchResult{
		Matched:     matched,
		Confidence:  confidence,
		Strategy:    "llm-judge",
		Explanation: explanation,
		Details: map[string]interface{}{
			"judge_response": responseText,
			"model":          m.config.LLM.Model,
			"provider":       m.config.LLM.Provider,
		},
	}, nil
}

// Name returns the matcher name
func (m *LLMJudgeMatcher) Name() string {
	return MatcherStrategyLLMJudge
}

// buildJudgePrompt constructs the prompt for the LLM judge
func (m *LLMJudgeMatcher) buildJudgePrompt(actual string, exp Expectation) string {
	template := m.config.JudgePrompt

	// Use default template if none provided
	if template == "" {
		template = `You are evaluating if an AI system's output matches the expected criteria.

Expected criteria: The output should contain one or more of these concepts:
{expected}

Actual output:
{actual}

Does the actual output satisfy the expected criteria? Consider semantic meaning, not just exact wording.
Respond with ONLY "YES" or "NO" followed by a confidence score (0.0-1.0) and brief explanation.

Format: YES|NO <confidence> - <explanation>

Example: YES 0.95 - The output clearly addresses all expected concepts`
	}

	// Build expected values list
	expectedList := ""
	for _, value := range exp.Values {
		expectedList += "- " + value + "\n"
	}
	if expectedList == "" && exp.Value != "" {
		expectedList = "- " + exp.Value + "\n"
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(template, "{expected}", expectedList)
	prompt = strings.ReplaceAll(prompt, "{actual}", actual)

	return prompt
}

// parseJudgment parses the LLM's response
func (m *LLMJudgeMatcher) parseJudgment(response string) (bool, float64, string) {
	response = strings.TrimSpace(response)

	// Parse response format: "YES 0.95 - Explanation..."
	matched := strings.HasPrefix(strings.ToUpper(response), "YES")

	// Extract confidence (simple heuristic)
	var confidence float64
	if matched {
		confidence = 0.9 // High confidence if YES
	} else {
		confidence = 0.1 // Low confidence if NO
	}

	// Try to extract numeric confidence if present
	// Format: YES|NO <number> - explanation
	parts := strings.Fields(response)
	if len(parts) >= 2 {
		if conf, err := strconv.ParseFloat(parts[1], 64); err == nil {
			confidence = conf
		}
	}

	return matched, confidence, response
}

// createJudgeAgent creates an AgenticGoKit agent from LLM config
func createJudgeAgent(config *LLMConfig) (agk.Agent, error) {
	// Create chat agent with options
	agent, err := agk.NewChatAgent(
		"eval-judge",
		agk.WithSystemPrompt("You are a precise evaluator. Follow the instructions exactly."),
		agk.WithLLMConfig(config.Provider, config.Model, float64(config.Temperature), config.MaxTokens),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat agent: %w", err)
	}

	return agent, nil
}
