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

メニューバーに `RateLimit` が表示される。メニューの `Refresh` で再取得、`Quit` で終了する。
