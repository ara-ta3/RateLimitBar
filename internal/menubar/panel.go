package menubar

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/usage"
)

type usagePanel struct {
	mu       sync.Mutex
	sources  []Source
	sel      *Selection
	windows  map[windowKey]string
	actions  map[string]chan<- struct{}
	commands []func(i18n.Language) popover.Command
	refresh  string
	quit     string
}

func newUsagePanel(sources []Source, sel *Selection) *usagePanel {
	return &usagePanel{sources: sources, sel: sel, windows: make(map[windowKey]string), actions: make(map[string]chan<- struct{})}
}

func (p *usagePanel) bind(clicked chan<- struct{}) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := strconv.Itoa(len(p.actions))
	p.actions[id] = clicked
	return id
}

func (p *usagePanel) window(provider, label string, clicked chan<- struct{}) {
	id := p.bind(clicked)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.windows[windowKey{provider, label}] = id
}

func (p *usagePanel) command(render func(i18n.Language) popover.Command, clicked chan<- struct{}) {
	id := p.bind(clicked)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.commands = append(p.commands, func(language i18n.Language) popover.Command {
		command := render(language)
		command.Action = id
		return command
	})
}

func (p *usagePanel) update(language i18n.Language, results []usage.Result) {
	popover.Update(p.model(language, results, time.Now()))
}

func (p *usagePanel) model(language i18n.Language, results []usage.Result, now time.Time) popover.Model {
	p.mu.Lock()
	defer p.mu.Unlock()
	model := popover.Model{
		Settings:      language.Text("Settings…", "設定…"),
		SettingsTitle: language.Text("RateLimitBar Settings", "RateLimitBarの設定"),
		GeneralTitle:  language.Text("General", "一般"),
		SourcesTitle:  language.Text("CLI sources", "CLIの取得元"),
		LanguageLabel: language.Text("Language", "言語"),
		ChangeSource:  language.Text("Choose…", "選択…"),
		Refresh:       popover.Command{Title: language.Text("Refresh", "更新"), Action: p.refresh},
		Quit:          popover.Command{Title: language.Text("Quit", "終了"), Action: p.quit},
	}
	for _, source := range p.sources {
		entry := popover.Provider{Name: source.Name, Message: language.Text("Loading…", "読み込み中…")}
		for _, result := range results {
			if result.Provider != source.Name {
				continue
			}
			entry.Message = ""
			switch {
			case result.Err != nil:
				entry.Message = formatLocalizedResultAt(result, now, language)
			case len(result.Usage.Windows) == 0:
				entry.Message = language.Text("No usage data", "使用率のデータがありません")
			default:
				if result.Usage.Stale {
					entry.Message = language.Text("Stale data", "古いデータ")
				}
				for _, window := range result.Usage.Windows {
					reset := language.Text("Reset time unavailable", "リセット時刻は不明です")
					if !window.ResetsAt.IsZero() {
						reset = language.Text("Resets in ", "リセットまで ") + formatRemaining(window.ResetsAt.Sub(now))
					}
					entry.Windows = append(entry.Windows, popover.Window{
						Label: windowLabel(language, window.Label), UsedPercent: window.UsedPercent, Reset: reset,
						Usage:     fmt.Sprintf(language.Text("Used %d%%", "使用済み %d%%"), window.UsedPercent),
						Remaining: fmt.Sprintf(language.Text("%d%% left", "残り %d%%"), max(0, 100-window.UsedPercent)),
						Selected:  p.sel.Enabled(source.Name, window.Label), Action: p.windows[windowKey{source.Name, window.Label}],
						Tooltip: language.Text("Show in menu bar", "メニューバーに表示"),
					})
				}
			}
			break
		}
		model.Providers = append(model.Providers, entry)
	}
	for _, command := range p.commands {
		model.Commands = append(model.Commands, command(language))
	}
	return model
}

func (p *usagePanel) dispatch(ctx context.Context, events <-chan string, redraw func()) {
	for {
		select {
		case <-ctx.Done():
			popover.Close()
			return
		case action := <-events:
			if action == "open" {
				redraw()
				continue
			}
			p.mu.Lock()
			clicked := p.actions[action]
			p.mu.Unlock()
			if clicked == nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case clicked <- struct{}{}:
			}
		}
	}
}
