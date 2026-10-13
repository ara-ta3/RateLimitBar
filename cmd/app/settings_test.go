package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ratelimitbar/internal/config"
	"ratelimitbar/internal/i18n"
)

func TestCLISettingsDisplayDiscoveredPaths(t *testing.T) {
	dir := t.TempDir()
	codexPath := filepath.Join(dir, "codex")
	claudePath := filepath.Join(dir, "claude")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claudePath, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	store, err := config.NewStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	actions := sourceActions(store)
	if got := actions[0].Title(); got != "Codex CLI: "+codexPath+"（変更…）" {
		t.Fatalf("Codex title = %q", got)
	}
	if got := actions[1].Title(); got != "Claude CLI: "+claudePath+"（変更…）" {
		t.Fatalf("Claude title = %q", got)
	}
}

func TestCLISettingsDisplaySelectedPathAndReportMissingCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	store, err := config.NewStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	actions := sourceActions(store)
	if got := actions[0].Title(); got != "Codex CLI: 未検出（指定…）" {
		t.Fatalf("Codex title = %q", got)
	}
	if got := actions[1].Title(); got != "Claude CLI: 未検出（指定…）" {
		t.Fatalf("Claude title = %q", got)
	}
	path := filepath.Join(t.TempDir(), "codex with spaces")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { s.CodexPath = path }); err != nil {
		t.Fatal(err)
	}
	if got := actions[0].Title(); got != "Codex CLI: "+path+"（変更…）" {
		t.Fatalf("selected Codex title = %q", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got := actions[0].Title(); got != "Codex CLI: 未検出（指定…）" {
		t.Fatalf("removed Codex title = %q", got)
	}
}

func TestSwitchLanguagePersistsAndClearingPathsKeepsLanguage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { s.CodexPath = "/cli/codex"; s.ClaudePath = "/cli/claude" }); err != nil {
		t.Fatal(err)
	}
	if !languageSettings(store).Change(context.Background(), i18n.English) {
		t.Fatal("language change failed")
	}
	restarted, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := restarted.Settings(); got != (config.Settings{CodexPath: "/cli/codex", ClaudePath: "/cli/claude", Language: i18n.English}) {
		t.Fatalf("settings = %+v", got)
	}
	actions := sourceActions(restarted)
	if got := actions[2].Title(); got != "Clear CLI paths (use automatic detection)" {
		t.Fatalf("title = %q", got)
	}
	if !actions[2].OnClick(context.Background()) {
		t.Fatal("reset failed")
	}
	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved != (config.Settings{Language: i18n.English}) {
		t.Fatalf("settings after reset = %+v", saved)
	}
	if !languageSettings(restarted).Change(context.Background(), i18n.Japanese) {
		t.Fatal("Japanese change failed")
	}
	saved, err = config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Language != i18n.Japanese {
		t.Fatalf("saved language = %q", saved.Language)
	}
}
