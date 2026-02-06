package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPTarget handles HTTP-based test execution
type HTTPTarget struct {
	baseURL string
	client  *http.Client
}

// NewHTTPTarget creates a new HTTP target
func NewHTTPTarget(baseURL string, timeout time.Duration) *HTTPTarget {
	return &HTTPTarget{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// InvokeRequest matches the EvalServer's request format
type InvokeRequest struct {
	Input     string                 `json:"input"`
	SessionID string                 `json:"sessionID,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// InvokeResponse matches the EvalServer's response format
type InvokeResponse struct {
	Output      string   `json:"output"`
	TraceID     string   `json:"trace_id"`
	SessionID   string   `json:"session_id"`
	DurationMs  int64    `json:"duration_ms"`
	Success     bool     `json:"success"`
	ToolsCalled []string `json:"tools_called,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// Invoke sends a test to the target and returns the response
func (ht *HTTPTarget) Invoke(input string, timeout int) (*InvokeResponse, error) {
	// Build request
	req := InvokeRequest{
		Input:     input,
		SessionID: "",
		Options: map[string]interface{}{
			"timeout": timeout,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send HTTP request
	httpReq, err := http.NewRequest("POST", ht.baseURL+"/invoke", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := ht.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var invokeResp InvokeResponse
	if err := json.Unmarshal(body, &invokeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invokeResp, nil
}

// Health checks if the target is healthy
func (ht *HTTPTarget) Health() error {
	resp, err := ht.client.Get(ht.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
