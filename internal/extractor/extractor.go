package extractor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrUnsupported = errors.New("unsupported file type")
	ErrTooLarge    = errors.New("file too large for OCR")
	ErrTooShort    = errors.New("extracted text too short")
)

type ExtractedContent struct {
	RawText   string
	CleanText string
	Tokens    []string
	Source    string
}

type Config struct {
	PDFPageLimit   int
	MaxOCRBytes    int64
	MinTextLen     int
	MaxTokens      int
	CommandTimeout time.Duration
	PopplerBin     string
	TesseractBin   string
	StopWords      map[string]struct{}
}

type Service struct {
	cfg    Config
	pdf    PDFBackend
	ocr    OCRBackend
	logger *slog.Logger
}

func New(cfg Config, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	cfg = withDefaults(cfg)
	if cfg.StopWords == nil {
		cfg.StopWords = defaultStopWords()
	}
	pdf := NewPopplerBackend(cfg, logger)
	ocr := NewTesseractBackend(cfg, logger)
	return NewWithBackends(cfg, pdf, ocr, logger)
}

func NewWithBackends(cfg Config, pdf PDFBackend, ocr OCRBackend, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	cfg = withDefaults(cfg)
	if cfg.StopWords == nil {
		cfg.StopWords = defaultStopWords()
	}
	return &Service{
		cfg:    cfg,
		pdf:    pdf,
		ocr:    ocr,
		logger: logger,
	}
}

func (s *Service) ExtractPath(ctx context.Context, path string) (ExtractedContent, error) {
	kind, err := detectKind(path)
	if err != nil {
		return ExtractedContent{}, err
	}
	switch kind {
	case "pdf":
		return s.extractPDF(ctx, path)
	case "image":
		return s.extractImage(ctx, path)
	default:
		return ExtractedContent{}, ErrUnsupported
	}
}

func (s *Service) extractPDF(ctx context.Context, path string) (ExtractedContent, error) {
	raw, err := s.pdf.ExtractText(ctx, path, s.cfg.PDFPageLimit)
	if err != nil {
		return ExtractedContent{}, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if err := s.checkOCRSize(path); err != nil {
			return ExtractedContent{}, err
		}
		imgPath, cleanup, err := s.pdf.RenderFirstPage(ctx, path)
		if err != nil {
			return ExtractedContent{}, err
		}
		defer cleanup()
		raw, err = s.ocr.ExtractText(ctx, imgPath)
		if err != nil {
			return ExtractedContent{}, err
		}
		return s.buildContent(raw, "pdf+ocr")
	}
	return s.buildContent(raw, "pdf")
}

func (s *Service) extractImage(ctx context.Context, path string) (ExtractedContent, error) {
	if err := s.checkOCRSize(path); err != nil {
		return ExtractedContent{}, err
	}
	raw, err := s.ocr.ExtractText(ctx, path)
	if err != nil {
		return ExtractedContent{}, err
	}
	return s.buildContent(raw, "ocr")
}

func (s *Service) buildContent(raw, source string) (ExtractedContent, error) {
	raw = strings.TrimSpace(raw)
	if utf8.RuneCountInString(raw) < s.cfg.MinTextLen {
		return ExtractedContent{}, ErrTooShort
	}
	tokens := cleanTokens(raw, s.cfg.StopWords, s.cfg.MaxTokens)
	clean := strings.Join(tokens, " ")
	return ExtractedContent{
		RawText:   raw,
		CleanText: clean,
		Tokens:    tokens,
		Source:    source,
	}, nil
}

func (s *Service) checkOCRSize(path string) error {
	if s.cfg.MaxOCRBytes <= 0 {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > s.cfg.MaxOCRBytes {
		return ErrTooLarge
	}
	return nil
}

func detectKind(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "pdf", nil
	case ".png", ".jpg", ".jpeg", ".webp":
		return "image", nil
	default:
		return "", ErrUnsupported
	}
}

func cleanTokens(text string, stopWords map[string]struct{}, maxTokens int) []string {
	lower := strings.ToLower(text)
	var b strings.Builder
	b.Grow(len(lower))
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	fields := strings.Fields(b.String())
	seen := make(map[string]struct{}, len(fields))
	tokens := make([]string, 0, len(fields))
	for _, tok := range fields {
		if _, ok := stopWords[tok]; ok {
			continue
		}
		if isNumericToken(tok) {
			continue
		}
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		tokens = append(tokens, tok)
		if maxTokens > 0 && len(tokens) >= maxTokens {
			break
		}
	}
	return tokens
}

func isNumericToken(tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] < '0' || tok[i] > '9' {
			return false
		}
	}
	return true
}

func withDefaults(cfg Config) Config {
	if cfg.PDFPageLimit <= 0 {
		cfg.PDFPageLimit = 2
	}
	if cfg.MaxOCRBytes <= 0 {
		cfg.MaxOCRBytes = 20 * 1024 * 1024
	}
	if cfg.MinTextLen <= 0 {
		cfg.MinTextLen = 10
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 24
	}
	if cfg.CommandTimeout <= 0 {
		cfg.CommandTimeout = 20 * time.Second
	}
	return cfg
}

func defaultStopWords() map[string]struct{} {
	words := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
		"has", "he", "in", "is", "it", "its", "of", "on", "or", "that",
		"the", "to", "was", "were", "will", "with", "this", "these", "those",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, name, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("%s timed out", name)
		}
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = strings.TrimSpace(stdout.String())
		}
		if errText != "" {
			return "", fmt.Errorf("%s failed: %w: %s", name, err, errText)
		}
		return "", fmt.Errorf("%s failed: %w", name, err)
	}
	return stdout.String(), nil
}

func resolveBinaryPath(binDir, exe string) string {
	exeName := exe
	if runtime.GOOS == "windows" && !strings.HasSuffix(exeName, ".exe") {
		exeName += ".exe"
	}
	if binDir != "" {
		if strings.HasSuffix(strings.ToLower(binDir), ".exe") {
			return binDir
		}
		if info, err := os.Stat(binDir); err == nil && !info.IsDir() {
			return binDir
		}
		return filepath.Join(binDir, exeName)
	}
	if path, err := exec.LookPath(exeName); err == nil {
		return path
	}
	if isPopplerExe(exeName) {
		if popplerBin := findPopplerBin(); popplerBin != "" {
			return filepath.Join(popplerBin, exeName)
		}
	}
	return exeName
}

func isPopplerExe(exeName string) bool {
	base := strings.ToLower(strings.TrimSuffix(exeName, ".exe"))
	return base == "pdftotext" || base == "pdftoppm" || base == "pdfinfo"
}

func findPopplerBin() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return ""
	}
	base := filepath.Join(localAppData, "Microsoft", "WinGet", "Packages")
	pkgs, _ := filepath.Glob(filepath.Join(base, "oschwartz10612.Poppler*"))
	for _, pkg := range pkgs {
		bins, _ := filepath.Glob(filepath.Join(pkg, "poppler-*", "Library", "bin"))
		for _, binDir := range bins {
			if _, err := os.Stat(filepath.Join(binDir, "pdftotext.exe")); err == nil {
				return binDir
			}
		}
	}
	return ""
}
