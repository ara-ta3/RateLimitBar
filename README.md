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
Claude 5h 23% (2h30m) / W 48% (3d4h)  Codex 5h 61% (1h20m)  Cursor M 18% (12d3h)
```

*Example display. `W` means weekly and `M` means monthly. Percentages indicate usage consumed.*

## Features

- **Three services in one place** — View Claude, Codex, and Cursor usage.
- **Reset countdowns** — See the time remaining when a reset timestamp is available.
- **Customizable display** — Toggle individual usage windows for each service.
- **Automatic and manual refresh** — Updates every 60 seconds, with a `Refresh` menu item for immediate updates.
- **Menu bar app** — Launch from Finder without adding an icon to the Dock.

## Supported services

| Service | Usage windows | Data source and requirements |
| --- | --- | --- |
| Claude | 5-hour and weekly | `cachedUsageUtilization` in `~/.claude.json`. Requires a Claude Code usage cache |
| Codex | 5-hour, weekly, and model-specific limits when available | `codex app-server`. Requires the `codex` CLI to be installed and signed in |
| Cursor | Monthly | Access token from macOS Keychain and the usage API. Requires Cursor credentials stored in Keychain |

Claude normally reads its local cache. Cached data older than 15 minutes is marked `(stale)` in the dropdown menu. You can optionally enable [automatic cache refresh](#automatic-claude-cache-refresh).

## Getting started

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

Click the menu bar display to view usage details and settings. Some menu labels are currently in Japanese; the labels below match the app.

| Menu item | Action |
| --- | --- |
| Checkbox items for each service | Toggle individual usage windows in the menu bar |
| `Refresh` | Fetch usage again |
| 取得元の設定 (CLI source settings) | View or change the Codex and Claude CLI paths |
| Claudeのキャッシュを自動更新 (Automatically refresh Claude cache) | Update stale cache data using the Claude CLI |
| `Quit` | Exit the app |

All display items are enabled at startup. Display selections and the Claude automatic cache refresh toggle are not saved between launches.

Countdowns update when usage is fetched. Less than one minute is shown as `<1m`, and a reset time in the past is shown as `0m`. Windows without a reset timestamp show only the usage percentage. Before the first fetch, or when all display items are disabled, the menu bar shows `RateLimit`.

### Configure CLI paths

By default, the app looks for `codex` and `claude` on its `PATH`. When launched as a macOS app, it loads the login environment of `$SHELL`, falling back to `/bin/zsh` if unset.

If a CLI cannot be found, or you want to use a specific executable:

1. Open 取得元の設定 (CLI source settings).
2. Click 未検出（指定…） (Not found — choose…) or the displayed CLI path.
3. Select the executable. Press `⌘⇧G` in the file picker to enter a path directly.

Find CLI paths in your terminal:

```sh
command -v codex
command -v claude
```

Selected paths are saved in `~/Library/Application Support/RateLimitBar/settings.json` and reused on subsequent launches. Selecting a path also triggers a refresh.

If a selected executable is deleted, the app shows 未検出 (Not found). Select another executable, or choose 取得元の指定をすべて解除（自動検出に戻す） (Clear all selected paths — return to automatic detection) to resume searching `PATH`.

The Claude CLI is used only for automatic cache refresh; it is not required to display the existing cache. Cursor uses Keychain and its API, so it has no CLI path setting.

### Automatic Claude cache refresh

Enable Claudeのキャッシュを自動更新 (Automatically refresh Claude cache) to run the following command when the cache is older than 15 minutes, then read the cache again:

```sh
claude -p '/usage'
```

- Disabled at startup; you can toggle it while the app is running.
- Requires the `claude` CLI to be installed and signed in. Refreshing involves network communication.
- Runs in a temporary directory outside the project, with a 30-second timeout.
- On failure, the app displays a fetch error and retries on the next update.
- Missing or unreadable caches are not automatically refreshed.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| CLI not found | Select the executable under 取得元の設定 (CLI source settings). A `PATH` configured only in `~/.zshrc` is not loaded when launching from Finder |
| Claude shows `(stale)` | The cache is older than 15 minutes. Update it through Claude Code or enable automatic cache refresh |
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
internal/config/   CLI path persistence
internal/dialog/   File picker dialog
internal/menubar/  Menu bar display, interactions, and refresh
internal/provider/ Usage fetching for each service
internal/usage/    Usage data types
packaging/macos/   App icons, launcher, and Info.plist
```

## Feedback and contributing

Bug reports and feature requests are welcome in [Issues](https://github.com/ara-ta3/RateLimitBar/issues), and changes are welcome as pull requests. For bug reports, include your macOS version, the affected service, and steps to reproduce. Do not include authentication tokens or other secrets.
