# RateLimitBar

macOS のメニューバーに常駐し、各 Provider の RateLimit 使用率を表示するアプリ。現在は Mock データを表示する。

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
