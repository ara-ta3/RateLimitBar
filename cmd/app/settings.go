package main

import (
	"context"
	"fmt"
	"log"

	"ratelimitbar/internal/config"
	"ratelimitbar/internal/dialog"
	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/provider"
)

func reportSettingError(ctx context.Context, language i18n.Language, err error) {
	log.Printf("settings: %v", err)
	if ctx.Err() == nil {
		if alertErr := dialog.ShowError(ctx, fmt.Errorf("%s: %w", language.Text("Could not update settings", "設定を更新できませんでした"), err)); alertErr != nil {
			log.Printf("show settings error: %v", alertErr)
		}
	}
}

func cliAction(store *config.Store, name, command string, current func(config.Settings) string, change func(*config.Settings, string)) menubar.Action {
	return menubar.Action{
		Label: name + " CLI",
		Detail: func() string {
			path, err := provider.ResolveExecutable(command, current(store.Settings()))
			if err != nil {
				return store.Settings().Language.Text("Not found", "未検出")
			}
			return string(path)
		},
		Title: func() string {
			path, err := provider.ResolveExecutable(command, current(store.Settings()))
			if err != nil {
				return name + " CLI: " + store.Settings().Language.Text("Not found (Select…)", "未検出（指定…）")
			}
			return name + " CLI: " + string(path) + store.Settings().Language.Text(" (Change…)", "（変更…）")
		},
		OnClick: func(ctx context.Context) bool {
			path, err := dialog.ChooseFile(ctx, store.Settings().Language.Text("Select the "+name+" CLI executable. Press ⌘⇧G to enter a path.", name+" CLIの実行ファイルを選択してください。⌘⇧Gでパスを入力できます。"))
			if err == nil && path == "" {
				return false
			}
			var executable provider.ExecutablePath
			if err == nil {
				executable, err = provider.ParseExecutablePath(path)
			}
			if err == nil {
				err = store.Update(func(s *config.Settings) { change(s, string(executable)) })
			}
			if err != nil {
				reportSettingError(ctx, store.Settings().Language, err)
				return false
			}
			return true
		},
	}
}

func sourceActions(store *config.Store) []menubar.Action {
	return []menubar.Action{
		cliAction(store, "Codex", "codex", func(s config.Settings) string { return s.CodexPath }, func(s *config.Settings, path string) { s.CodexPath = path }),
		cliAction(store, "Claude", "claude", func(s config.Settings) string { return s.ClaudePath }, func(s *config.Settings, path string) { s.ClaudePath = path }),
		{
			Title: func() string {
				return store.Settings().Language.Text("Clear CLI paths (use automatic detection)", "取得元の指定をすべて解除（自動検出に戻す）")
			},
			OnClick: func(ctx context.Context) bool {
				if err := store.Update(func(s *config.Settings) {
					s.CodexPath = ""
					s.ClaudePath = ""
				}); err != nil {
					reportSettingError(ctx, store.Settings().Language, err)
					return false
				}
				return true
			},
		},
	}
}

func languageSettings(store *config.Store) menubar.LanguageSettings {
	return menubar.LanguageSettings{
		Current: func() i18n.Language { return store.Settings().Language },
		Change: func(ctx context.Context, language i18n.Language) bool {
			if err := store.Update(func(s *config.Settings) { s.Language = language }); err != nil {
				reportSettingError(ctx, store.Settings().Language, err)
				return false
			}
			return true
		},
	}
}
