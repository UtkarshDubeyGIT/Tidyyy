package namer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CloudGenerator struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Temp     float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func NewCloudGenerator(baseURL string, apiKey string, model string) *CloudGenerator {
	if strings.TrimSpace(baseURL) == "" || strings.TrimSpace(apiKey) == "" || strings.TrimSpace(model) == "" {
		return nil
	}
	return &CloudGenerator{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

func (c *CloudGenerator) Generate(ctx context.Context, extractedText string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("cloud generator not configured")
	}

	reqBody := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{Role: "system", Content: "Return only a slug: 2 to 5 lowercase words joined by hyphen, no timestamps, no counters."},
			{Role: "user", Content: extractedText},
		},
		Temp: 0.1,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("cloud naming failed: %s", strings.TrimSpace(string(body)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("cloud naming returned no choices")
	}

	slug := NormalizeSlug(parsed.Choices[0].Message.Content)
	if !IsValidSlug(slug) {
		return "", fmt.Errorf("invalid slug from cloud response")
	}
	return slug, nil
}
