package menubar

import (
	"context"
	"sync"
	"time"

	"fyne.io/systray"
	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

type LanguageSettings struct {
	Current func() i18n.Language
	Change  func(context.Context, i18n.Language) bool
}

type localizedMenu struct {
	mu       sync.Mutex
	language LanguageSettings
	redraws  []func(i18n.Language)
	results  []usage.Result
	items    []TitleSetter
	panel    *usagePanel
}

func (m *localizedMenu) bind(redraw func(i18n.Language)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.redraws = append(m.redraws, redraw)
	redraw(m.language.Current())
}

func (m *localizedMenu) redraw() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.redrawLocked()
}

func (m *localizedMenu) update(results []usage.Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results = results
	m.redrawLocked()
}

func (m *localizedMenu) redrawLocked() {
	language := m.language.Current()
	for _, redraw := range m.redraws {
		redraw(language)
	}
	if m.panel != nil {
		m.panel.update(language, m.results)
	}
	if len(m.results) != len(m.items) {
		return
	}
	now := time.Now()
	for i, r := range m.results {
		m.items[i].SetTitle(formatLocalizedResultAt(r, now, language))
	}
}

func (m *localizedMenu) text(item TitleSetter, english, japanese string) {
	m.bind(func(language i18n.Language) { item.SetTitle(language.Text(english, japanese)) })
}

func (m *localizedMenu) window(item TitleSetter, name, label string) {
	m.bind(func(language i18n.Language) {
		item.SetTitle(name + " " + windowLabel(language, label))
	})
}

func (m *localizedMenu) addLanguageMenu(ctx context.Context) {
	menu := systray.AddMenuItem("", "")
	m.text(menu, "Language", "言語")
	english := menu.AddSubMenuItemCheckbox("English", "", false)
	japanese := menu.AddSubMenuItemCheckbox("日本語", "", false)
	if m.panel != nil {
		m.panel.command(func(l i18n.Language) popover.Command {
			return popover.Command{Kind: popover.LanguageCommand, Title: "English", Checked: l == i18n.English}
		}, english.ClickedCh)
		m.panel.command(func(l i18n.Language) popover.Command {
			return popover.Command{Kind: popover.LanguageCommand, Title: "日本語", Checked: l == i18n.Japanese}
		}, japanese.ClickedCh)
	}
	m.bind(func(language i18n.Language) {
		if language == i18n.English {
			english.Check()
			japanese.Uncheck()
		} else {
			japanese.Check()
			english.Uncheck()
		}
	})
	go func() {
		for {
			var language i18n.Language
			select {
			case <-ctx.Done():
				return
			case <-english.ClickedCh:
				language = i18n.English
			case <-japanese.ClickedCh:
				language = i18n.Japanese
			}
			m.language.Change(ctx, language)
			m.redraw()
		}
	}()
}

func windowLabel(language i18n.Language, label string) string {
	switch label {
	case provider.WindowWeekly:
		return language.Text(label, "週間")
	case provider.WindowMonthly:
		return language.Text(label, "月間")
	default:
		return label
	}
}
