package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tidyyy/internal/watcher"
)

func main() {
	w, err := watcher.New()
	if err != nil {
		log.Fatal(err)
	}

	// Watch your Downloads folder — change this path if needed
	folderToWatch := os.Getenv("USERPROFILE") + "\\Downloads"
	if err := w.AddFolder(folderToWatch); err != nil {
		log.Fatal(err)
	}

	fmt.Println("👀 Watching:", folderToWatch)
	fmt.Println("Drop a file in there to test. Press Ctrl+C to stop.")

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
