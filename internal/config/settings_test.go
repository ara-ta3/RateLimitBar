package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"ratelimitbar/internal/config"
)

func TestMissingSettingsUseAutomaticDiscovery(t *testing.T) {
	store, err := config.NewStore(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Settings(); got != (config.Settings{}) {
		t.Fatalf("Settings() = %+v, want automatic discovery", got)
	}
}

func TestCLIPathsSurviveRestartAndUpdatingClaudeKeepsCodex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "RateLimitBar", "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { s.CodexPath = "/cli/codex" }); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { s.ClaudePath = "/cli/claude" }); err != nil {
		t.Fatal(err)
	}
	restarted, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := restarted.Settings(); got != (config.Settings{CodexPath: "/cli/codex", ClaudePath: "/cli/claude"}) {
		t.Fatalf("restarted Settings() = %+v", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("settings permissions = %o, want 600", got)
	}
}

func TestResetPersistsAutomaticDiscovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := config.Save(path, config.Settings{CodexPath: "/cli/codex", ClaudePath: "/cli/claude"}); err != nil {
		t.Fatal(err)
	}
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { *s = config.Settings{} }); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != (config.Settings{}) {
		t.Fatalf("Load() = %+v, want automatic discovery", got)
	}
}

func TestFailedSaveKeepsCurrentPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { s.CodexPath = "/cli/codex" }); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(s *config.Settings) { s.CodexPath = "/other/codex" }); err == nil {
		t.Fatal("Update() error = nil, want failed rename")
	}
	if got := store.Settings(); got != (config.Settings{CodexPath: "/cli/codex"}) {
		t.Fatalf("Settings() after failed save = %+v", got)
	}
}

func TestMalformedSettingsReportAnErrorAndAllowReselection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"codexPath":`), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.NewStore(path)
	if err == nil {
		t.Fatal("NewStore() error = nil, want malformed JSON error")
	}
	if err := store.Update(func(s *config.Settings) { s.CodexPath = "/cli/codex" }); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != (config.Settings{CodexPath: "/cli/codex"}) {
		t.Fatalf("Load() after reselection = %+v", got)
	}
}
