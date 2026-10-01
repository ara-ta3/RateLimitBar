package menubar

import (
	"context"
	"errors"
	"fmt"

	"ratelimitbar/internal/usage"
)

// Fetch は全 Provider の Result を返す。一部が失敗した場合も Result は返し、error に失敗を含める。
type Fetch func(ctx context.Context) ([]usage.Result, error)

type TitleSetter interface {
	SetTitle(title string)
}

// ErrResultCountMismatch は Result 数と項目数が合わず、表示を更新できないことを表す。
var ErrResultCountMismatch = errors.New("menubar: result count does not match menu item count")

// ApplyRefresh は fetch の結果を既存の項目へ反映する。
// 取得 error は表示の反映後にそのまま返す。Result 数と項目数が合わない場合は反映せず ErrResultCountMismatch を返す。
func ApplyRefresh(ctx context.Context, fetch Fetch, items []TitleSetter) error {
	results, fetchErr := fetch(ctx)
	if len(results) != len(items) {
		return fmt.Errorf("%w: %d results, %d items", ErrResultCountMismatch, len(results), len(items))
	}
	for i, r := range results {
		items[i].SetTitle(FormatResult(r))
	}
	return fetchErr
}
