package watcher

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// Watcher wraps fsnotify and exposes a channel of file events
type Watcher struct {
	fw      *fsnotify.Watcher
	Events  chan string // path of newly created files
	Errors  chan error
	done    chan struct{}
}

func New() (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fw:     fw,
		Events: make(chan string, 64), // buffered so we don't block
		Errors: make(chan error, 8),
		done:   make(chan struct{}),
	}, nil
}

// AddFolder registers a folder to be watched
func (w *Watcher) AddFolder(path string) error {
	return w.fw.Add(path)
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

			// We only care about new files landing in the folder
			// fsnotify fires Create for new files AND for renames-to
			if event.Has(fsnotify.Create) {
				log.Printf("[watcher] new file: %s", event.Name)
				w.Events <- event.Name // forward to pipeline
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