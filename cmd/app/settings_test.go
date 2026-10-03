package main

import (
	"os"
	"path/filepath"
	"testing"

	"ratelimitbar/internal/config"
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
