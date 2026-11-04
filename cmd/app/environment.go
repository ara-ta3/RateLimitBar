package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func loadBundlePath(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if !isAppBundleExecutable(executable) {
		return nil
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	path, err := loginShellPath(ctx, shell)
	if err != nil {
		return err
	}
	return os.Setenv("PATH", path)
}

func isAppBundleExecutable(path string) bool {
	macos := filepath.Dir(path)
	contents := filepath.Dir(macos)
	return filepath.Base(macos) == "MacOS" && filepath.Base(contents) == "Contents" && filepath.Ext(filepath.Dir(contents)) == ".app"
}

func loginShellPath(ctx context.Context, shell string) (string, error) {
	cmd := exec.CommandContext(ctx, shell, "-l", "-c", `printf '\000%s\000' "$PATH"`)
	cmd.WaitDelay = 100 * time.Millisecond
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s: %w", shell, err)
	}
	parts := strings.Split(string(output), "\x00")
	if len(parts) != 3 || parts[1] == "" {
		return "", fmt.Errorf("%s returned no PATH", shell)
	}
	return parts[1], nil
}
