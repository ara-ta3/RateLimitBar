# RateLimitBar

macOS のメニューバーに常駐し、各 Provider の RateLimit 使用率を表示するアプリ。Claude は `~/.claude.json` の `cachedUsageUtilization` から実データを表示し、60 秒ごとに自動更新する。キャッシュが 15 分より古い場合は `(stale)` を付ける。Codex は `codex app-server` から、Cursor は Keychain の access token と usage API から実データを取得して表示する。Codex のモデル固有の limit は、取得できたときにメニューの切り替え項目へ追加する。

## ビルド

```sh
make build
```

## macOS アプリとして出力

macOS 上で次を実行すると、ビルドした Mac の CPU 向けに `dist/RateLimitBar.app` を生成する。

```sh
make app
open dist/RateLimitBar.app
```

Finder でダブルクリックして起動でき、`/Applications` へコピーして使うこともできる。Dock には表示せず、メニューバーの `Quit` で終了する。ローカル利用向けのアプリで、配布用の Developer ID 署名・公証は行わない。

Codex の使用率取得には、インストール・ログイン済みの `codex` CLI が必要。アプリ起動時に `$SHELL`（未設定なら `/bin/zsh`）のログイン環境を読み込むため、CLI の `PATH` は `~/.zprofile` などログイン時に読み込まれる設定へ追加する。対話シェル専用の `~/.zshrc` だけに設定した `PATH` は読み込まれない。

## テスト

```sh
make test
```

## 起動

```sh
make run
```

メニューバーのタイトルには、オンにした Provider/Window の使用率が `Claude 5h 23% / W 48%  Codex 5h 61%` の形で並ぶ（取得前とすべてオフのときは `RateLimit`）。メニューのチェック付き項目で Window ごとにオン/オフを切り替えられる（起動時は全てオンで、設定は保持しない）。メニューの `Refresh` で再取得、`Quit` で終了する。ターミナルから起動した場合は `Ctrl+C` でも終了できる。
