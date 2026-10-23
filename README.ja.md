<p align="center">
  <img src="packaging/macos/AppIcon.png" width="128" alt="RateLimitBar アイコン">
</p>

<h1 align="center">RateLimitBar</h1>

<p align="center"><a href="README.md">English</a> | 日本語</p>

<p align="center">Claude・Codex・Cursor の使用率を、macOS のメニューバーで。</p>

<p align="center">
  <a href="https://github.com/ara-ta3/RateLimitBar/actions/workflows/ci.yml"><img src="https://github.com/ara-ta3/RateLimitBar/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <img src="https://img.shields.io/badge/platform-macOS-black" alt="macOS">
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go 1.27">
</p>

RateLimitBar は、AI コーディングツールの使用率とリセットまでの残り時間を表示するメニューバーアプリです。60 秒ごとに自動更新し、表示するサービスや利用枠を選べます。

```text
CL ━━╸──  CX ━━━──  CU ╸────
```

*メニューバーの模式図。`CL` はClaude、`CX` はCodex、`CU` はCursorです。各バーは、チェックした利用枠のうち使用済み割合が最も高い枠を表します。正確な数値とリセット時間はクリック後のパネルで確認できます。*

## 特徴

- **3 サービスをまとめて確認** — Claude・Codex・Cursor の使用率を表示。
- **リセットまでの時間を表示** — 取得できたリセット時刻から残り時間を表示。
- **表示項目を選択** — サービスの利用枠ごとにオン・オフを切り替え。
- **自動・手動更新** — 60 秒ごとの更新に加え、メニューの「更新」で再取得。
- **メニューバーに常駐** — Dock に表示せず、Finder から起動可能。

## 対応サービス

| サービス | 表示する利用枠 | 取得元・必要な環境 |
| --- | --- | --- |
| Claude | 5 時間・週間 | `~/.claude.json` の `cachedUsageUtilization`。Claude Code の使用率キャッシュが必要 |
| Codex | 5 時間・週間・取得できたモデル固有の枠 | `codex app-server`。インストール・ログイン済みの `codex` CLI が必要 |
| Cursor | 月間 | macOS Keychain のアクセストークンと usage API。Cursor の認証情報が Keychain に保存されている必要あり |

