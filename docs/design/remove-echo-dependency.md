---
title: Web フレームワーク echo への依存を外し、Go 標準ライブラリだけで動かす
created: 2026-11-03
updated: 2026-11-03
status: 実装済み
---

# 要求

````text
このアプリケーションから、webフレームワーク echo への依存をなくして Go の標準実装だけにする設計をして
````

# 設計: echo を `net/http` / `log/slog` に置き換える

## 目的

`github.com/labstack/echo/v5`（と間接依存の `golang.org/x/time`）を `go.mod` から外し、外部モジュール依存ゼロにする。依存更新（Renovate PR・`go-mod-fix`）の手間と、echo のメジャー更新（v4→v5 で API が変わった前例がある）に追従するコストを無くす。

**クライアントから見た HTTP の振る舞いは、決定 #4 の 1 ヘッダを除いて一切変えない。** 本アプリはリクエストを観察するための道具なので、応答の形が変わること自体が利用者にとっての不具合になる。

## 現状（出発点）

### 読んだ既存設計文書

- `ARCHITECTURE.md` / `SPECS.md` は**どちらも無い**（リポジトリ直下・`docs/` 配下とも）。参照できる既存設計文書が無いので、補助として `CLAUDE.md` と `README.md` を読み、現状はコードと実測から起こした。
- `CLAUDE.md` の記述のうち今回を縛るもの:「The codebase is intentionally minimal with no test files」。決定 #6 でテストを追加するため、この一文は実装後に書き換える。
- `README.md` には Docker での起動方法とビルド方法しか無く、今回の設計には効かない。

### 現在の配線

| ファイル | echo に依存している箇所 |
| --- | --- |
| `server.go` `main()` | `echo.New()` / `middleware.RequestLoggerWithConfig(...)` / `middleware.Secure()` / `e.GET("/*", dumpRequest)` / `e.Start(":<port>")` |
| `echo.go` `dumpRequest(c *echo.Context) error` | `c.Request()` で `*http.Request` を取り出し、`c.JSON(http.StatusOK, r)` で返す |
| `version.go` | 依存なし |
| `go.mod` | `require github.com/labstack/echo/v5 v5.4.0`、間接で `golang.org/x/time v0.15.0` |

`dumpRequest` が触っているのは実質 `*http.Request`（`RequestURI` / `RemoteAddr` / `Header`）だけで、echo 固有の機能はルーティング・ミドルウェア・JSON 書き出し・サーバ起動の 4 点に限られる。

### echo が実際にやっていること（v5.4.0 のソースと、現行バイナリへの curl で実測）

| 項目 | 現状の振る舞い | 出どころ |
| --- | --- | --- |
| GET 任意パス | 200、`Content-Type: application/json`（charset なし）、本文は `json.Encoder.Encode` の出力（末尾 `\n` 付き） | `context.go` `json()` → `json.go` `DefaultJSONSerializer` |
| パスの正規化 | しない。`/../x` は 200 で `"path":"/../x"` を返す | echo ルータ |
| OPTIONS | 204、`Allow: OPTIONS, GET`、本文なし | echo ルータ |
| HEAD | 405、`Allow: OPTIONS, GET`、`Content-Type` も本文も無し | echo ルータ |
| POST / PUT / CONNECT 等 | 405、`Allow: OPTIONS, GET`、`Content-Type: application/json`、本文 `{"message":"Method Not Allowed"}\n` | echo ルータ + 既定エラーハンドラ |
| セキュリティヘッダ（全応答） | `X-Xss-Protection: 1; mode=block` / `X-Content-Type-Options: nosniff` / `X-Frame-Options: SAMEORIGIN`（HSTS 等は既定値 0/空で出ない） | `middleware/secure.go` |
| アクセスログ | `slog.Info("request", "remoteIP", …, "method", …, "status", …, "uri", …, "latency", …)` を既定 slog（stderr、テキスト形式）へ。`remoteIP` は `RemoteAddr` のホスト部（`IPExtractor` 未設定なので `X-Forwarded-For` は見ない）、`uri` は `RequestURI` | `server.go` の `LogValuesFunc` + `context.go` `RealIP()` |
| 起動ログ | echo の slog（JSON 形式、stdout）に banner 行と `"http(s) server started","address":"[::]:8080"` の 2 行 | `server.go` `StartConfig.start()` |
| `http.Server` 設定 | `ReadTimeout: 30s`、`ErrorLog` を slog へ | 同上 |
| 終了 | SIGINT / SIGTERM で `Shutdown`（猶予 10 秒）。`http.ErrServerClosed` はエラー扱いしない | `echo.go` `Start()` / `server.go` `gracefulShutdown()` |
| 起動失敗（ポート使用中など） | `e.Start` がエラーを返し `log.Fatal` | `server.go` |

