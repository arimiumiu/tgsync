package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Handler func(filePath string)

type Watcher struct {
	w       *fsnotify.Watcher
	rootDir string
	handler Handler
}

func New(rootDir string, handler Handler) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	if err := w.Add(rootDir); err != nil {
		return nil, fmt.Errorf("watch root dir: %w", err)
	}

	// Watch all existing subdirectories at startup
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("read watch dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			sub := filepath.Join(rootDir, e.Name())
			if err := w.Add(sub); err != nil {
				log.Printf("warn: could not watch %s: %v", sub, err)
			}
		}
	}

	return &Watcher{w: w, rootDir: rootDir, handler: handler}, nil
}

// Run starts the event loop. Blocks until the watcher is closed.
func (wt *Watcher) Run() {
	for {
		select {
		case event, ok := <-wt.w.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err != nil {
					continue
				}
				if info.IsDir() {
					// New subfolder created — watch it too
					if err := wt.w.Add(event.Name); err != nil {
						log.Printf("warn: could not watch new dir %s: %v", event.Name, err)
					}
					continue
				}
				// Handle the file in a goroutine so we don't block the event loop
				go func(path string) {
					if err := waitUntilStable(path); err != nil {
						log.Printf("warn: stability check failed for %s: %v", path, err)
						return
					}
					wt.handler(path)
				}(event.Name)
			}

		case err, ok := <-wt.w.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func (wt *Watcher) Close() error {
	return wt.w.Close()
}

// waitUntilStable polls the file size every 500ms until it stops changing.
// This is the cross-platform way to detect that a file copy is complete.
func waitUntilStable(path string) error {
	var prev int64 = -1
	for {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat: %w", err)
		}
		size := info.Size()
		if size == prev && size > 0 {
			return nil
		}
		prev = size
		time.Sleep(500 * time.Millisecond)
	}
}
