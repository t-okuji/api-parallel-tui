# API Concurrent Runner

Go + Bubble Tea v2 で作成した、API 同時実行用の TUI ツールです。

TUI 上でリクエストを編集し、そのまま複数回・複数並列で実行できます。

## Features

- TUI 上で API リクエストを編集
- 複数の Request 定義を保持
- `Parallelism` で同時実行数を制御
- `Repeat` で各 Request の実行回数を指定
- 実行結果を `Request 1`, `Request 2` ごとに表示
- 選択中の結果の `Body Preview` を表示
- `Payload` の JSON 色付きプレビューを表示
- SQLite へのセッション保存 / 読み込み
- 画面が小さい場合でも縦スクロール可能

## Screen

入力項目:

- `Parallelism`
- `Repeat`
- `Name`
- `Method`
- `URL`
- `Headers`
- `Payload`

`Headers` は複数行の `Key: Value` 形式です。

```text
Content-Type: application/json
Authorization: Bearer your-token
X-Request-ID: test-001
```

`Payload` には JSON をそのまま入力できます。

## Parallelism And Repeat

`Parallelism` は同時に走る最大実行数です。

`Repeat` は各 Request を何回送るかです。

例:

- Request が 1 件
- `Parallelism = 3`
- `Repeat = 5`

この場合、同じ Request を 5 回送信し、最大 3 本まで同時実行します。

## Requirements

- Go 1.26 以上

## Run

```bash
go run ./cmd/api-parallel-tui
```

## Build

通常ビルド:

```bash
mkdir -p build
go build -o build/api-parallel-tui ./cmd/api-parallel-tui
```

Windows 向け `.exe`:

```bash
mkdir -p build
GOOS=windows GOARCH=amd64 go build -o build/api-parallel-tui-windows-amd64.exe ./cmd/api-parallel-tui
```

ARM Windows 向け:

```bash
mkdir -p build
GOOS=windows GOARCH=arm64 go build -o build/api-parallel-tui-windows-arm64.exe ./cmd/api-parallel-tui
```

## Lint

実行:

```bash
golangci-lint run
```

フォーマット:

```bash
golangci-lint fmt
```

## Keybindings

基本操作:

- `q` / `ctrl+c`: 終了
- `ctrl+r`: 実行
- `ctrl+n`: Request 追加
- `ctrl+d`: Request 削除
- `ctrl+s`: セッション名を入力して保存
- `ctrl+o`: 保存済みセッションを読み込み

入力欄移動:

- `up`: 同じ Request 内で前の入力欄へ移動
- `down`: 同じ Request 内で次の入力欄へ移動
- `tab`: 次の Request の同じ欄へ移動
- `shift+tab`: 前の Request の同じ欄へ移動

結果選択:

- `ctrl+j`: 次の結果を選択
- `ctrl+k`: 前の結果を選択

スクロール:

- マウスホイール
- `pgup`
- `pgdn`
- `ctrl+b`
- `ctrl+f`

## Result View

結果は `Request 1`, `Request 2` ごとにグループ化して表示されます。

`Repeat > 1` の場合、各実行は `#1/5`, `#2/5` のように表示されます。

選択中の結果には `>` が付き、そのレスポンス本文が `Body Preview` に表示されます。

## Save Format

保存先はカレントディレクトリの `sessions.db` です。

保存内容:

- `concurrency`
- `repeat`
- `requests`
- `session name`

## Notes

- HTTP レスポンス本文は最大 4096 bytes まで読み込みます
- `Content-Type` 未指定で Body がある場合は `application/json` を自動設定します
- `Body Preview` は JSON の場合、自動で整形表示します

## Example

```text
Parallelism: 10
Repeat:      5
Method:      POST
URL:         https://example.com/api
Headers:
Content-Type: application/json
Authorization: Bearer your-token

Payload:
{"name":"Taro","role":"admin"}
```
