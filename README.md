<p align="center">
  <img src="packaging/macos/AppIcon.png" width="128" alt="RateLimitBar icon">
</p>

<h1 align="center">RateLimitBar</h1>

<p align="center">English | <a href="README.ja.md">日本語</a></p>

<p align="center">Claude, Codex, and Cursor usage at a glance in your macOS menu bar.</p>

<p align="center">
  <a href="https://github.com/ara-ta3/RateLimitBar/actions/workflows/ci.yml"><img src="https://github.com/ara-ta3/RateLimitBar/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <img src="https://img.shields.io/badge/platform-macOS-black" alt="macOS">
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go 1.27">
</p>

RateLimitBar is a macOS menu bar app that displays usage percentages and the time remaining until rate limits reset for AI coding tools. It refreshes every 60 seconds and lets you choose which services and usage windows to display.

```text
Claude 5h ━╸─── Weekly ━━━╸─  Codex 5h ╸────  Cursor Monthly ━━━──
```

*Schematic menu bar display. Full service names and window labels identify each bar. Each checked window has its own bar: select the five-hour window, weekly window, both, or a model-specific window. Click to see exact percentages and reset countdowns.*

## Features

- **Three services in one place** — View Claude, Codex, and Cursor usage.
- **Reset countdowns** — See the time remaining when a reset timestamp is available.
- **Customizable display** — Toggle individual usage windows for each service.
- **Automatic and manual refresh** — Updates every 60 seconds, with a `Refresh` menu item for immediate updates.
- **Menu bar app** — Launch from Finder without adding an icon to the Dock.

## Supported services

