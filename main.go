package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/tidyyy/internal/extractor"
	"github.com/tidyyy/internal/triage"
	"github.com/tidyyy/internal/watcher"
)

func main() {
	var filePath string
	var watchDir string
	var popplerBin string
	var tesseractBin string
	var maxOCRMB int
	var timeoutSec int

	flag.StringVar(&filePath, "file", "", "path to a single file for extraction (one-off mode)")
	flag.StringVar(&watchDir, "watch", "", "directory to watch for new files (daemon mode)")
	flag.StringVar(&popplerBin, "poppler-bin", "", "path to Poppler bin directory (optional)")
	flag.StringVar(&tesseractBin, "tesseract-bin", "", "path to tesseract executable (optional)")
	flag.IntVar(&maxOCRMB, "max-ocr-mb", 20, "max image size in MB for OCR")
	flag.IntVar(&timeoutSec, "timeout-sec", 20, "command timeout in seconds")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := extractor.Config{
		PopplerBin:     popplerBin,
		TesseractBin:   tesseractBin,
		MaxOCRBytes:    int64(maxOCRMB) * 1024 * 1024,
		CommandTimeout: time.Duration(timeoutSec) * time.Second,
	}
	svc := extractor.New(cfg, logger)

	// One-off extraction mode
	if filePath != "" {
		extractAndPrint(svc, filePath)
		return
	}

	// Daemon mode: Watcher + Triage + Extractor
	if watchDir == "" {
		// Use positional argument if flag is missing, or fall back to default
		if flag.NArg() > 0 {
			watchDir = flag.Arg(0)
		} else {
			watchDir = filepath.Join(os.Getenv("HOME"), "Downloads")
			fmt.Println("ℹ️  No folder specified. Watching fallback:", watchDir)
		}
	}

	absWatchDir, err := filepath.Abs(watchDir)
	if err != nil {
		log.Fatalf("Invalid watch path: %v", err)
	}

	w, err := watcher.New()
	if err != nil {
		log.Fatal(err)
	}

	tr := triage.New(64, logger)

	if err := w.AddFolder(absWatchDir); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("👀 Watching: %s\n", absWatchDir)
	fmt.Println("Press Ctrl+C to stop.")

	w.Start()
	defer w.Stop()

	// Pipeline: Watcher -> Triage
	go func() {
		for path := range w.Events {
			// Small delay to allow file to be fully written (simple approach)
			time.Sleep(500 * time.Millisecond)
			_, err := tr.Accept(context.Background(), path)
			if err != nil {
				logger.Error("triage failed", "path", path, "err", err)
			}
		}
	}()

	// Pipeline: Triage -> Extractor
	go func() {
		for job := range tr.Queue() {
			logger.Info("extracting", "path", job.Path)
			content, err := svc.ExtractPath(context.Background(), job.Path)
			if err != nil {
				logger.Error("extraction failed", "path", job.Path, "err", err)
				continue
			}
			fmt.Printf("\n--- Extracted from %s ---\n", filepath.Base(job.Path))
			fmt.Printf("Source: %s\n", content.Source)
			fmt.Printf("Tokens: %s\n", strings.Join(content.Tokens, ", "))
			fmt.Printf("Text (excerpt): %.100s...\n", content.CleanText)
			fmt.Println("-------------------------------------------")
		}
	}()

	// Keep the program alive until Ctrl+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down.")
}

func extractAndPrint(svc *extractor.Service, filePath string) {
	content, err := svc.ExtractPath(context.Background(), filePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("source: %s\n", content.Source)
	fmt.Printf("clean: %s\n", content.CleanText)
	fmt.Printf("tokens: %s\n", strings.Join(content.Tokens, ", "))
	fmt.Printf("\nraw:\n%s\n", content.RawText)
}
