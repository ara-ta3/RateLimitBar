package dialog

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func ChooseFile(ctx context.Context, prompt string) (string, error) {
	output, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", `
on run argv
try
    set selectedFile to choose file with prompt (item 1 of argv) with invisibles
    return POSIX path of selectedFile
on error number -128
    return ""
end try
end run`, prompt).Output()
	if err != nil {
		return "", fmt.Errorf("ファイル選択を開けませんでした: %w", err)
	}
	return strings.TrimSuffix(string(output), "\n"), nil
}

func ShowError(ctx context.Context, err error) error {
	return exec.CommandContext(ctx, "/usr/bin/osascript", "-e", `
on run argv
    display alert "RateLimitBar" message (item 1 of argv) as critical
end run`, err.Error()).Run()
}
