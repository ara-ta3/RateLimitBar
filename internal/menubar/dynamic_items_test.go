package menubar_test

import (
	"errors"
	"slices"
	"testing"

	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/usage"
)

type addedCheckbox struct{ provider, label string }

type fakeCheckboxAdder struct{ added []addedCheckbox }

func (f *fakeCheckboxAdder) AddCheckbox(provider, label string) {
	f.added = append(f.added, addedCheckbox{provider, label})
}

func codexResult(windows ...usage.Window) usage.Result { return success("Codex", windows...) }

var (
	codexFiveHour = usage.Window{Label: "5h", UsedPercent: 61}
	codexWeekly   = usage.Window{Label: "Weekly", UsedPercent: 37}
	sparkFiveHour = usage.Window{Label: "GPT-5.3-Codex-Spark 5h", UsedPercent: 30}
)

func TestDynamicItemsAddsNothingForFixedLabels(t *testing.T) {
	adder := &fakeCheckboxAdder{}
	items := menubar.NewDynamicItems(menubar.NewSelection(testSources()), adder)

	items.Apply(allSucceeded())

	if len(adder.added) != 0 {
		t.Errorf("added = %v, want none for labels already declared by the sources", adder.added)
	}
}

func TestDynamicItemsAddsAnItemWhenANewLabelAppearsOnLaterRefresh(t *testing.T) {
	adder := &fakeCheckboxAdder{}
	items := menubar.NewDynamicItems(menubar.NewSelection(testSources()), adder)

	items.Apply([]usage.Result{codexResult(codexFiveHour, codexWeekly)})
	if len(adder.added) != 0 {
		t.Fatalf("added after first refresh = %v, want none", adder.added)
	}
	items.Apply([]usage.Result{codexResult(codexFiveHour, codexWeekly, sparkFiveHour)})

	want := []addedCheckbox{{"Codex", "GPT-5.3-Codex-Spark 5h"}}
	if !slices.Equal(adder.added, want) {
		t.Errorf("added = %v, want %v", adder.added, want)
	}
}

func TestDynamicItemsDoesNotAddTheSameLabelTwice(t *testing.T) {
	adder := &fakeCheckboxAdder{}
	items := menubar.NewDynamicItems(menubar.NewSelection(testSources()), adder)
	result := codexResult(codexFiveHour, sparkFiveHour)

	items.Apply([]usage.Result{result})
	items.Apply([]usage.Result{result})
	items.Apply([]usage.Result{codexResult(codexFiveHour)})
	items.Apply([]usage.Result{result})

	if len(adder.added) != 1 {
		t.Errorf("added = %v, want exactly one item (items are kept once added)", adder.added)
	}
}

func TestDynamicItemsKeysByProviderAndLabel(t *testing.T) {
	adder := &fakeCheckboxAdder{}
	items := menubar.NewDynamicItems(menubar.NewSelection(testSources()), adder)

	items.Apply([]usage.Result{
		codexResult(sparkFiveHour),
		success("Claude", sparkFiveHour),
	})

	want := []addedCheckbox{
		{"Codex", "GPT-5.3-Codex-Spark 5h"},
		{"Claude", "GPT-5.3-Codex-Spark 5h"},
	}
	if !slices.Equal(adder.added, want) {
		t.Errorf("added = %v, want %v", adder.added, want)
	}
}

func TestDynamicItemsIgnoresFailedResults(t *testing.T) {
	adder := &fakeCheckboxAdder{}
	items := menubar.NewDynamicItems(menubar.NewSelection(testSources()), adder)

	items.Apply([]usage.Result{{Provider: "Codex", Err: errors.New("down")}})

	if len(adder.added) != 0 {
		t.Errorf("added = %v, want none for a failed result", adder.added)
	}
}

func TestDynamicLabelIsOnByDefaultAndShownInTitle(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	items := menubar.NewDynamicItems(sel, &fakeCheckboxAdder{})
	results := []usage.Result{codexResult(codexFiveHour, sparkFiveHour)}

	items.Apply(results)

	if !sel.Enabled("Codex", "GPT-5.3-Codex-Spark 5h") {
		t.Error("dynamic label must start enabled")
	}
	want := "Codex 5h 61% / GPT-5.3-Codex-Spark 5h 30%"
	if got := menubar.FormatTitle(results, sel); got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestToggleOfDynamicLabelAppliesToTheTitle(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	items := menubar.NewDynamicItems(sel, &fakeCheckboxAdder{})
	results := []usage.Result{codexResult(codexFiveHour, sparkFiveHour)}
	items.Apply(results)

	if sel.Toggle("Codex", "GPT-5.3-Codex-Spark 5h") {
		t.Fatal("Toggle() = true, want turned off")
	}

	if got, want := menubar.FormatTitle(results, sel), "Codex 5h 61%"; got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestDynamicLabelKeepsProviderVisibleWhenOnlyItIsEnabledAndFetchFails(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	items := menubar.NewDynamicItems(sel, &fakeCheckboxAdder{})
	items.Apply([]usage.Result{codexResult(codexFiveHour, codexWeekly, sparkFiveHour)})
	sel.Toggle("Codex", "5h")
	sel.Toggle("Codex", "Weekly")

	failed := []usage.Result{{Provider: "Codex", Err: errors.New("down")}}

	if got, want := menubar.FormatTitle(failed, sel), "Codex --"; got != want {
		t.Errorf("FormatTitle() = %q, want %q (an enabled dynamic label counts for the provider)", got, want)
	}
}

func TestProviderWithEveryLabelDisabledIsHiddenOnFailureEvenWithDynamicLabels(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	items := menubar.NewDynamicItems(sel, &fakeCheckboxAdder{})
	items.Apply([]usage.Result{codexResult(codexFiveHour, codexWeekly, sparkFiveHour)})
	for _, label := range []string{"5h", "Weekly", "GPT-5.3-Codex-Spark 5h"} {
		sel.Toggle("Codex", label)
	}

	failed := []usage.Result{{Provider: "Codex", Err: errors.New("down")}}

	if got := menubar.FormatTitle(failed, sel); got != "RateLimit" {
		t.Errorf("FormatTitle() = %q, want the fixed title", got)
	}
}
