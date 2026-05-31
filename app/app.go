package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"tgsync/internal/config"
	"tgsync/internal/mapper"
	"tgsync/internal/uploader"
	"tgsync/internal/watcher"
)

// Notifier is a function the tray layer provides to show OS notifications.
type Notifier func(title, body string)

// App is the core application. It owns the watcher and uploader.
type App struct {
	cfg      *config.Config
	cfgPath  string
	mapper   *mapper.Mapper
	uploader *uploader.Uploader
	watcher  *watcher.Watcher
	notify   Notifier
	mu       sync.Mutex // guards restarts
}

func New(cfgPath string, notify Notifier) (*App, error) {
	a := &App{cfgPath: cfgPath, notify: notify}
	if err := a.load(); err != nil {
		return nil, err
	}
	return a, nil
}

// load (re)reads config and rebuilds mapper + uploader.
func (a *App) load() error {
	cfg, err := config.Load(a.cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.cfg = cfg
	a.mapper = mapper.New(cfg.Mappings)
	a.uploader = uploader.New(cfg.BotToken, cfg.ChatID)
	return nil
}

// Start runs the startup scan and launches the file watcher.
func (a *App) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Println("scanning existing files...")
	if err := a.scanExisting(); err != nil {
		log.Printf("startup scan error: %v", err)
	}

	w, err := watcher.New(a.cfg.WatchDir, func(path string) {
		a.processFile(path)
	})
	if err != nil {
		return fmt.Errorf("start watcher: %w", err)
	}
	a.watcher = w

	go w.Run()
	log.Printf("watching %s", a.cfg.WatchDir)
	return nil
}

// Reload stops the watcher, reloads config, and restarts.
func (a *App) Reload() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.watcher != nil {
		_ = a.watcher.Close()
	}
	if err := a.load(); err != nil {
		return err
	}

	log.Println("config reloaded")

	w, err := watcher.New(a.cfg.WatchDir, func(path string) {
		a.processFile(path)
	})
	if err != nil {
		return fmt.Errorf("restart watcher: %w", err)
	}
	a.watcher = w
	go w.Run()
	return nil
}

// Stop shuts down the watcher gracefully.
func (a *App) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watcher != nil {
		_ = a.watcher.Close()
	}
}

// WatchDir returns the currently watched directory (used by tray to open it).
func (a *App) WatchDir() string {
	return a.cfg.WatchDir
}

func (a *App) scanExisting() error {
	return filepath.Walk(a.cfg.WatchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		a.processFile(path)
		return nil
	})
}

func (a *App) processFile(path string) {
	info, err := os.Stat(path)
	if err != nil {
		log.Printf("warn: stat %s: %v", path, err)
		return
	}

	if info.Size() > a.cfg.MaxFileSizeBytes {
		msg := fmt.Sprintf("skipping %s — %.1f MB exceeds limit", filepath.Base(path), float64(info.Size())/1e6)
		log.Printf("warn: %s", msg)
		a.notify("tgsync — skipped", msg)
		return
	}

	topicID, ok := a.mapper.TopicID(path)
	if !ok {
		subfolder := filepath.Base(filepath.Dir(path))
		log.Printf("warn: skipping %s — subfolder %q not mapped", path, subfolder)
		return
	}

	log.Printf("uploading %s → topic %d", path, topicID)
	if err := a.uploader.SendFile(path, topicID); err != nil {
		msg := fmt.Sprintf("%s failed: %v", filepath.Base(path), err)
		log.Printf("error: %s", msg)
		a.notify("tgsync — upload failed", msg)
		return
	}

	if err := os.Remove(path); err != nil {
		log.Printf("warn: uploaded but could not remove %s: %v", path, err)
		return
	}

	name := filepath.Base(path)
	subfolder := filepath.Base(filepath.Dir(path))
	log.Printf("done: %s → #%s", name, subfolder)
	a.notify("tgsync", fmt.Sprintf("%s → #%s ✓", name, subfolder))
}