### 既にあって使えるもの

- `dumpRequest` の中身（`DumpResult` の組み立て）は `*http.Request` しか使っていないので、そのまま `http.HandlerFunc` に移せる。
- アクセスログは既に `log/slog` の既定ロガーに出しているので、ログ出力側は書き換え不要。値の集め方だけ自前にする。
- フラグ解析・`PORT` 環境変数・`showVersion()` は既に標準ライブラリだけで書かれている。
- CI（`.github/workflows/ci.yaml`）は既に `go test -v ./...` を実行しているので、テストを足せばそのまま CI で回る。

## 決定

| # | 論点 | 決定 | 理由 |
| --- | --- | --- | --- |
| 1 | 置き換え先 | `net/http`・`log/slog`・`encoding/json`・`os/signal` など標準ライブラリだけを使う。他のルータ（chi 等）にも移らない | 要求そのもの。ルート 1 本・ミドルウェア 2 つの規模なので標準だけで足りる |
| 2 | GET 以外のメソッドの応答 | 現状と完全に同じにする。OPTIONS → 204 + `Allow: OPTIONS, GET`。HEAD → 405 + `Allow` のみ（`Content-Type`・本文なし）。その他 → 405 + `Allow` + `Content-Type: application/json` + `{"message":"Method Not Allowed"}\n` | ユーザー確認済み。観察用ツールなので応答が変わると利用者の検証結果が変わる |
| 3 | ルーティングの仕組み | `http.ServeMux` を使わず、1 つの `http.Handler` が全リクエストを受けてメソッドで分岐する | ServeMux は非正規パス（`/../x`、`//a`）を 301 でリダイレクトし、`GET /` 登録では HEAD を自動で受け付け、405 の本文もテキストになる。どれも現状と食い違い、決定 #2 と「パスを正規化しない」振る舞いを壊す。ルートが 1 本なのでマルチプレクサが要らない |
| 4 | セキュリティヘッダ | `X-Content-Type-Options: nosniff` と `X-Frame-Options: SAMEORIGIN` を全応答（200/204/405）に付ける。`X-XSS-Protection` は**付けない** | ユーザー確認済み。X-XSS-Protection は現行ブラウザで廃止済みで、残すと逆に害があるとされるヘッダ。外から見える唯一の差分になる |
| 5 | アクセスログ | 現状と同じキー・同じ値の意味で `slog.Info("request", ...)` を出す。ステータスは `ResponseWriter` を包んで捕まえ、`WriteHeader` が呼ばれなければ 200。`remoteIP` は `net.SplitHostPort(RemoteAddr)` のホスト部、失敗したら `RemoteAddr` そのまま | ログを見ている運用を壊さない。echo の `extractIP` と同じ規則 |
| 6 | テスト | `net/http/httptest` で、ハンドラ全体（ミドルウェア込み）の応答を固定するテストを追加する | ユーザー確認済み。「変わらないこと」を機械的に確かめる手段が他に無い |
| 7 | テストを先に書く順序 | まず echo のままで `http.Handler` を返す関数を切り出してテストを書き、echo 版で緑にしてから置き換える | `*echo.Echo` は `http.Handler` を実装しているので、同じテストを置き換え前後の両方に当てられる。置き換え前に緑であることが「テストが現状を正しく写している」ことの証拠になる |
| 8 | 起動ログ | echo の banner 行は出さない。代わりに既定 slog で `slog.Info("http server started", "address", <listener の実アドレス>)` を 1 行出す。出力先は既定 slog（stderr・テキスト形式）に揃う | banner は echo の宣伝文で意味が無い。起動確認とアドレス表示の役目だけ残す。形式と出力先（stdout→stderr）が変わるが、ログは HTTP の契約ではなく、アクセスログと揃う方が扱いやすい。ユーザー未確認（必要なら JSON/stdout に戻せる） |
| 9 | サーバ設定と終了処理 | `http.Server{ReadTimeout: 30 * time.Second}` を維持。`signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` で受け、`Shutdown` を 10 秒の猶予で呼ぶ。`http.ErrServerClosed` は正常終了 | 現状の echo `Start()` と同じ。コンテナ（distroless）で SIGTERM 時に処理中リクエストを捨てない挙動を保つ |
| 10 | リッスン | `net.Listen("tcp", ":<port>")` してから `Serve` する。Listen 失敗は `log.Fatal` | 起動ログに実アドレスを出せる（決定 #8）。失敗時の振る舞いは現状と同じ |
| 11 | JSON の書き出し | `Content-Type: application/json` を設定し、`json.NewEncoder(w).Encode(r)` で書く。`DumpResult` は文字列と map だけなのでエンコードは失敗しないが、書き込みエラー（接続切断など）は `slog.Error` で記録する | echo の `DefaultJSONSerializer` と同一の出力（HTML エスケープ有り・末尾改行・charset なし）になる |
| 12 | ファイル構成 | `echo.go` はハンドラ、`server.go` は起動とミドルウェアのまま。ミドルウェアは `server.go` に置き、新ファイルは `server_test.go` だけ | 現状の 2 ファイル分担を崩さない。ミドルウェアは各十数行で、ファイルを分けるほどではない |
| 13 | `echo.go` のファイル名 | 変えない | 「echo API」（エコーサーバ）の意味でアプリ名と一致しており、フレームワーク名ではない |

