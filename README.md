# tgsync

super simple sync app: folders act like sinks, you throw files in them and you upload them automatically to a telegram topic group.

Watches a folder and uploads new files to a Telegram supergroup, routing each
subfolder to its own topic. Files are removed after a confirmed upload.
Lives in the system tray with native OS notifications.

## Setup

### 1. Dependencies

```bash
go mod tidy
```

### 2. Icon files

Put your icons in `assets/`:
- `icon.png` — 256×256 PNG (Linux, macOS)
- `icon.ico` — multi-size ICO (Windows)

Both are required at compile time. Quick way to make the ICO from the PNG:
```bash
convert icon.png -define icon:auto-resize=256,48,32,16 assets/icon.ico
```

### 3. Config

Edit `config.yaml`:
```yaml
bot_token: "YOUR_BOT_TOKEN"
chat_id: -1001234567890      # must start with -100
watch_dir: "/path/to/folder"

mappings:
  subfolder-name: 4          # topic ID from Telegram Web URL
  docs: 12
  music: 37
```

**Getting the topic ID:** open your supergroup in Telegram Web, click a topic,
look at the URL — the number after `/t/` is the topic ID.

**Getting the chat ID:** add @userinfobot to your group and send any message.

**Bot permissions:** add the bot to your supergroup as an admin with at least
"Send Messages" and "Send Media" permissions.

### 4. Build & run

**Linux / macOS:**
```bash
go build -o tgsync .
./tgsync -config config.yaml
```

**Windows (no console window):**
```bash
go build -ldflags="-H windowsgui" -o tgsync.exe .
tgsync.exe -config config.yaml
```

## How it works

```
watch_dir/
  photos/   →  topic 4
  docs/     →  topic 12
  music/    →  topic 37
```

1. On startup, scans all existing files and uploads any not yet sent.
2. Watches for new files via fsnotify.
3. When a file appears, polls its size every 500ms until it stops changing
   (handles large copies safely, works on all platforms).
4. Looks up the subfolder in `mappings` to find the topic ID.
5. Uploads via Telegram Bot API (`sendDocument` with `message_thread_id`).
6. On success: deletes the local file + shows a native OS notification.
7. On failure: logs the error + notifies, file is left in place for retry.

## Tray menu

| Item | Action |
|------|--------|
| ✅ Watching | Status indicator (disabled) |
| Open watch folder | Opens watch_dir in your file manager |
| Reload config | Re-reads config.yaml, restarts watcher |
| Quit | Graceful shutdown |

## Notes

- Unmapped subfolders are skipped with a log warning (file is left alone).
- Files exceeding `max_file_size_bytes` are skipped with a notification.
- Files placed directly in `watch_dir` (not in a subfolder) are skipped.
- New subfolders created after startup are watched automatically.
- Symlinks are not followed.

## Linux dependencies

`beeep` uses `libnotify` for notifications on Linux. Install if needed:
```bash
# Arch
sudo pacman -S libnotify

# Debian/Ubuntu
sudo apt install libnotify-bin
```

`systray` on Linux requires a system tray implementation. Most desktop
environments include one (GNOME needs the AppIndicator extension).
