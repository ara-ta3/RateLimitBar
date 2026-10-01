package menubar

import "ratelimitbar/internal/usage"

// CheckboxAdder は、provider と label の切り替え項目をメニューへ追加する。
type CheckboxAdder interface {
	AddCheckbox(provider, label string)
}

// DynamicItems は、取得して初めて分かる Window のラベルを切り替え項目として追加する。
// 起動時に宣言されたラベルは対象外で、一度追加した項目は取得結果から消えても残す。
type DynamicItems struct {
	sel   *Selection
	adder CheckboxAdder
}

func NewDynamicItems(sel *Selection, adder CheckboxAdder) *DynamicItems {
	return &DynamicItems{sel: sel, adder: adder}
}

// Apply は、成功した結果の Window のうち、まだ項目のない (provider, label) を追加する。
func (d *DynamicItems) Apply(results []usage.Result) {
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		for _, w := range r.Usage.Windows {
			if d.sel.declare(r.Provider, w.Label) {
				d.adder.AddCheckbox(r.Provider, w.Label)
			}
		}
	}
}
