package claudeusage_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"ratelimitbar/internal/claudeusage"
)

var capturedAt = time.Unix(1700000000, 0).UTC()

const statusLineInput = `{"session_id":"private-session","transcript_path":"/private/transcript","model":{"display_name":"Opus"},"rate_limits":{"five_hour":{"used_percentage":23.4,"resets_at":1700006400},"seven_day":{"used_percentage":48,"resets_at":1700438400}}}`

func TestCaptureWritesOnlyUsageAndTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "with spaces", "claude-usage.json")
	var output bytes.Buffer
	if err := claudeusage.Capture(strings.NewReader(statusLineInput), &output, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"five_hour":{"used_percentage":23.4,"resets_at":1700006400},"seven_day":{"used_percentage":48,"resets_at":1700438400}}}` {
		t.Fatalf("cache = %s", got)
	}
	snapshot, err := claudeusage.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot, claudeusage.Snapshot{FetchedAt: capturedAt, FiveHour: &claudeusage.Window{UsedPercentage: 23.4, ResetsAt: time.Unix(1700006400, 0)}, SevenDay: &claudeusage.Window{UsedPercentage: 48, ResetsAt: time.Unix(1700438400, 0)}}) {
		t.Fatalf("Read() = %+v", snapshot)
	}
	if output.String() != "Claude 5h 23% / Weekly 48%\n" {
		t.Fatalf("output = %q", output.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestCaptureKeepsPreviousCacheWhenRateLimitsAreAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	if err := claudeusage.Capture(strings.NewReader(statusLineInput), &bytes.Buffer{}, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := claudeusage.Capture(strings.NewReader(`{"rate_limits":null}`), &output, path, capturedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("cache changed: %s", after)
	}
	if output.String() != "" {
		t.Fatalf("output = %q, want empty", output.String())
	}
}

func TestCaptureReplacesWindowsInsteadOfMergingOldValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	if err := claudeusage.Capture(strings.NewReader(statusLineInput), &bytes.Buffer{}, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := claudeusage.Capture(strings.NewReader(`{"rate_limits":{"seven_day":{"used_percentage":0,"resets_at":1700438400}}}`), &output, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	got, err := claudeusage.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, claudeusage.Snapshot{FetchedAt: capturedAt, SevenDay: &claudeusage.Window{UsedPercentage: 0, ResetsAt: time.Unix(1700438400, 0)}}) {
		t.Fatalf("Read() = %+v", got)
	}
	if output.String() != "Claude Weekly 0%\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestCaptureRejectsMalformedUsageWithoutOverwritingCache(t *testing.T) {
	cases := []string{
		`not json`,
		`{"rate_limits":`,
		`{"rate_limits":{"five_hour":{}}}`,
		`{"rate_limits":{"five_hour":{"used_percentage":null,"resets_at":1700006400}}}`,
		`{"rate_limits":{"five_hour":{"used_percentage":101,"resets_at":1700006400}}}`,
		`{"rate_limits":{"five_hour":{"used_percentage":-1,"resets_at":1700006400}}}`,
		`{"rate_limits":{"five_hour":{"used_percentage":"5","resets_at":1700006400}}}`,
		`{"rate_limits":{"five_hour":{"used_percentage":5,"resets_at":0}}}`,
		`{"rate_limits":{"seven_day":{"used_percentage":5,"resets_at":null}}}`,
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "claude-usage.json")
			if err := os.WriteFile(path, []byte("previous cache"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := claudeusage.Capture(strings.NewReader(input), &bytes.Buffer{}, path, capturedAt); err == nil {
				t.Fatal("Capture() error = nil")
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "previous cache" {
				t.Fatalf("cache = %q", got)
			}
		})
	}
}

func TestReadRejectsMalformedCache(t *testing.T) {
	for _, data := range []string{``, `null`, `{}`, `{"fetched_at":"invalid","rate_limits":{}}`, `{"rate_limits":{}}`, `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"five_hour":{"used_percentage":5}}}`} {
		t.Run(data, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "claude-usage.json")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := claudeusage.Read(path); err == nil {
				t.Fatal("Read() error = nil")
			}
		})
	}
}

func TestCaptureReportsWriteFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := claudeusage.Capture(strings.NewReader(statusLineInput), &bytes.Buffer{}, filepath.Join(path, "cache.json"), capturedAt); err == nil {
		t.Fatal("Capture() error = nil for invalid cache directory")
	}
}

func TestCaptureRoundsHalfPercentUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	var output bytes.Buffer
	if err := claudeusage.Capture(strings.NewReader(`{"rate_limits":{"five_hour":{"used_percentage":24.5,"resets_at":1700006400}}}`), &output, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	if output.String() != "Claude 5h 25%\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestCaptureEmptyRateLimitsClearsPreviousWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	if err := claudeusage.Capture(strings.NewReader(statusLineInput), &bytes.Buffer{}, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := claudeusage.Capture(strings.NewReader(`{"rate_limits":{}}`), &output, path, capturedAt); err != nil {
		t.Fatal(err)
	}
	got, err := claudeusage.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, claudeusage.Snapshot{FetchedAt: capturedAt}) {
		t.Fatalf("Read() = %+v", got)
	}
	if output.String() != "" {
		t.Fatalf("output = %q", output.String())
	}
}
