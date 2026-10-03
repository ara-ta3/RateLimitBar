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

## Claudeのキャッシュ自動更新

`-claude-auto-refresh` を付けて起動すると、Claudeのキャッシュが15分より古い場合に `claude -p '/usage'` を実行し、完了後にキャッシュを読み直す。自動更新は初期状態ではオフで、通常は1分ごとにローカルファイルを読み直す。

```sh
make build
./bin/ratelimitbar -claude-auto-refresh
```

macOSアプリでは、終了してから次のように起動する。

```sh
open dist/RateLimitBar.app --args -claude-auto-refresh
```

インストール・ログイン済みの `claude` CLIが必要。Claude Codeによる更新時は通信が発生する。コマンドはプロジェクト外の一時ディレクトリで実行し、30秒でタイムアウトする。実行に失敗した場合は取得エラーを表示し、次の更新時に再試行する。キャッシュがない場合や読み取れない場合は、自動更新の対象にしない。
