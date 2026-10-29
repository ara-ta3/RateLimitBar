package provider

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/creack/pty"

	"ratelimitbar/internal/claudeusage"
)

const DefaultClaudeTimeout = 20 * time.Second

func refreshClaude(ctx context.Context, path string) error {
	executable, err := ResolveExecutable("claude", "")
	if err != nil {
		return err
	}
	cmd := exec.Command(string(executable), "/usage", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`)
	cmd.Dir = filepath.Dir(path)
	if err := os.MkdirAll(cmd.Dir, 0700); err != nil {
		return err
	}
	return runClaudeUsage(ctx, cmd, path)
}

func runClaudeUsage(ctx context.Context, cmd *exec.Cmd, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	startedAt := time.Now()
	terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 120})
	if err != nil {
		return fmt.Errorf("start Claude /usage: %w", err)
	}
	drained := make(chan struct{})
	go func() {
		io.Copy(io.Discard, terminal)
		close(drained)
	}()
	exited := make(chan error, 1)
	go func() {
		exited <- cmd.Wait()
		close(exited)
	}()
	defer func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		terminal.Close()
		<-exited
		<-drained
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Claude /usage did not update the statusLine cache: %w", ctx.Err())
		case err := <-exited:
			return fmt.Errorf("Claude /usage exited before updating the statusLine cache: %v", err)
		case <-ticker.C:
			// /usage の画面を閉じると、取得した使用率が statusLine に渡される。
			if _, err := terminal.Write([]byte{'\x1b'}); err != nil {
				return fmt.Errorf("close Claude /usage screen: %w", err)
			}
			snapshot, err := claudeusage.Read(path)
			if err == nil && snapshot.FetchedAt.After(startedAt) && (snapshot.FiveHour != nil || snapshot.SevenDay != nil) {
				return nil
			}
		}
	}
}
