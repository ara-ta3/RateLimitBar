package menubar

import (
	"slices"
	"sync"
)

// Source は Provider 名と、その Provider が持つ Window のラベルの組。
type Source struct {
	Name    string
	Windows []string
}

type windowKey struct {
	provider string
	label    string
}

// Selection はタイトルに表示する Provider/Window の選択状態を持つ。初期状態は全てオンで、永続化しない。
type Selection struct {
	mu       sync.Mutex
	windows  map[string][]string
	disabled map[windowKey]bool
}

func NewSelection(sources []Source) *Selection {
	windows := make(map[string][]string, len(sources))
	for _, s := range sources {
		windows[s.Name] = slices.Clone(s.Windows)
	}
	return &Selection{windows: windows, disabled: map[windowKey]bool{}}
}

// Toggle は Window のオン/オフを反転し、反転後の状態(オンなら true)を返す。
func (s *Selection) Toggle(provider, label string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := windowKey{provider, label}
	s.disabled[key] = !s.disabled[key]
	return !s.disabled[key]
}

func (s *Selection) Enabled(provider, label string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.disabled[windowKey{provider, label}]
}

// anyEnabled は provider のいずれかの Window がオンかを返す。
func (s *Selection) anyEnabled(provider string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, label := range s.windows[provider] {
		if !s.disabled[windowKey{provider, label}] {
			return true
		}
	}
	return false
}

// declare は label を provider の Window として登録し、新規に登録したときだけ true を返す。
// 登録済みのラベルの選択状態は変えない。登録したラベルは anyEnabled の対象になる。
func (s *Selection) declare(provider, label string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(s.windows[provider], label) {
		return false
	}
	s.windows[provider] = append(s.windows[provider], label)
	return true
}
