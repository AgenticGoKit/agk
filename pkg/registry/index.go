package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// DefaultRegistryURL is the URL of the official AGK template registry.
	// For production we'd point to the official repo.
	DefaultRegistryURL = "https://raw.githubusercontent.com/agk-templates/registry/main/index.json"
)

// RegistryIndex represents the structure of the registry index.json file.
type RegistryIndex struct {
	Templates map[string]string `json:"templates"`
}

// FetchIndex fetches and parses the registry index from the given URL.
func FetchIndex(url string) (*RegistryIndex, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry index: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status: %s", resp.Status)
	}

	var index RegistryIndex
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to decode registry index: %w", err)
	}

	return &index, nil
}
