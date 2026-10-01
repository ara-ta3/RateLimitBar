package menubar

import (
	"sync"

	"ratelimitbar/internal/usage"
)

// TitleState は最新の取得結果を保持し、取得時と切り替え時の両方で同じ手順でタイトルを作り直す。
type TitleState struct {
	mu      sync.Mutex
	title   TitleSetter
	sel     *Selection
	results []usage.Result
}

func NewTitleState(title TitleSetter, sel *Selection) *TitleState {
	return &TitleState{title: title, sel: sel}
}

func (t *TitleState) update(results []usage.Result) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.results = results
	t.title.SetTitle(FormatTitle(t.results, t.sel))
}

// Redraw は保持している最新の結果から、現在の選択でタイトルを作り直す。
func (t *TitleState) Redraw() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.title.SetTitle(FormatTitle(t.results, t.sel))
}
