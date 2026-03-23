package watcher

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const settleDelay = 2 * time.Second

// Watcher wraps fsnotify and exposes a channel of file events
type Watcher struct {
	fw      *fsnotify.Watcher
	Events  chan string // path of newly created files
	Errors  chan error
	done    chan struct{}
	mu      sync.Mutex
	pending map[string]*time.Timer
}

func New() (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fw:      fw,
		Events:  make(chan string, 64), // buffered so we don't block
		Errors:  make(chan error, 8),
		done:    make(chan struct{}),
		pending: make(map[string]*time.Timer),
	}, nil
}

// AddFolder registers a folder to be watched
func (w *Watcher) AddFolder(path string) error {
	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.fw.Add(p)
		}
		return nil
	})
}

// RemoveFolder unregisters a folder
func (w *Watcher) RemoveFolder(path string) error {
	return w.fw.Remove(path)
}

// Start begins the event loop in a goroutine
func (w *Watcher) Start() {
	go w.loop()
}

// Stop shuts down the watcher cleanly
func (w *Watcher) Stop() {
	close(w.done)
	w.fw.Close()
}

// loop is the internal event dispatcher
func (w *Watcher) loop() {
	for {
		select {
		case event, ok := <-w.fw.Events:
			if !ok {
				return // channel closed
			}

			// Keep recursive watches for newly-created subdirectories.
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := w.AddFolder(event.Name); err != nil {
						w.Errors <- err
					}
					continue
				}
			}

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Write) {
				if info, err := os.Stat(event.Name); err == nil && !info.IsDir() {
					w.schedule(event.Name)
				}
			}

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			w.Errors <- err

		case <-w.done:
			return
		}
	}
}

func (w *Watcher) schedule(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if t, ok := w.pending[path]; ok {
		t.Stop()
	}

	w.pending[path] = time.AfterFunc(settleDelay, func() {
		log.Printf("[watcher] file settled: %s", path)
		w.Events <- path
		
		w.mu.Lock()
		delete(w.pending, path)
		w.mu.Unlock()
	})
}
