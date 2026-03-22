package namer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type Generator interface {
	Generate(ctx context.Context, extractedText string) (string, error)
}

type Config struct {
	ModelPath    string
	LlamaCLIPath string
	EnableCloud  bool
	CloudBaseURL string
	CloudAPIKey  string
	CloudModel   string
	UseFallback  bool
}

type Service struct {
	local  Generator
	cloud  Generator
	logger *slog.Logger
	config Config
}

func New(cfg Config, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	local := NewLocalGenerator(cfg.ModelPath, cfg.LlamaCLIPath)
	var cloud Generator
	if cfg.EnableCloud {
		cloud = NewCloudGenerator(cfg.CloudBaseURL, cfg.CloudAPIKey, cfg.CloudModel)
	}

	return &Service{local: local, cloud: cloud, logger: logger, config: cfg}
}

func (s *Service) GenerateSlug(ctx context.Context, text string) (slug string, source string, err error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", "", fmt.Errorf("empty text for naming")
	}

	if s.local != nil {
		slug, err = s.local.Generate(ctx, trimmed)
		if err == nil && IsValidSlug(slug) {
			return slug, "local", nil
		}
		s.logger.Warn("local naming failed, attempting fallback", "err", err)
	}

	if s.cloud != nil {
		slug, err = s.cloud.Generate(ctx, trimmed)
		if err == nil && IsValidSlug(slug) {
			return slug, "cloud", nil
		}
		s.logger.Warn("cloud naming failed", "err", err)
	}

	if !s.config.UseFallback {
		return "", "", fmt.Errorf("no valid slug from local/cloud model and fallback disabled")
	}

	fallback := FallbackSlug(trimmed, 5)
	return fallback, "fallback", nil
}
