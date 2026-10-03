package provider

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
)

var ErrExecutableUnavailable = errors.New("executable unavailable")

type ExecutablePath string

func ParseExecutablePath(path string) (ExecutablePath, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: 絶対パスを指定してください", ErrExecutableUnavailable)
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrExecutableUnavailable, err)
	}
	return ExecutablePath(resolved), nil
}

func ResolveExecutable(name, configured string) (ExecutablePath, error) {
	if configured != "" {
		return ParseExecutablePath(configured)
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrExecutableUnavailable, err)
	}
	return ExecutablePath(path), nil
}
