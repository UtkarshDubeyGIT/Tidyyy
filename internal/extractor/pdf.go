package extractor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type PDFBackend interface {
	ExtractText(ctx context.Context, path string, pageLimit int) (string, error)
	RenderFirstPage(ctx context.Context, path string) (string, func(), error)
}

type PopplerBackend struct {
	pdftotextPath string
	pdftoppmPath  string
	timeout       time.Duration
	logger        *slog.Logger
}

func NewPopplerBackend(cfg Config, logger *slog.Logger) *PopplerBackend {
	if logger == nil {
		logger = slog.Default()
	}
	return &PopplerBackend{
		pdftotextPath: resolveBinaryPath(cfg.PopplerBin, "pdftotext"),
		pdftoppmPath:  resolveBinaryPath(cfg.PopplerBin, "pdftoppm"),
		timeout:       cfg.CommandTimeout,
		logger:        logger,
	}
}

func (p *PopplerBackend) ExtractText(ctx context.Context, path string, pageLimit int) (string, error) {
	if pageLimit <= 0 {
		pageLimit = 1
	}
	args := []string{
		"-f", "1",
		"-l", strconv.Itoa(pageLimit),
		"-enc", "UTF-8",
		path, "-",
	}
	out, err := runCommand(ctx, p.timeout, p.pdftotextPath, args...)
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return out, nil
}

func (p *PopplerBackend) RenderFirstPage(ctx context.Context, path string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "tidyyy-pdf-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}
	base := filepath.Join(tmpDir, "page")
	args := []string{
		"-f", "1",
		"-l", "1",
		"-png",
		"-singlefile",
		path,
		base,
	}
	if _, err := runCommand(ctx, p.timeout, p.pdftoppmPath, args...); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("pdftoppm: %w", err)
	}
	imagePath := base + ".png"
	if _, err := os.Stat(imagePath); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return imagePath, cleanup, nil
}