| Service | Usage windows | Data source and requirements |
| --- | --- | --- |
| Claude | 5-hour and weekly | Usage saved by Claude Code’s `statusLine`. Requires [integration setup](#connect-claude-code) |
| Codex | 5-hour, weekly, and model-specific limits when available | `codex app-server`. Requires the `codex` CLI to be installed and signed in |
| Cursor | Monthly | Access token from macOS Keychain and the usage API. Requires Cursor credentials stored in Keychain |

Claude runs interactive `claude /usage` on refresh and reads the usage saved by statusLine. Failed refreshes and data older than 15 minutes are marked `(stale)`. The panel shows the last received time and recovery steps.

## Getting started

### Install a release

Download the ZIP for your Mac's CPU from [Releases](https://github.com/ara-ta3/RateLimitBar/releases), extract it, and copy `RateLimitBar.app` to `/Applications`.

- Apple Silicon (M-series): `RateLimitBar-v<version>-macOS-arm64.zip`
- Intel Mac: `RateLimitBar-v<version>-macOS-amd64.zip`

Release builds target macOS 14 or later. Go and Xcode are not required, but you still need a signed-in CLI or local credentials for the services you use.

These developer releases are not Developer ID signed or notarized. If macOS blocks the app, try opening it first, then allow it in System Settings → Privacy & Security.

### Requirements

- macOS
- Go 1.27 or later
- Xcode Command Line Tools (install with `xcode-select --install`)
- A signed-in CLI or local credentials for the services you use (see the table above)

### Build and launch from source

```sh
git clone https://github.com/ara-ta3/RateLimitBar.git
cd RateLimitBar
make app
open dist/RateLimitBar.app
```

This generates `dist/RateLimitBar.app` for the CPU architecture of the Mac used to build it. You can launch it by double-clicking in Finder or copy it to `/Applications`.

> The generated app is intended for local use. It is not Developer ID signed or notarized.

## Usage

Click the menu bar display to open a panel with usage bars and reset countdowns for each service. Bars and colors represent the used percentage for every service: green below 70%, yellow from 70%, and red from 90%. Both used and remaining percentages are labeled explicitly. Check or uncheck a usage window to show or hide its own bar in the menu bar. Fetching and panel updates continue when a window is unchecked. The panel’s Settings… button opens a separate window for language selection and the Codex CLI source. Clicking outside keeps the settings window open; selecting Settings… again brings the same window to the front. Right-click to open the traditional menu.

| Menu item | Action |
| --- | --- |
| Checkbox items for each service | Toggle individual usage windows in the menu bar |
| `Refresh` | Fetch usage again |
| CLI source settings / 取得元の設定 | View or change the Codex CLI path |
| `Quit` | Exit the app |
| 言語 / Language | Switch between 日本語 and English; the selection is saved |

All display items are enabled at startup. Display selections are not saved between launches.

Countdowns update when usage is fetched; the panel also updates its countdowns when opened. Less than one minute is shown as `<1m`, and a reset time in the past is shown as `0m`. Windows without a reset timestamp show only the usage percentage. Before the first fetch, or when all display items are disabled, the menu bar shows a chart icon. Failed or missing data is shown as a dash, while stale usage bars are gray. Hover to see percentages for the selected windows and their fetch status.

### Display language

Select **言語 → English** or **Language → 日本語** to switch the menu, usage status, and file selection prompts immediately. The default is Japanese. Your selection is saved in `settings.json` and restored on the next launch. Service names and compact time units (`5h`, `1h30m`) stay the same.

### Configure CLI paths

By default, the app looks for `codex` on its `PATH`. When launched as a macOS app, it loads the login environment of `$SHELL`, falling back to `/bin/zsh` if unset.

If a CLI cannot be found, or you want to use a specific executable:

1. Open CLI source settings / 取得元の設定.
2. Click Not found (Select…) or the displayed CLI path.
3. Select the executable. Press `⌘⇧G` in the file picker to enter a path directly.

Find CLI paths in your terminal:

```sh
command -v codex
```

Selected paths are saved in `~/Library/Application Support/RateLimitBar/settings.json` and reused on subsequent launches. Selecting a path also triggers a refresh.

If a selected executable is deleted, the app shows Not found. Select another executable, or choose Clear CLI paths (use automatic detection) to resume searching `PATH`.

Claude uses the statusLine integration below, and Cursor uses Keychain and its API. Neither needs a CLI path setting.

### Connect Claude Code

Place `RateLimitBar.app` in `/Applications` and merge the following into your existing `~/.claude/settings.json`. Do not replace the entire settings file. If you already have a statusLine, use the wrapper below to keep its display.

```json
{
  "statusLine": {
    "type": "command",
    "command": "\"/Applications/RateLimitBar.app/Contents/MacOS/ratelimitbar\" claude-statusline"
  }
}
```

For source builds, run `make build` and use the absolute path to `bin/ratelimitbar`. Use the binary, not the `launch` script.

After configuring statusLine, run the following once in a terminal to complete login and workspace trust for the dedicated cache folder. After `/usage` loads, press Esc and check that the status line shows text such as `Claude 5h 5% / Weekly 6%`, then enter `/exit`.

```sh
mkdir -p ~/Library/Caches/RateLimitBar
cd ~/Library/Caches/RateLimitBar
claude /usage
```

The official docs cover Pro and Max; Team was also verified locally. Environments without `rate_limits`, such as API key billing, cannot provide this data. See the [official statusLine specification](https://code.claude.com/docs/en/statusline#rate-limit-usage).

- The cache is `~/Library/Caches/RateLimitBar/claude-usage.json`. Only usage percentages, reset times, and the received timestamp are saved, with `0600` permissions.
- At startup, every 60 seconds, and on Refresh, RateLimitBar launches interactive `claude /usage` in a pseudo-terminal. It closes the usage screen, waits for statusLine to save new data, and terminates the process it started. Each attempt has a 20-second timeout.
- `claude` is resolved from PATH and runs in the dedicated cache folder with additional MCP servers disabled. Existing Claude sessions are untouched. User-configured startup hooks and the statusLine command still run.
- `/usage` does not request model inference, but starts Claude and requests usage data. `claude -p /usage` returns session costs and token counts and is not used here.
- If refresh fails, saved values and their received time are retained and marked stale. Data older than 15 minutes is also stale; expired windows are omitted.
- StatusLine input without `rate_limits` leaves saved values and their timestamp untouched. The old `~/.claude.json` usage cache and legacy `claudePath` setting are not used.

#### When Claude shows stale data

1. Run `claude /usage` in the dedicated cache folder above to check login, workspace trust, and statusLine configuration. Press Esc to close the usage screen.
2. Select Refresh in RateLimitBar. Check that `claude` is on PATH and the configured statusLine binary exists.
3. Open usage page in the Claude card opens [Claude’s usage screen](https://claude.ai/settings/usage). Setup & help opens these instructions.

Opening the usage page does not update RateLimitBar. Last received is the time statusLine delivered data.

#### Automatic updates

RateLimitBar runs a new `/usage` every 60 seconds even when you have no Claude Code session open. No `statusLine.refreshInterval` setting is needed. This integration depends on interactive UI behavior; if a Claude Code update breaks it, saved values remain visible as stale data.

To keep an existing statusLine, save this wrapper script and set its absolute path as `statusLine.command`. Replace the last command with your existing statusLine command and make the wrapper executable with `chmod +x`.

```sh
#!/bin/sh
payload=$(cat)
printf '%s' "$payload" | "/Applications/RateLimitBar.app/Contents/MacOS/ratelimitbar" claude-statusline >/dev/null
printf '%s' "$payload" | /absolute/path/to/your-existing-statusline
```

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| CLI not found | Select the executable under CLI source settings / 取得元の設定. A `PATH` configured only in `~/.zshrc` is not loaded when launching from Finder |
| Claude shows `(stale)` | Refresh failed or data is over 15 minutes old. Run `/usage` in the cache folder above to check login, workspace trust, and statusLine |
| Claude fails to fetch | Check login and workspace trust in the cache folder, statusLine configuration, and the Claude and integration binaries. Usage is unavailable without a saved capture or after all saved windows expire |
| `Failed to fetch` | Check the service's authentication and data source, then select `Refresh`. Cursor requires Keychain credentials and access to its usage API |

## Development

```sh
make build      # Build bin/ratelimitbar
make run        # Build and launch from the terminal
make app        # Build dist/RateLimitBar.app
make test       # Run tests
make fmt/check  # Apply Go formatting and check for no changes
```

When launched from the terminal, you can also exit with `Ctrl+C`. CI runs formatting checks, builds, and tests on macOS.

### Project structure

```text
cmd/app/           Entry point and settings actions
internal/claudeusage/ statusLine usage parsing and persistence
internal/config/   CLI path persistence
internal/dialog/   File picker dialog
internal/menubar/  Menu bar display, interactions, and refresh
internal/provider/ Usage fetching for each service
internal/usage/    Usage data types
packaging/macos/   App icons, launcher, and Info.plist
```

### Releases

Merge the changes containing the Release workflow into `main`, then push a `v<major>.<minor>.<patch>` tag pointing to the release commit. The initial version is `v0.1.0`.

```sh
git switch main
git pull --ff-only origin main
git tag -a v0.1.0 -m 'Release v0.1.0'
git push origin v0.1.0
```

GitHub Actions checks formatting, runs tests, and builds the app on both Apple Silicon and Intel runners. After both builds succeed, it publishes a GitHub Release with both ZIPs and `SHA256SUMS.txt`. The app version is taken from the tag. No signing secrets are needed.

## Feedback and contributing

Bug reports and feature requests are welcome in [Issues](https://github.com/ara-ta3/RateLimitBar/issues), and changes are welcome as pull requests. For bug reports, include your macOS version, the affected service, and steps to reproduce. Do not include authentication tokens or other secrets.