## 変更点

### 1. `server.go` ─ echo を外し、ハンドラ組み立てと起動を標準ライブラリで書く

公開する形（P1 で切り出し、P2 で中身を差し替える）:

```go
// newHandler はミドルウェア込みのアプリ全体を返す。テストからも使う。
func newHandler() http.Handler
```

P2 後の組み立て（擬似コード）:

```go
func newHandler() http.Handler {
    return requestLogger(secureHeaders(http.HandlerFunc(route)))
}

// route: 決定 #2, #3
func route(w, r) {
    switch r.Method {
    case GET:     dumpRequest(w, r)
    case OPTIONS: w.Header().Set("Allow", "OPTIONS, GET"); w.WriteHeader(204)
    case HEAD:    Allow を付けて 405（本文・Content-Type なし）
    default:      Allow を付け、JSON {"message":"Method Not Allowed"} で 405
    }
}

// secureHeaders: 決定 #4。next を呼ぶ前に 2 ヘッダを Set
// requestLogger: 決定 #5。statusRecorder{ResponseWriter, status} で包み、
//                終了後に slog.Info("request", "remoteIP", ..., "method", ..., "status", ..., "uri", r.RequestURI, "latency", time.Since(start))
```

`main()` は決定 #8〜#10 のとおり: `net.Listen` → 起動ログ → `srv.Serve` を goroutine で回し、`signal.NotifyContext` の `Done` で `srv.Shutdown(10 秒)`。フラグ・`getPortNumber()`・`-version` は触らない。

### 2. `echo.go` ─ ハンドラのシグネチャを標準形にする

```go
func dumpRequest(w http.ResponseWriter, req *http.Request)
```

`c.Request()` を `req` に置き換え、末尾を決定 #11 の JSON 書き出しにするだけ。`DumpResult` の組み立て（ヘッダは同名複数なら最後の値、`X_APP_NAME` はリクエスト毎に読む）は一字も変えない。

### 3. `server_test.go`（新規）─ 応答を固定するテスト

`httptest.NewRecorder()` に `newHandler().ServeHTTP` を当てる表駆動テスト。確かめる項目は「検証」節の表。`time` / `hostname` / `remoteAddr` は値が揺れるので、JSON を `DumpResult` にデコードして形と固定値だけ比べる。

