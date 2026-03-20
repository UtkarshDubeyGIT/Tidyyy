package extractor

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type OCRBackend interface {
	ExtractText(ctx context.Context, path string) (string, error)
}

type TesseractBackend struct {
	tesseractPath string
	timeout       time.Duration
	logger        *slog.Logger
	psm           string
}

func NewTesseractBackend(cfg Config, logger *slog.Logger) *TesseractBackend {
	if logger == nil {
		logger = slog.Default()
	}
	return &TesseractBackend{
		tesseractPath: resolveBinaryPath(cfg.TesseractBin, "tesseract"),
		timeout:       cfg.CommandTimeout,
		logger:        logger,
		psm:           "6",
	}
}

func (t *TesseractBackend) ExtractText(ctx context.Context, path string) (string, error) {
	args := []string{path, "stdout", "--psm", t.psm}
	out, err := runCommand(ctx, t.timeout, t.tesseractPath, args...)
	if err != nil {
		return "", fmt.Errorf("tesseract: %w", err)
	}
	return out, nil
}
