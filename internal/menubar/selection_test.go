package menubar_test

import (
	"sync"
	"testing"

	"ratelimitbar/internal/menubar"
)

func testSources() []menubar.Source {
	return []menubar.Source{
		{Name: "Claude", Windows: []string{"5h", "Weekly"}},
		{Name: "Codex", Windows: []string{"5h", "Weekly"}},
		{Name: "Cursor", Windows: []string{"Monthly"}},
	}
}

func TestSelectionStartsWithEveryWindowEnabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())

	for _, src := range testSources() {
		for _, label := range src.Windows {
			if !sel.Enabled(src.Name, label) {
				t.Errorf("Enabled(%q, %q) = false, want true initially", src.Name, label)
			}
		}
	}
}

func TestSelectionToggleFlipsStateAndReturnsNewState(t *testing.T) {
	sel := menubar.NewSelection(testSources())

	if got := sel.Toggle("Claude", "Weekly"); got {
		t.Errorf("first Toggle() = %v, want false (turned off)", got)
	}
	if sel.Enabled("Claude", "Weekly") {
		t.Error("Enabled() = true after turning off")
	}
	if got := sel.Toggle("Claude", "Weekly"); !got {
		t.Errorf("second Toggle() = %v, want true (turned on again)", got)
	}
	if !sel.Enabled("Claude", "Weekly") {
		t.Error("Enabled() = false after turning on again")
	}
}

func TestSelectionToggleOnlyAffectsTheNamedWindow(t *testing.T) {
	sel := menubar.NewSelection(testSources())

	sel.Toggle("Claude", "5h")

	if sel.Enabled("Claude", "5h") {
		t.Error("Claude 5h should be off")
	}
	if !sel.Enabled("Claude", "Weekly") {
		t.Error("Claude Weekly must stay on")
	}
	if !sel.Enabled("Codex", "5h") {
		t.Error("Codex 5h has the same label but a different provider, it must stay on")
	}
	if !sel.Enabled("Cursor", "Monthly") {
		t.Error("Cursor Monthly must stay on")
	}
}

func TestSelectionInstancesDoNotShareState(t *testing.T) {
	a := menubar.NewSelection(testSources())
	b := menubar.NewSelection(testSources())

	a.Toggle("Claude", "5h")

	if !b.Enabled("Claude", "5h") {
		t.Error("toggling one Selection must not change another one (state is not persisted or shared)")
	}
}

func TestSelectionConcurrentTogglesAreConsistent(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	const pairs = 100
	var wg sync.WaitGroup
	for i := 0; i < pairs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sel.Toggle("Claude", "Weekly")
			_ = sel.Enabled("Claude", "Weekly")
			sel.Toggle("Claude", "Weekly")
		}()
	}
	wg.Wait()

	if !sel.Enabled("Claude", "Weekly") {
		t.Error("an even number of toggles must leave the window enabled")
	}
}
