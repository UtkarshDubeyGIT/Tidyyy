package namer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const DefaultModelPath = "./gemma-3-270m-it-Q8_0.gguf"

type LocalGenerator struct {
	modelPath string
	cliPath   string
	maxTokens int
	temp      float64
	topP      float64
}

type localResponse struct {
	Slug string `json:"slug"`
}

func NewLocalGenerator(modelPath string, llamaCLIPath string) *LocalGenerator {
	if strings.TrimSpace(modelPath) == "" {
		modelPath = DefaultModelPath
	}
	if strings.TrimSpace(llamaCLIPath) == "" {
		llamaCLIPath = "llama-cli"
	}

	return &LocalGenerator{
		modelPath: modelPath,
		cliPath:   llamaCLIPath,
		maxTokens: 80,
		temp:      0.1,
		topP:      0.9,
	}
}

func runLlamaCLI(ctx context.Context, cliPath string, modelPath string, prompt string, maxTokens int, temperature float64, topP float64) (string, error) {
	args := []string{
		"-m", modelPath,
		"-p", prompt,
		"-n", fmt.Sprintf("%d", maxTokens),
		"--temp", fmt.Sprintf("%.2f", temperature),
		"--top-p", fmt.Sprintf("%.2f", topP),
		"--single-turn",
		"--no-display-prompt",
		"--simple-io",
	}
	cmd := exec.CommandContext(ctx, cliPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("llama-cli failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	text := extractGeneratedText(string(out), prompt)
	if text == "" {
		return "", errors.New("empty output from llama-cli")
	}
	return text, nil
}

func (l *LocalGenerator) Generate(ctx context.Context, extractedText string) (string, error) {
	if strings.TrimSpace(extractedText) == "" {
		return "", errors.New("extracted text is empty")
	}

	prepared := prepareNamingInput(extractedText)
	prompt := buildLocalPrompt(prepared)
	raw, err := runLlamaCLI(ctx, l.cliPath, l.modelPath, prompt, l.maxTokens, l.temp, l.topP)
	if err != nil {
		return "", err
	}

	slug := parseSlugFromModel(raw, prepared)
	if IsValidSlug(slug) {
		return slug, nil
	}

	strictPrompt := buildStrictRetryPrompt(prepared)
	strictRaw, strictErr := runLlamaCLI(ctx, l.cliPath, l.modelPath, strictPrompt, 24, 0.0, 0.8)
	if strictErr != nil {
		return "", fmt.Errorf("strict retry failed after invalid first attempt: %w", strictErr)
	}

	slug = parseSlugFromModel(strictRaw, prepared)
	if !IsValidSlug(slug) {
		return "", fmt.Errorf("invalid slug from local model: %q", slug)
	}
	return slug, nil
}

func buildLocalPrompt(extractedText string) string {
	return fmt.Sprintf(
		"Generate a concise filename slug from the text. Return ONLY JSON with key 'slug'. Rules: 2 to 5 words, lowercase, words separated by hyphen, no timestamps, no counters, no file extension. Text: %s",
		extractedText,
	)
}

func buildStrictRetryPrompt(extractedText string) string {
	return fmt.Sprintf(
		"Return ONLY JSON with one key named slug. Constraints are strict: 2 to 5 words, lowercase a-z only, hyphen between words, no numbers, no timestamps, no counters, no generic placeholders, no extra keys, no commentary. Text: %s",
		extractedText,
	)
}

func prepareNamingInput(text string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(fields) == 0 {
		return ""
	}

	kept := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.Trim(f, "-_")
		if f == "" {
			continue
		}
		if len(f) < 2 {
			continue
		}
		if strings.Contains(f, "ggml") || strings.Contains(f, "llama") || strings.Contains(f, "metal") {
			continue
		}
		kept = append(kept, f)
		if len(kept) >= 80 {
			break
		}
	}

	if len(kept) == 0 {
		return strings.TrimSpace(text)
	}
	return strings.Join(kept, " ")
}

func parseSlugFromModel(raw string, extractedText string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return FallbackSlug(extractedText, 5)
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		var resp localResponse
		if err := json.Unmarshal([]byte(trimmed[start:end+1]), &resp); err == nil {
			normalized := NormalizeSlug(resp.Slug)
			if normalized != "" && !isPlaceholderSlug(normalized) {
				return normalized
			}
		}
	}

	normalized := NormalizeSlug(trimmed)
	if isPlaceholderSlug(normalized) {
		return ""
	}
	return normalized
}

func isPlaceholderSlug(slug string) bool {
	if slug == "" {
		return true
	}
	bad := map[string]struct{}{
		"two-to-five-words": {},
		"example-slug":      {},
		"your-slug-here":    {},
		"document-file":     {},
	}
	_, exists := bad[slug]
	return exists
}

func extractGeneratedText(raw string, prompt string) string {
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	anchor := "\n> " + prompt
	if idx := strings.Index(text, anchor); idx >= 0 {
		candidate := text[idx+len(anchor):]
		candidate = strings.TrimLeft(candidate, "\n ")

		if end := strings.Index(candidate, "\n["); end >= 0 {
			candidate = candidate[:end]
		}
		if end := strings.Index(candidate, "\n>"); end >= 0 {
			candidate = candidate[:end]
		}
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}

	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "ggml_") || strings.HasPrefix(trimmed, "llama_") || strings.HasPrefix(trimmed, "build") || strings.HasPrefix(trimmed, "model") || strings.HasPrefix(trimmed, "modalities") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, ">") {
			continue
		}
		kept = append(kept, trimmed)
	}

	return strings.TrimSpace(strings.Join(kept, " "))
}
