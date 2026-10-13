package i18n

import (
	"encoding/json"
	"testing"
)

func TestParseJapanese(t *testing.T) {
	var language Language
	if err := json.Unmarshal([]byte(`"ja"`), &language); err != nil {
		t.Fatal(err)
	}
	if language != Japanese {
		t.Fatalf("language = %q", language)
	}
	if got := language.Text("Refresh", "更新"); got != "更新" {
		t.Fatalf("text = %q", got)
	}
}

func TestParseEnglish(t *testing.T) {
	var language Language
	if err := json.Unmarshal([]byte(`"en"`), &language); err != nil {
		t.Fatal(err)
	}
	if language != English {
		t.Fatalf("language = %q", language)
	}
	if got := language.Text("Refresh", "更新"); got != "Refresh" {
		t.Fatalf("text = %q", got)
	}
}

func TestRejectUnsupportedLanguage(t *testing.T) {
	var language Language
	if err := json.Unmarshal([]byte(`"fr"`), &language); err == nil {
		t.Fatal("expected unsupported language error")
	}
}
