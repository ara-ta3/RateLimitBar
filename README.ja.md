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
Claude 5h ━╸─── 週間 ━━━╸─  Codex 5h ╸────  Cursor 月間 ━━━──
```

*メニューバーの模式図。サービス名と利用枠の名前を表示し、チェックした枠それぞれの使用済み割合をバーで表します。5時間だけ・週間だけ・両方など、表示する枠を選べます。正確な数値とリセット時間はクリック後のパネルで確認できます。*

## 特徴

- **3 サービスをまとめて確認** — Claude・Codex・Cursor の使用率を表示。
- **リセットまでの時間を表示** — 取得できたリセット時刻から残り時間を表示。
- **表示項目を選択** — サービスの利用枠ごとにオン・オフを切り替え。
- **自動・手動更新** — 60 秒ごとの更新に加え、メニューの「更新」で再取得。
- **メニューバーに常駐** — Dock に表示せず、Finder から起動可能。

## 対応サービス

| サービス | 表示する利用枠 | 取得元・必要な環境 |
| --- | --- | --- |
| Claude | 5 時間・週間 | Claude Code の `statusLine` から保存した使用量。[連携設定](#claude-code-との連携)が必要 |
| Codex | 5 時間・週間・取得できたモデル固有の枠 | `codex app-server`。インストール・ログイン済みの `codex` CLI が必要 |
| Cursor | 月間 | macOS Keychain のアクセストークンと usage API。Cursor の認証情報が Keychain に保存されている必要あり |

Claude は statusLine が保存したローカルの使用量を読み取ります。15 分より古いデータは「古いデータ」と表示します。Claude Code を使っていない間は更新されません。

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

メニューバーの表示をクリックすると、サービスごとの使用率バーとリセットまでの残り時間を表示するパネルが開きます。バーと色は全サービスで「使用済みの割合」を表し、70%未満が緑、70%以上が黄、90%以上が赤です。使用済みと残りの割合を併記します。利用枠のチェックで、それぞれのバーをメニューバーに表示するかを切り替えられます。オフにしてもデータの取得とパネルの更新は続きます。パネル内の「設定…」から独立した設定ウインドウを開き、言語・Codex CLIの取得元を変更できます。設定ウインドウは外をクリックしても閉じず、もう一度「設定…」を選ぶと同じウインドウが前面に戻ります。右クリックでは従来のメニューが開きます。

| メニュー項目 | 操作 |
| --- | --- |
| 各サービスのチェック付き項目 | 利用枠ごとにメニューバーへの表示を切り替え |
| 更新 / Refresh | 使用率を再取得 |
| 取得元の設定 | Codex の CLI パスを確認・変更 |
| 終了 / Quit | アプリを終了 |
| 言語 / Language | 日本語・英語を切り替え（選択を保存） |

表示項目は起動時にすべてオンになります。表示の選択は、次回起動時に引き継ぎません。

残り時間は使用率の取得時に更新し、パネル内では開いたときにも更新します。1 分未満は `<1m`、リセット時刻を過ぎると `0m` を表示し、リセット時刻が取得できない枠は使用率だけを表示します。取得前やすべての表示項目がオフのときは、メニューバーにグラフのアイコンを表示します。取得失敗やデータなしはバーの代わりにダッシュを表示し、古いデータのバーは灰色にします。マウスを重ねると、選択した枠の数値と取得状態を確認できます。

### 表示言語を切り替える

「言語 → English」または「Language → 日本語」で、メニュー・取得状況・ファイル選択の案内をすぐに切り替えられます。初期設定は日本語です。選択は `settings.json` に保存され、次回起動時にも引き継ぎます。サービス名と `5h`、`1h30m` などの短い時間表記は共通です。

### CLI の取得元を設定

初期状態では、アプリの `PATH` から `codex` を自動検出します。macOS アプリから起動するときは `$SHELL`（未設定なら `/bin/zsh`）のログイン環境を読み込みます。

CLI が見つからない場合や、特定の実行ファイルを使いたい場合は、次の手順で指定できます。

1. メニューの「取得元の設定」を開く。
2. 「未検出（指定…）」または表示された CLI パスをクリックする。
3. 実行ファイルを選択する。ファイル選択画面で `⌘⇧G` を押すと、パスを入力できます。

CLI のパスはターミナルで確認できます。

```sh
command -v codex
```

指定したパスは `~/Library/Application Support/RateLimitBar/settings.json` に保存され、次回起動時にも使われます。指定後は自動で再取得します。

指定した実行ファイルが削除された場合は「未検出」と表示します。再指定するか、「取得元の指定をすべて解除（自動検出に戻す）」で `PATH` 検索に戻せます。

Claude は以下の statusLine 連携、Cursor は Keychain と API を使用するため、CLI パス設定の対象外です。

### Claude Code との連携

`RateLimitBar.app` を `/Applications` に配置し、`~/.claude/settings.json` の既存設定に次を追加してください。既に `statusLine` を設定している場合は、下の併用例を使って既存の表示を引き継げます。設定ファイル全体を置き換えないでください。

```json
{
  "statusLine": {
    "type": "command",
    "command": "\"/Applications/RateLimitBar.app/Contents/MacOS/ratelimitbar\" claude-statusline"
  }
}
```

ソースから利用する場合は、`make build` 後の `bin/ratelimitbar` の絶対パスを指定します。`launch` ではなく `ratelimitbar` を指定してください。

Claude Code を新しく起動して1回応答を受け取ると、statusLine に `Claude 5h 5% / Weekly 6%` のように表示され、RateLimitBar の次回更新にも反映されます。Pro・Max は公式ドキュメントの対象で、Team でも実機確認済みです。API キーでの利用など、`rate_limits` が渡されない環境では使用量を取得できません。[公式の statusLine 仕様](https://code.claude.com/docs/en/statusline#rate-limit-usage)も参照してください。

- 保存先は `~/Library/Caches/RateLimitBar/claude-usage.json` です。使用率・リセット時刻・受信時刻だけを保存し、ファイルの権限は `0600` にします。
- RateLimitBar は60秒ごとにファイルを読み取ります。「更新」でも読み直せますが、Claude Code への問い合わせは行いません。
- 15分より古い値は古いデータとして表示します。リセット時刻を過ぎた枠は、次の取得時に表示対象から外します。
- 初回応答前などで `rate_limits` がない場合は、保存済みの値と受信時刻を維持します。5時間・週間のどちらかだけが渡された場合は、その枠だけを保存します。
- 従来の `~/.claude.json` と `claude -p /usage` による自動更新は使用しません。旧 `claudePath` 設定は無視し、Codex の取得元と言語設定は引き継ぎます。

既存の statusLine を併用する場合は、次のスクリプトを作り、その絶対パスを `statusLine.command` に設定します。最後の行は現在使っている statusLine コマンドに置き換え、スクリプトには `chmod +x` で実行権限を付けてください。

```sh
#!/bin/sh
payload=$(cat)
printf '%s' "$payload" | "/Applications/RateLimitBar.app/Contents/MacOS/ratelimitbar" claude-statusline >/dev/null
printf '%s' "$payload" | /absolute/path/to/your-existing-statusline
```

## トラブルシューティング

| 症状 | 確認すること |
| --- | --- |
| CLI が見つからない | 「取得元の設定」で実行ファイルを指定してください。`~/.zshrc` だけに設定した `PATH` は、Finder 起動時には読み込まれません |
| Claude に「古いデータ」が付く | 最後の受信から15分を過ぎています。連携設定済みの Claude Code で応答を受け取ってください |
| Claude の取得に失敗する | statusLine の設定と実行ファイルのパスを確認してください。初回応答前・連携未設定・すべての枠が期限切れの場合は表示できません |
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
internal/claudeusage/ statusLine の使用量の解析・保存
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
