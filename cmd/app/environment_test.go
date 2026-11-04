package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginShellPathPreservesSpacesAndNewlines(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sh -c \"$3\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/cli with spaces:/cli\nwith newline:/usr/bin")
	path, err := loginShellPath(context.Background(), shell)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/cli with spaces:/cli\nwith newline:/usr/bin" {
		t.Fatalf("PATH = %q", path)
	}
}

func TestLoginShellPathIgnoresStartupAndLogoutOutput(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\n[ \"$1\" = -l ] && [ \"$2\" = -c ] || exit 1\nprintf 'welcome\\n\\000/custom/bin:/usr/bin\\000goodbye\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	path, err := loginShellPath(context.Background(), shell)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/custom/bin:/usr/bin" {
		t.Fatalf("PATH = %q, want /custom/bin:/usr/bin", path)
	}
}

func TestLoginShellPathRejectsFailedShell(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := loginShellPath(context.Background(), shell)
	if err == nil {
		t.Fatal("want shell failure")
	}
}

func TestLoginShellPathRejectsMissingPath(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nprintf 'welcome only'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := loginShellPath(context.Background(), shell)
	if err == nil {
		t.Fatal("want missing PATH error")
	}
}

func TestLoginShellPathStopsWaitingForShell(t *testing.T) {
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := loginShellPath(ctx, shell)
	if err == nil {
		t.Fatal("want timeout error")
	}
	if ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("context error = %v, want deadline exceeded", ctx.Err())
	}
}

func TestAppBundleExecutableIsRecognized(t *testing.T) {
	if isAppBundleExecutable("/Applications/Rate Limit Bar.app/Contents/MacOS/ratelimitbar") != true {
		t.Fatal("want app bundle executable")
	}
}

func TestTerminalExecutableDoesNotLoadLoginShell(t *testing.T) {
	if isAppBundleExecutable("/work/RateLimitBar/bin/ratelimitbar") != false {
		t.Fatal("want terminal executable")
	}
}
