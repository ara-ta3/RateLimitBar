package provider_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ratelimitbar/internal/provider"
)

func TestResolveExecutableDiscoversCLIOnPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := provider.ResolveExecutable("codex", "")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != path {
		t.Fatalf("ResolveExecutable() = %q, want %q", got, path)
	}
}

func TestResolveExecutablePrefersSelectedPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	selected := filepath.Join(t.TempDir(), "codex with spaces")
	if err := os.WriteFile(selected, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := provider.ResolveExecutable("codex", selected)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != selected {
		t.Fatalf("ResolveExecutable() = %q, want %q", got, selected)
	}
}

func TestResolveExecutableDoesNotHideMissingSelectedPathWithFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	_, err := provider.ResolveExecutable("codex", filepath.Join(dir, "removed"))
	if !errors.Is(err, provider.ErrExecutableUnavailable) {
		t.Fatalf("ResolveExecutable() error = %v, want unavailable", err)
	}
}

func TestParseExecutablePathRejectsRelativePath(t *testing.T) {
	_, err := provider.ParseExecutablePath("codex")
	if !errors.Is(err, provider.ErrExecutableUnavailable) {
		t.Fatalf("ParseExecutablePath() error = %v, want unavailable", err)
	}
}

func TestParseExecutablePathRejectsNonExecutableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(path, []byte("not executable"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := provider.ParseExecutablePath(path)
	if !errors.Is(err, provider.ErrExecutableUnavailable) {
		t.Fatalf("ParseExecutablePath() error = %v, want unavailable", err)
	}
}
