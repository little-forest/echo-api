# echo-api

[English](README.md) | 日本語

[![CI](https://github.com/little-forest/echo-api/actions/workflows/ci.yaml/badge.svg)](https://github.com/little-forest/echo-api/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/v/release/little-forest/echo-api)](https://github.com/little-forest/echo-api/releases)
[![Docker Image](https://img.shields.io/docker/v/littlef/echo-api?label=docker&sort=semver)](https://hub.docker.com/r/littlef/echo-api)
[![Go Version](https://img.shields.io/github/go-mod/go-version/little-forest/echo-api)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

受け取ったリクエストの内容（パス、ヘッダ、接続元アドレスなど）を JSON で返す、小さな HTTP サーバです。
プロキシ、ロードバランサ、Ingress コントローラ、HTTP クライアントを経由したリクエストが、バックエンドにどう届いているかを確認できます。

```console
$ curl -s 'localhost:8080/hello?name=world' | jq
{
  "path": "/hello?name=world",
  "time": "2026-11-03T11:58:22.994029+09:00",
  "remoteAddr": "[::1]:60811",
  "hostname": "my-host",
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl/8.7.1"
  }
}
```

## 特長

- **任意のパスに応答**: パスに関係なく、すべての `GET` リクエストに対してリクエスト内容を JSON で返します。
- **リクエストをそのまま確認できる**: パスの正規化やリダイレクトをしないため、`/foo//bar` や `/a/../b` も送られたとおりに表示されます。
- **外部依存なし**: Go の標準ライブラリ（`net/http`、`log/slog`、`encoding/json`）だけで作られています。
- **小さなコンテナイメージ**: distroless ベースのマルチアーキテクチャイメージ（`linux/amd64`、`linux/arm64`）を提供しています。
- **アクセスログ**: `log/slog` で、リクエストごとに 1 行の構造化ログを出力します。
- **グレースフルシャットダウン**: `SIGINT` / `SIGTERM` を受けると処理中のリクエストを終えてから停止します。

## クイックスタート

```console
$ docker run --rm -p 8080:8080 littlef/echo-api:1.0.0
$ curl localhost:8080/
```

## インストール

### Docker

`linux/amd64` と `linux/arm64` 向けのイメージを [Docker Hub](https://hub.docker.com/r/littlef/echo-api) で公開しています。
タグはリリースバージョンから先頭の `v` を除いたものです（例: `1.0.0`）。

```console
$ docker pull littlef/echo-api:1.0.0
```

### ビルド済みバイナリ

[Releases](https://github.com/little-forest/echo-api/releases) ページから、Linux / macOS / Windows（amd64 / arm64）向けのアーカイブをダウンロードできます。

### go install

```console
$ go install github.com/little-forest/echo-api@latest
```

## 使い方

```console
$ echo-api [-p PORT] [-v]
```

| オプション | 説明 | デフォルト |
| --- | --- | --- |
| `-p`, `--port` | 待ち受けるポート番号 | `8080` |
| `-v`, `--version` | バージョン情報を表示して終了 | |

| 環境変数 | 説明 |
| --- | --- |
| `PORT` | 待ち受けるポート番号。**設定されている場合は `-p` / `--port` より優先されます。** |
| `X_APP_NAME` | 設定すると、JSON レスポンスの `headers` フィールドに、この値を持つ `X-Application-Name` が追加されます。HTTP レスポンスヘッダとしては送られません。複数のデプロイを見分けるときに便利です。 |

例:

```console
# ポート 8081 で待ち受ける
$ echo-api -p 8081

# Docker で同じことをする
$ docker run --rm -p 8081:8081 littlef/echo-api:1.0.0 -p 8081

# レスポンスでインスタンスを識別する
$ docker run --rm -p 8080:8080 -e X_APP_NAME=frontend-a littlef/echo-api:1.0.0
```

## レスポンス形式

`GET` リクエストには `200 OK`、`Content-Type: application/json` で次の JSON を返します。

| フィールド | 説明 |
| --- | --- |
| `path` | 受け取ったリクエスト URI（クエリ文字列を含む） |
| `time` | リクエストを処理した時刻（ナノ秒付きの RFC 3339 形式） |
| `remoteAddr` | 接続元クライアントのアドレス（`host:port`）。プロキシ経由の場合はプロキシのアドレスになります。 |
| `hostname` | echo-api が動いているマシン（またはコンテナ）のホスト名 |
| `headers` | リクエストヘッダ（ヘッダ名と値のオブジェクト） |

`headers` についての注意:

- ヘッダ名は Go の正規形式に変換されます（例: `x-foo` は `X-Foo` になります）。
- 同じヘッダが複数回送られた場合は、最後の値だけが表示されます。
- `Host` ヘッダは含まれません。Go の `net/http` がヘッダマップから取り除くためです。

## HTTP の挙動

| メソッド | ステータス | レスポンス |
| --- | --- | --- |
| `GET` | `200 OK` | リクエスト内容の JSON（上記参照） |
| `OPTIONS` | `204 No Content` | `Allow: OPTIONS, GET`、ボディなし |
| `HEAD` | `405 Method Not Allowed` | `Allow: OPTIONS, GET`、ボディなし |
| その他（`POST`、`PUT` など） | `405 Method Not Allowed` | `Allow: OPTIONS, GET`、ボディ `{"message":"Method Not Allowed"}` |

すべてのレスポンスに `X-Content-Type-Options: nosniff` と `X-Frame-Options: SAMEORIGIN` が付きます。

## ログ

ログは `log/slog` で標準エラー出力に書き出されます。リクエストごとに 1 行出力されます。

```text
2026/11/03 11:58:22 INFO http server started address=[::]:8080
2026/11/03 11:58:22 INFO request remoteIP=::1 method=GET status=200 uri="/foo//bar?x=1" latency=28.459µs
```

## 利用例

- リバースプロキシ、ロードバランサ、Ingress が追加・書き換えるヘッダ（`X-Forwarded-For`、`X-Forwarded-Proto` など）を確認する。
- ルーティングやリライトルールを通ったあと、バックエンドに届くパスとクエリ文字列を確認する。
- HTTP クライアントが実際に何を送っているかを確認してデバッグする。
- Kubernetes などで疎通確認用の仮のバックエンドとして使う。`X_APP_NAME` と `hostname` で、どのインスタンスが応答したかがわかります。

## 開発

必要なもの:

- [Go](https://go.dev/)（バージョンは [`go.mod`](go.mod) を参照）
- [Task](https://taskfile.dev/)
- [aqua](https://aquaproj.github.io/)（`golangci-lint` などのツールをインストールします）

```console
$ task setup    # aqua でツールをインストール
$ task run      # ローカルでポート 8080 で起動
$ task test     # テストを実行
$ task lint     # golangci-lint を実行
$ task format   # コードを整形
$ task build    # GoReleaser でリリース成果物をローカルにビルド（スナップショット）
```

## リリース

`v1.2.3` のような形式のタグを push すると GitHub Actions が動き、[GoReleaser](https://goreleaser.com/) がバイナリを GitHub Releases に、Docker イメージを Docker Hub に公開します。

## コントリビュート

Issue と Pull Request を歓迎します。Pull Request を出す前に、`task lint` と `task test` が通ることを確認してください。

echo-api はリクエストを観察するためのツールなので、レスポンスそのものが公開インターフェースです。
HTTP の仕様（JSON の形、ステータスコード、ヘッダ）は [`server_test.go`](server_test.go) のテストで固定しており、これを変える変更は破壊的変更として扱います。

## ライセンス

[MIT](LICENSE) © KOMORI Yusuke