### 4. `go.mod` / `go.sum` ─ 依存を外す

`go mod tidy` で `require` を全て削除。`go.sum` は空になるので削除する（`go mod tidy` の結果に従う）。`go` / `toolchain` ディレクティブは触らない。

## 触らないもの

| 触らないもの | 理由 |
| --- | --- |
| `DumpResult` の JSON 形とフィールドの値の決め方 | 応答の契約そのもの |
| 同名ヘッダが複数あるとき最後の値しか返さない挙動 | 既存の振る舞い。直すなら別要求 |
| `X_APP_NAME` が応答ヘッダではなく JSON の `headers` に入る挙動 | 既存の振る舞い（`CLAUDE.md` の「add custom header to responses」とは食い違うが、コードが正）。今回は変えない |
| GET 以外も受け付ける・リクエストボディをダンプする等の機能追加 | 決定 #2。依存除去とは別の要求 |
| `version.go` と `.goreleaser.yml` の ldflags（`-X github.com/little-forest/echo-api/main.version=...`） | package main には `main.version` で指す必要があり、現状 `dist/` のバイナリでも `-version` が `dev` を返す（実測）。不具合だが今回の主題外。別件として扱う |
| `Dockerfile` / CI / Renovate / `go-mod-fix` ワークフロー | 依存が無くなっても設定はそのまま動く。Renovate の対象が減るだけ |
| `HttpHeader` / `NewHttpHeader`（未使用） | echo 依存と無関係。消すなら別のコミットで |

## フェーズ

| | 内容 | これだけで何が変わるか |
| --- | --- | --- |
| **P1** | echo のまま `newHandler() http.Handler`（`*echo.Echo` を返す）を切り出し、`main()` はそれを使う形にする。`server_test.go` を追加し、**現状の振る舞い（`X-Xss-Protection` が付くことを含む）** をテストに写して緑にする | 利用者から見た振る舞いは何も変わらない。`go test` が現状の HTTP 契約を固定するようになる |
| **P2** | `echo.go` / `server.go` を標準ライブラリに置き換え、`go mod tidy`。テストの `X-Xss-Protection` の期待だけを「無いこと」に書き換える（決定 #4）。それ以外のテストは一切変えずに緑であること | echo 依存が消え、`go.mod` の require がゼロになる。応答から `X-Xss-Protection` が消える。起動ログが 1 行のテキスト形式になる |
| **（やらない）** | ldflags の修正、GET 以外のダンプ、未使用型の削除 | 「触らないもの」のとおり、依存除去と独立した変更なので混ぜない |

## 検証

### 自動（`go test ./...`）

| 対象 | 確認すること |
| --- | --- |
| `GET /foo/bar?x=1` | 200、`Content-Type` が `application/json` と完全一致、本文が `\n` で終わる、`path` が `/foo/bar?x=1`、`headers` にリクエストヘッダが入る |
| 同名ヘッダ 2 つ（`X-Multi: a`, `X-Multi: b`） | `headers["X-Multi"] == "b"` |
| `X_APP_NAME=foo`（`t.Setenv`） | `headers["X-Application-Name"] == "foo"`、未設定なら キーが無い |
| 非正規パス（`RequestURI` が `/../x`） | 200 で `path` が `/../x` のまま（リダイレクトしない）── 決定 #3 |
| OPTIONS | 204、`Allow: OPTIONS, GET`、本文空 |
| HEAD | 405、`Allow: OPTIONS, GET`、`Content-Type` 無し |
| POST / PUT / DELETE / CONNECT | 405、`Allow: OPTIONS, GET`、`Content-Type: application/json`、本文 `{"message":"Method Not Allowed"}\n` |
| 全ケース共通 | `X-Content-Type-Options: nosniff`、`X-Frame-Options: SAMEORIGIN` がある。`X-Xss-Protection` は P1 では有り / P2 では無し |
| `go.mod`（P2） | `go list -m all` が本モジュール 1 行だけを返す |

P1 の時点で全テストが echo 版で緑であること、P2 で `X-Xss-Protection` の 1 行以外を書き換えずに緑であること ── これが「変わっていない」ことの主たる証拠。