Claude は通常、ローカルのキャッシュを読み取ります。15 分より古いキャッシュには、ドロップダウンで `(stale)` を付けます。必要に応じて[キャッシュの自動更新](#claude-のキャッシュ自動更新)を有効にできます。

## セットアップ

### リリース版をインストール

[Releases](https://github.com/ara-ta3/RateLimitBar/releases) から、MacのCPUに対応するZIPをダウンロードして展開し、`RateLimitBar.app` を `/Applications` にコピーしてください。

- Apple Silicon（Mシリーズ）: `RateLimitBar-v<version>-macOS-arm64.zip`
- Intel Mac: `RateLimitBar-v<version>-macOS-amd64.zip`

リリース版はmacOS 14以降を対象にビルドしています。Go・Xcodeは不要ですが、利用するサービスの認証済みCLIまたはローカル認証情報が必要です。

開発者向けの未署名・未公証の配布です。macOSに起動を止められた場合は、一度起動を試した後に「システム設定 → プライバシーとセキュリティ」から許可してください。

### 必要な環境

- macOS
- Go 1.27 以降
- Xcode Command Line Tools（`xcode-select --install` でインストール）
- 利用するサービスの認証済み CLI またはローカル認証情報（上の表を参照）

### ソースからビルドして起動

```sh
git clone https://github.com/ara-ta3/RateLimitBar.git
cd RateLimitBar
make app
open dist/RateLimitBar.app
```

`dist/RateLimitBar.app` は、ビルドした Mac の CPU 向けに生成されます。Finder でダブルクリックして起動でき、`/Applications` にコピーして使うこともできます。

> 生成されるアプリはローカル利用向けです。Developer ID 署名・公証は行いません。

## 使い方

メニューバーの表示をクリックすると、サービスごとの使用率バーとリセットまでの残り時間を表示するパネルが開きます。バーと色は全サービスで「使用済みの割合」を表し、70%未満が緑、70%以上が黄、90%以上が赤です。使用済みと残りの割合を併記します。利用枠のチェックでメニューバーの代表値に含めるかを切り替えられます。オフにしてもデータの取得とパネルの更新は続きます。パネル内の「設定…」から独立した設定ウインドウを開き、言語・キャッシュの自動更新・CLIの取得元を変更できます。設定ウインドウは外をクリックしても閉じず、もう一度「設定…」を選ぶと同じウインドウが前面に戻ります。右クリックでは従来のメニューが開きます。

| メニュー項目 | 操作 |
| --- | --- |
| 各サービスのチェック付き項目 | 利用枠ごとにメニューバーへの表示を切り替え |
| 更新 / Refresh | 使用率を再取得 |
| 取得元の設定 | Codex・Claude の CLI パスを確認・変更 |
| Claudeのキャッシュを自動更新 | 古いキャッシュを Claude CLI で更新 |
| 終了 / Quit | アプリを終了 |
| 言語 / Language | 日本語・英語を切り替え（選択を保存） |

表示項目は起動時にすべてオンになります。表示の選択と Claude のキャッシュ自動更新のオン・オフは、次回起動時に引き継ぎません。

残り時間は使用率の取得時に更新し、パネル内では開いたときにも更新します。1 分未満は `<1m`、リセット時刻を過ぎると `0m` を表示し、リセット時刻が取得できない枠は使用率だけを表示します。取得前やすべての表示項目がオフのときは、メニューバーにグラフのアイコンを表示します。取得失敗やデータなしはバーの代わりにダッシュを表示し、古いデータのバーは灰色にします。マウスを重ねると、選択した枠の数値と取得状態を確認できます。

### 表示言語を切り替える

「言語 → English」または「Language → 日本語」で、メニュー・取得状況・ファイル選択の案内をすぐに切り替えられます。初期設定は日本語です。選択は `settings.json` に保存され、次回起動時にも引き継ぎます。サービス名と `5h`、`1h30m` などの短い時間表記は共通です。

### CLI の取得元を設定

初期状態では、アプリの `PATH` から `codex` / `claude` を自動検出します。macOS アプリから起動するときは `$SHELL`（未設定なら `/bin/zsh`）のログイン環境を読み込みます。

CLI が見つからない場合や、特定の実行ファイルを使いたい場合は、次の手順で指定できます。

1. メニューの「取得元の設定」を開く。
2. 「未検出（指定…）」または表示された CLI パスをクリックする。
3. 実行ファイルを選択する。ファイル選択画面で `⌘⇧G` を押すと、パスを入力できます。

CLI のパスはターミナルで確認できます。

```sh
command -v codex
command -v claude
```

指定したパスは `~/Library/Application Support/RateLimitBar/settings.json` に保存され、次回起動時にも使われます。指定後は自動で再取得します。

指定した実行ファイルが削除された場合は「未検出」と表示します。再指定するか、「取得元の指定をすべて解除（自動検出に戻す）」で `PATH` 検索に戻せます。

Claude CLI はキャッシュの自動更新に使用します。通常のキャッシュ表示には必要ありません。Cursor は Keychain と API を使用するため、この設定の対象外です。

### Claude のキャッシュ自動更新

メニューの「Claudeのキャッシュを自動更新」をオンにすると、キャッシュが 15 分より古い場合に次のコマンドを実行し、完了後にキャッシュを読み直します。

```sh
claude -p '/usage'
```

- 起動時はオフ。アプリを起動したまま切り替えられます。
- インストール・ログイン済みの `claude` CLI が必要で、更新時に通信が発生します。
- コマンドはプロジェクト外の一時ディレクトリで実行し、30 秒でタイムアウトします。
- 失敗時は取得エラーを表示し、次の更新時に再試行します。
- キャッシュがない場合や読み取れない場合は、自動更新の対象にしません。

## トラブルシューティング

| 症状 | 確認すること |
| --- | --- |
| CLI が見つからない | 「取得元の設定」で実行ファイルを指定してください。`~/.zshrc` だけに設定した `PATH` は、Finder 起動時には読み込まれません |
| Claude に `(stale)` が付く | キャッシュが 15 分より古い状態です。Claude Code で更新するか、キャッシュの自動更新を有効にしてください |
| 「取得に失敗しました」が表示される | 対応サービスの認証状態と取得元を確認し、「更新」で再取得してください。Cursor は Keychain の認証情報と API への接続が必要です |

## 開発

```sh
make build      # bin/ratelimitbar を生成
make run        # ビルドしてターミナルから起動
make app        # dist/RateLimitBar.app を生成
make test       # テストを実行
make fmt/check  # Go の整形を適用し、差分がないことを確認
```

ターミナルから起動した場合は、`Ctrl+C` でも終了できます。CI は macOS 上で整形チェック・ビルド・テストを実行します。

### ディレクトリ構成

```text
cmd/app/           エントリーポイント・設定操作
internal/config/   CLI パス設定の保存
internal/dialog/   ファイル選択ダイアログ
internal/menubar/  メニューバーの表示・操作・更新
internal/provider/ 各サービスの使用率取得
internal/usage/    使用率データの型
packaging/macos/   macOS アプリのアイコン・起動スクリプト・Info.plist
```

### リリース

Release workflowを含む変更を `main` にマージし、リリース対象のコミットに `v<major>.<minor>.<patch>` 形式のタグを付けてpushします。初回は `v0.1.0` です。

```sh
git switch main
git pull --ff-only origin main
git tag -a v0.1.0 -m 'Release v0.1.0'
git push origin v0.1.0
```

GitHub ActionsがApple Silicon・Intelそれぞれで整形チェック、テスト、アプリのビルドを行い、両方が成功したらZIPと `SHA256SUMS.txt` を添付したGitHub Releaseを公開します。アプリのバージョンにはタグの値を設定します。署名用のSecretは不要です。

## フィードバック・コントリビューション

不具合報告や機能提案は [Issues](https://github.com/ara-ta3/RateLimitBar/issues) へ、変更案は Pull Request で受け付けています。不具合報告には、macOS のバージョン、対象サービス、再現手順を添えてください。認証トークンなどの機密情報は含めないでください。
