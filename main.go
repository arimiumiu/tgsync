package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"fyne.io/systray"
	"github.com/gen2brain/beeep"

	"tgsync/app"
)

//go:embed assets/icon.png
var iconPNG []byte

//go:embed assets/icon.ico
var iconICO []byte

func main() {
	cfgPath := flag.String("config", defaultConfigPath(), "path to config file")
	flag.Parse()

	// systray.Run blocks the main goroutine (required on macOS).
	// onReady runs in a separate goroutine.
	systray.Run(onReady(*cfgPath), onExit)
}

func onReady(cfgPath string) func() {
	return func() {
		if runtime.GOOS == "windows" {
			systray.SetIcon(iconICO)
		} else {
			systray.SetIcon(iconPNG)
		}
		systray.SetTitle("tgsync")
		systray.SetTooltip("tgsync — Telegram folder sync")

		mStatus := systray.AddMenuItem("⏳ Starting…", "Current status")
		mStatus.Disable()
		systray.AddSeparator()
		mEditConfig := systray.AddMenuItem("Edit config…", "Open config.yaml in default editor")
		mOpenFolder := systray.AddMenuItem("Open watch folder", "Open the watched folder in your file manager")
		mReload     := systray.AddMenuItem("Reload config", "Reload config.yaml without restarting")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit tgsync")

		notify := func(title, body string) {
			_ = beeep.Notify(title, body, "")
		}

		if err := ensureConfig(cfgPath); err != nil {
			log.Printf("warn: could not create default config: %v", err)
		}

		// tryStart attempts to create the app and start the watcher.
		// Returns (app, nil) on success, (nil, err) on failure.
		tryStart := func() (*app.App, error) {
			a, err := app.New(cfgPath, notify)
			if err != nil {
				return nil, err
			}
			if err := a.Start(); err != nil {
				return nil, err
			}
			return a, nil
		}

		a, err := tryStart()
		if err != nil {
			log.Printf("error: %v", err)
			mStatus.SetTitle("❌ Config error — check logs")
			mStatus.Enable()
		} else {
			mStatus.SetTitle("✅ Watching")
		}

		for {
			select {
			case <-mEditConfig.ClickedCh:
				openFile(cfgPath)

			case <-mOpenFolder.ClickedCh:
				if a != nil {
					openFolder(a.WatchDir())
				} else {
					openFolder(filepath.Dir(cfgPath))
				}

			case <-mReload.ClickedCh:
				mStatus.SetTitle("⏳ Reloading…")
				if a != nil {
					a.Stop()
				}
				a, err = tryStart()
				if err != nil {
					log.Printf("reload error: %v", err)
					mStatus.SetTitle("❌ Config error — check logs")
					mStatus.Enable()
					notify("tgsync — reload failed", err.Error())
				} else {
					mStatus.SetTitle("✅ Watching")
					notify("tgsync", "Config reloaded ✓")
				}

			case <-mQuit.ClickedCh:
				if a != nil {
					a.Stop()
				}
				systray.Quit()
				return
			}
		}
	}
}

func onExit() {
	os.Exit(0)
}

func defaultConfigPath() string {
    if dir, err := os.UserConfigDir(); err == nil {
        return filepath.Join(dir, "tgsync", "config.yaml")
    }
    return "config.yaml"
}

func ensureConfig(path string) error {
    if _, err := os.Stat(path); err == nil {
        return nil // already exists
    }
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return fmt.Errorf("create config dir: %w", err)
    }
    return os.WriteFile(path, []byte(defaultConfig), 0644)
}

const defaultConfig = `bot_token: ""
chat_id: 0
watch_dir: ""

mappings:
  # subfolder-name: topic-id
  # docs: 12
`
// openFolder opens the given directory in the native file manager.
func openFolder(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux, bsd, etc.
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("warn: could not open folder: %v", err)
	}
}

func openFile(path string) {
	var cmd *exec.Cmd
	log.Printf("i am opening a file")
	switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
		case "darwin":
			cmd = exec.Command("open", path)
		default:
			cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}
