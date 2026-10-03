package main

import (
	"context"
	"fmt"
	"log"

	"ratelimitbar/internal/config"
	"ratelimitbar/internal/dialog"
	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/provider"
)

func reportSettingError(ctx context.Context, err error) {
	log.Printf("settings: %v", err)
	if ctx.Err() == nil {
		if alertErr := dialog.ShowError(ctx, err); alertErr != nil {
			log.Printf("show settings error: %v", alertErr)
		}
	}
}

func cliAction(store *config.Store, name, command string, current func(config.Settings) string, change func(*config.Settings, string)) menubar.Action {
	return menubar.Action{
		Title: func() string {
			path, err := provider.ResolveExecutable(command, current(store.Settings()))
			if err != nil {
				return name + " CLI: 未検出（指定…）"
			}
			return name + " CLI: " + string(path) + "（変更…）"
		},
		OnClick: func(ctx context.Context) bool {
			path, err := dialog.ChooseFile(ctx, name+" CLIの実行ファイルを選択してください。⌘⇧Gでパスを入力できます。")
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
				reportSettingError(ctx, err)
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
			Title: func() string { return "取得元の指定をすべて解除（自動検出に戻す）" },
			OnClick: func(ctx context.Context) bool {
				if err := store.Update(func(s *config.Settings) { *s = config.Settings{} }); err != nil {
					reportSettingError(ctx, fmt.Errorf("指定を解除できませんでした: %w", err))
					return false
				}
				return true
			},
		},
	}
}
