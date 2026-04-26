package aiops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client communicates with a local Ollama instance via its REST API.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type modelInfo struct {
	Name string `json:"name"`
}

type listModelsResponse struct {
	Models []modelInfo `json:"models"`
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Ping checks if Ollama is reachable and the configured model is available.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	var lr listModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return err
	}

	for _, m := range lr.Models {
		if m.Name == c.model || m.Name == c.model+":latest" {
			return nil
		}
	}
	return fmt.Errorf("model %q not found in Ollama — run: ollama pull %s", c.model, c.model)
}

// Generate sends a prompt to Ollama and returns the full response text.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama generate error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned HTTP %d", resp.StatusCode)
	}

	var gr generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", err
	}

	return gr.Response, nil
}

// Model returns the configured model name.
func (c *Client) Model() string {
	return c.model
}
