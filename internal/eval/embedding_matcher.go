package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// EmbeddingMatcher uses embeddings to evaluate semantic similarity
type EmbeddingMatcher struct {
	config   *SemanticConfig
	embedder EmbeddingClient
}

// EmbeddingClient interface for generating embeddings
type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// NewEmbeddingMatcher creates a new embedding matcher
func NewEmbeddingMatcher(config *SemanticConfig) (*EmbeddingMatcher, error) {
	// Validate embedding config
	if config.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration required for embedding strategy")
	}

	// Create embedding client
	embedder, err := createEmbeddingClient(config.Embedding)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding client: %w", err)
	}

	return &EmbeddingMatcher{
		config:   config,
		embedder: embedder,
	}, nil
}

// Match evaluates semantic similarity using embeddings
func (m *EmbeddingMatcher) Match(ctx context.Context, actual string, exp Expectation) (*MatchResult, error) {
	// Get embedding for actual output
	actualEmbed, err := m.embedder.Embed(ctx, actual)
	if err != nil {
		return nil, fmt.Errorf("failed to embed actual output: %w", err)
	}

	// Compare with each expected value
	var maxSimilarity float64
	var bestMatch string

	values := exp.Values
	if len(values) == 0 && exp.Value != "" {
		values = []string{exp.Value}
	}

	for _, expected := range values {
		expectedEmbed, err := m.embedder.Embed(ctx, expected)
		if err != nil {
			continue
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(actualEmbed, expectedEmbed)

		if similarity > maxSimilarity {
			maxSimilarity = similarity
			bestMatch = expected
		}
	}

	threshold := m.config.Threshold
	matched := maxSimilarity >= threshold

	explanation := fmt.Sprintf("Similarity: %.2f (threshold: %.2f) - Best match: %s",
		maxSimilarity, threshold, bestMatch)

	return &MatchResult{
		Matched:     matched,
		Confidence:  maxSimilarity,
		Strategy:    "embedding",
		Explanation: explanation,
		Details: map[string]interface{}{
			"similarity": maxSimilarity,
			"threshold":  threshold,
			"best_match": bestMatch,
			"model":      m.config.Embedding.Model,
		},
	}, nil
}

// Name returns the matcher name
func (m *EmbeddingMatcher) Name() string {
	return MatcherStrategyEmbedding
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ========================================
// Embedding Clients
// ========================================

// createEmbeddingClient creates appropriate embedding client based on provider
func createEmbeddingClient(config *EmbeddingConfig) (EmbeddingClient, error) {
	switch config.Provider {
	case "ollama":
		return NewOllamaEmbeddingClient(config)
	case "openai":
		return NewOpenAIEmbeddingClient(config)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", config.Provider)
	}
}

// ========================================
// Ollama Embedding Client
// ========================================

type OllamaEmbeddingClient struct {
	baseURL string
	model   string
	client  *http.Client
}

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

func NewOllamaEmbeddingClient(config *EmbeddingConfig) (*OllamaEmbeddingClient, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaEmbeddingClient{
		baseURL: baseURL,
		model:   config.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *OllamaEmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := ollamaEmbedRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/api/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embedding, nil
}

// ========================================
// OpenAI Embedding Client
// ========================================

type OpenAIEmbeddingClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

type openaiEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openaiEmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func NewOpenAIEmbeddingClient(config *EmbeddingConfig) (*OpenAIEmbeddingClient, error) {
	// TODO: Get API key from environment or config
	apiKey := "" // Get from env: os.Getenv("OPENAI_API_KEY")

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIEmbeddingClient{
		apiKey:  apiKey,
		model:   config.Model,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *OpenAIEmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := openaiEmbedRequest{
		Model: c.model,
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result openaiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned from OpenAI")
	}

	return result.Data[0].Embedding, nil
}