### 手動

1. `go build -o echo-api . && ./echo-api -p 18080` で起動し、stderr に `http server started address=[::]:18080` が 1 行出ること。echo の banner が出ないこと。
2. `curl -si 'localhost:18080/foo?x=1'` → 200、ヘッダは `Content-Type: application/json` / `X-Content-Type-Options` / `X-Frame-Options` / `Date` / `Content-Length` で、`X-Xss-Protection` が無いこと。
3. `curl -si -X POST localhost:18080/a`、`curl -sI localhost:18080/a`、`curl -si -X OPTIONS localhost:18080/a` → それぞれ現状の表どおり。
4. `curl -si --path-as-is 'localhost:18080/../x'` → 200 で `"path":"/../x"`（301 にならない）。
5. stderr のアクセスログが `INFO request remoteIP=::1 method=GET status=200 uri=/foo?x=1 latency=...` の形で、405/204 のときも `status` が正しいこと。
6. `PORT=18081 ./echo-api` で 18081 で待ち受けること。`./echo-api -version` が従来どおり表示されること。
7. 起動中に Ctrl+C（SIGINT）、および `kill -TERM <pid>` で、エラーを出さずに終了コード 0 で終わること。
8. 18080 を使用中の状態で起動し、`log.Fatal` で異常終了すること。
9. `task lint` と `goreleaser release --clean --snapshot` が通り、Docker イメージで `docker run --rm -p 8081:8081 <image> -p 8081` が同じ応答を返すこと。

## 見送った案

| 案 | 見送った理由 |
| --- | --- |
| `http.ServeMux` に `GET /` で登録する | 決定 #3。HEAD の自動受け付け・非正規パスの 301・テキストの 405 本文で現状と食い違う |
| ServeMux に `/` で登録し、ハンドラ内でメソッド分岐 | メソッドの問題は解けるが、非正規パスの 301 リダイレクトが残る。ルート 1 本ならマルチプレクサを挟む利点が無い |
| chi など軽量ルータへ移る | 要求は「Go の標準実装だけ」。外部依存が 1 つ残るので目的を満たさない |
| `X-XSS-Protection` も残して完全互換にする | ユーザーの判断で外す（決定 #4） |
| テストを書かず curl の手動確認だけにする | ユーザーの判断でテストを足す（決定 #6）。手動だけでは HEAD の `Content-Type` 有無のような細部を見落とす |
| 置き換えとテスト追加を 1 フェーズでやる | テストが echo 版で緑だった証拠が残らず、テスト自体が現状を写し損ねていても気づけない（決定 #7） |
| 起動ログを echo と同じ JSON・stdout で再現する | 出せるが、banner を失う以上は完全再現にならず、アクセスログ（stderr・テキスト）と出力先が分かれたままになる。決定 #8 |

## 実装後に更新する文書

`ARCHITECTURE.md` / `SPECS.md` は存在しないので更新対象なし（新設もしない）。代わりに次を更新する。

| 文書 | 追記する内容 |
| --- | --- |
| `CLAUDE.md`「Core Components」 | `server.go` の説明を「Logger and Secure middlewares」から、標準 `net/http` のハンドラ組み立て（メソッド分岐・セキュリティヘッダ・アクセスログ）と graceful shutdown に書き換える。`echo.go` の説明から echo の型への言及を消す |
| `CLAUDE.md`「Tools and Dependencies」 | 「Echo v4: Web framework」の行を削除し、外部モジュール依存が無いことを書く（既に v5 なのに v4 と書かれている誤りもここで消える） |
| `CLAUDE.md` 末尾 | 「no test files」の一文を、`server_test.go` が HTTP 応答の契約（405/204・ヘッダ・JSON 形）を固定していることに書き換える |
| `CLAUDE.md`「Configuration」 | `X_APP_NAME` の説明を実際の挙動（JSON の `headers` に `X-Application-Name` として入る）に合わせる ── 今回コードは変えないが、実装中に誤記を確認したので併せて直す |
| `README.md` | 変更不要（振る舞いの記述が無い） |
