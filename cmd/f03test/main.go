// Temporary test runner for F-03 extraction. Delete before merging to main.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/tidyyy/internal/extractor"
)

func main() {
	var filePath string
	var popplerBin string
	var tesseractBin string
	var maxOCRMB int
	var timeoutSec int

	flag.StringVar(&filePath, "file", "", "path to file for extraction")
	flag.StringVar(&popplerBin, "poppler-bin", "", "path to Poppler bin directory (optional)")
	flag.StringVar(&tesseractBin, "tesseract-bin", "", "path to tesseract executable (optional)")
	flag.IntVar(&maxOCRMB, "max-ocr-mb", 20, "max image size in MB for OCR")
	flag.IntVar(&timeoutSec, "timeout-sec", 20, "command timeout in seconds")
	flag.Parse()

	if filePath == "" {
		flag.Usage()
		os.Exit(2)
	}

	cfg := extractor.Config{
		PopplerBin:     popplerBin,
		TesseractBin:   tesseractBin,
		MaxOCRBytes:    int64(maxOCRMB) * 1024 * 1024,
		CommandTimeout: time.Duration(timeoutSec) * time.Second,
	}

	svc := extractor.New(cfg, slog.Default())
	content, err := svc.ExtractPath(context.Background(), filePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("source: %s\n", content.Source)
	fmt.Printf("clean: %s\n", content.CleanText)
	fmt.Printf("tokens: %s\n", strings.Join(content.Tokens, ", "))
	fmt.Printf("\nraw:\n%s\n", content.RawText)
}
