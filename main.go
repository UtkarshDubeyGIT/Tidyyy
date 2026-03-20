package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/tidyyy/internal/watcher"
)

func main() {
	// Determine folder to watch: use first CLI arg, or fall back to ~/Downloads
	var folderToWatch string
	if len(os.Args) > 1 {
		abs, err := filepath.Abs(os.Args[1])
		if err != nil {
			log.Fatalf("Invalid path: %v", err)
		}
		folderToWatch = abs
	} else {
		folderToWatch = filepath.Join(os.Getenv("HOME"), "Downloads")
		fmt.Println("ℹ️  No folder specified. Usage: go run main.go <folder-path>")
		fmt.Println("   Falling back to:", folderToWatch)
	}

	w, err := watcher.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := w.AddFolder(folderToWatch); err != nil {
		log.Fatal(err)
	}

	fmt.Println("👀 Watching:", folderToWatch)
	fmt.Println("Press Ctrl+C to stop.")

	w.Start()
	defer w.Stop()

	// Print every new file we detect
	go func() {
		for path := range w.Events {
			fmt.Println("✅ Detected:", path)
		}
	}()

	// Keep the program alive until Ctrl+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down.")
}
