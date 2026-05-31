package main

import (
	_ "embed"
	"flag"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/gen2brain/beeep"
	"fyne.io/systray"

	"tgsync/app"
)

//go:embed assets/icon.png
var iconPNG []byte

//go:embed assets/icon.ico
var iconICO []byte

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// systray.Run blocks the main goroutine (required on macOS).
	// onReady runs in a separate goroutine.
	systray.Run(onReady(*cfgPath), onExit)
}

func onReady(cfgPath string) func() {
	return func() {
		// Set tray icon — Windows wants ICO, others PNG
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
		mOpenFolder := systray.AddMenuItem("Open watch folder", "Open the watched folder in your file manager")
		mReload := systray.AddMenuItem("Reload config", "Reload config.yaml without restarting")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit tgsync")

		notify := func(title, body string) {
			_ = beeep.Notify(title, body, "")
		}

		a, err := app.New(cfgPath, notify)
		if err != nil {
			log.Printf("fatal: %v", err)
			mStatus.SetTitle("❌ Config error — check logs")
			mStatus.Enable()
			// Keep tray alive so the user can quit cleanly
			<-mQuit.ClickedCh
			systray.Quit()
			return
		}

		if err := a.Start(); err != nil {
			log.Printf("fatal: %v", err)
			mStatus.SetTitle("❌ Start error — check logs")
			mStatus.Enable()
			<-mQuit.ClickedCh
			systray.Quit()
			return
		}

		mStatus.SetTitle("✅ Watching")

		for {
			select {
			case <-mOpenFolder.ClickedCh:
				openFolder(a.WatchDir())

			case <-mReload.ClickedCh:
				mStatus.SetTitle("⏳ Reloading…")
				if err := a.Reload(); err != nil {
					log.Printf("reload error: %v", err)
					mStatus.SetTitle("❌ Reload failed — check logs")
					_ = beeep.Notify("tgsync — reload failed", err.Error(), "")
				} else {
					mStatus.SetTitle("✅ Watching")
					_ = beeep.Notify("tgsync", "Config reloaded ✓", "")
				}

			case <-mQuit.ClickedCh:
				a.Stop()
				systray.Quit()
				return
			}
		}
	}
}

func onExit() {
	os.Exit(0)
}

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
