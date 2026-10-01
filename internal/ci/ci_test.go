package ci_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const repoRoot = "../.."

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatalf("%s を読めません: %v", rel, err)
	}
	return string(b)
}

func newFmtCheckDir(t *testing.T, goSource string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(readRepoFile(t, "Makefile")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(goSource), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runFmtCheck(t *testing.T, goSource string) (string, error) {
	t.Helper()
	cmd := exec.Command("make", "fmt-check")
	cmd.Dir = newFmtCheckDir(t, goSource)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestFmtCheckPassesForFormattedFiles(t *testing.T) {
	out, err := runFmtCheck(t, "package sample\n\nfunc F() {}\n")
	if err != nil {
		t.Fatalf("整形済みなのに失敗: %v\n%s", err, out)
	}
}

func TestFmtCheckFailsAndNamesUnformattedFile(t *testing.T) {
	out, err := runFmtCheck(t, "package sample\nfunc  F( ) {\n}\n")
	if err == nil {
		t.Fatalf("未整形なのに成功しました\n%s", out)
	}
	if !strings.Contains(out, "sample.go") {
		t.Fatalf("出力に未整形ファイル名がありません:\n%s", out)
	}
}

func TestFmtCheckRunsEvenWhenFileNamedFmtCheckExists(t *testing.T) {
	dir := newFmtCheckDir(t, "package sample\nfunc  F( ) {\n}\n")
	if err := os.WriteFile(filepath.Join(dir, "fmt-check"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("make", "fmt-check")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("同名ファイルがあるため recipe が実行されず成功しました\n%s", out)
	}
	if !strings.Contains(string(out), "sample.go") {
		t.Fatalf("出力に未整形ファイル名がありません:\n%s", out)
	}
}
