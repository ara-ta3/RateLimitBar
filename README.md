# RateLimitBar

macOS のメニューバーに常駐し、各 Provider の RateLimit 使用率を表示するアプリ。Claude は `~/.claude.json` の `cachedUsageUtilization` から実データを表示し、60 秒ごとに自動更新する。キャッシュが 15 分より古い場合は `(stale)` を付ける。Codex と Cursor は Mock データを表示する。

## ビルド

```sh
make build
```

## テスト

```sh
make test
```

## 起動

```sh
make run
```

メニューバーのタイトルには、オンにした Provider/Window の使用率が `Claude 5h 23% / W 48%  Codex 5h 61%` の形で並ぶ（取得前とすべてオフのときは `RateLimit`）。メニューのチェック付き項目で Window ごとにオン/オフを切り替えられる（起動時は全てオンで、設定は保持しない）。メニューの `Refresh` で再取得、`Quit` で終了する。ターミナルから起動した場合は `Ctrl+C` でも終了できる。
