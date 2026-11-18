# gohns

[English](README.md) | 日本語

hns のレンダラーの Go 実装です。CGI または FastCGI として動きます。

hns は ハイパー日記システム(Hyper Nikki System)の略で、HNF は Hyper Nikki
File(ハイパー日記ファイル)の略です。gohns は HNF 形式のうち、よく使う部分だけを
実装していて、すべてではありません。何を読めるか、元とどう違うかは
[docs/hnf.ja.md](docs/hnf.ja.md) を見てください。

gohns は、HNF 形式の日記ファイル(1 日 1 ファイル)を読み、
[mustache](https://github.com/cbroglie/mustache) のテンプレートで HTML のページに
します。どの日を表示するかは、URL のクエリ文字列(`?202403`、`?0101`、`?title`
など)で選びます。データベースはなく、書き込みもしません。プログラムは、設定と
日記ファイルを読むだけです。

## プログラム

| ディレクトリ | プログラム | 内容 |
|---|---|---|
| `cmd/cgihns` | `index.cgi` | CGI プログラム |
| `cmd/fcgihns` | `index.fcgi` | FastCGI プログラム(たとえば `mod_fcgid` 用) |
| `cmd/server` | | ポート 12345 で動く単独の HTTP サーバー。試すためのもの |
| `cmd/cuihns` | | `/` を 1 回要求して、ヘッダーと HTML を表示する。デバッグ用 |

4 つとも、`hns/` の同じコードを使います。

## ビルド

Go が要ります(`go.mod` の `go` を見てください)。cgo は要りません。

```sh
go build -o index.cgi ./cmd/cgihns
go build -o index.fcgi ./cmd/fcgihns
CGO_ENABLED=0 go build -o index.fcgi ./cmd/fcgihns   # 静的なバイナリ
```

`make` は 4 つすべてをビルドします。`server` と `cuihns` は、デバッガを使えるように、
最適化なし(`-N -l`)でビルドされます。

依存は `github.com/cbroglie/mustache` と `golang.org/x/text` です。

## 試す

`sample/` に、設定、テンプレート、いくつかの日記があります。プログラムは、作業
ディレクトリからの相対でファイルを読むので、`sample/` で実行します:

```sh
go build -o /tmp/gohns-server ./cmd/server
cd sample
/tmp/gohns-server &
curl 'http://localhost:12345/?202403'
```

## 設定

設定は、作業ディレクトリにある JSON ファイルです。`config.json` か、
`configpath.json` の `ConfigPath` が指すファイルです:

```json
{ "ConfigPath": "/path/to/gohns.json" }
```

ファイルは、要求のたびに読み直します。読めない、または不正なときは、プログラムは
`500` を返し、理由を標準エラー(Web サーバーのエラーログ)に書きます。壊れた
`configpath.json` は、ログに記録され、`config.json` が使われます。

`sample/config.json` が、完全な例です。プログラムが使う項目は、次のとおりです:

| 項目 | 意味 |
|---|---|
| `Title` | サイトのタイトル。テンプレートでは `{{Title}}` |
| `DiaryDir` | 日記のディレクトリ([日記ファイル](#日記ファイル)を見てください)。作業ディレクトリからの相対か、絶対パス |
| `DiaryCode` | 日記ファイルの文字コード: `eucjp`、`sjis`(`shiftjis`、`cp932`、`windows31j` も可。すべて Shift_JIS として復号します)、`iso2022jp`(`jis`)、`utf8`(または空)。大文字小文字と `-`、`_` は区別しません。出力は常に UTF-8 です |
| `StartYear` | 見に行く最初の年 |
| `DefaultMax`、`DefaultTitleMax` | クエリなしで表示する日数と、タイトルの一覧に表示する日数 |
| `RecentReverse`、`YMDReverse` | 新しい順(`true`)か古い順(`false`)か。最近の日と、クエリで選んだ日のそれぞれ |
| `HtmlSeq`、`Tmpl` | ページと、その mustache のテンプレート |
| `TitleHtmlSeq`、`TitleTmpl` | タイトルの一覧(`?title`)のための、同じもの |

`UserMail`、`Auther`、`MyDiaryURI`、`Theme`、`CacheDir`、`LogDir`、
`RecentTitleReverse`、`YMDTitleReverse`、`FNCountStyle`、`Style`、`PIM` は、読みますが、
プログラムでは使いません。

### ページ

`HtmlSeq` は、ページの部品のリストで、この順に書き出されます:

| 項目 | 書き出すもの |
|---|---|
| `"Head1"` | テンプレート `DocType` と `Head1` |
| `"Head2"` | テンプレート `Head2` |
| `"OpenBody"`、`"CloseBody"` | 同じ名前のテンプレート(`CloseBody` は `{{procTime}}` を表示できます) |
| `"Header"`、`"Footer"` | テンプレート `Title` と `Footer` |
| `"Diaries"` | 日記: `OpenDiaries`、続けて日ごとに `OpenDate`、節、`CloseDate`、最後に `CloseDiaries` |
| `["StaticFile", "path"]` | ファイルを、そのまま |
| `["ConvFile", "path"]` | ファイルを、`DiaryCode` から UTF-8 に変換して |

日記のコマンド(`New`、`Link`、`Img` など)のテンプレートと、プログラムが設定する
変数(`year`、`month`、`day`、`sec`、`uri`、`text` など)は、`sample/config.json` と
`hns/render/render.go` にあるとおりです。

### テーマ

元の hns では、テーマは、設定 `Theme` で名前を指定するディレクトリです(サンプルは
`"Theme": "ice"`)。`theme.ph`(テンプレート、マーク、ユーザー変数を書いた Perl の
ファイル)、画像、必要なら `head.txt` と `foot.txt` を置きます。gohns は、`Theme` も
`theme.ph` も読みません。テンプレートは、設定の `Tmpl` にあります。それでも、テーマに
属するファイルは、その名前のディレクトリに置きます。サンプルの `ice/` は、テーマの
ディレクトリで、スタイルシートと、`HtmlSeq` が `StaticFile` で読み込むページの 2 つの
部品が入っています。

## 日記ファイル

1 日の日記は、ファイル `DiaryDir/YYYY/dYYYYMMDD.hnf` です。たとえば
`diary/2024/d20240301.hnf`。形式は [docs/hnf.ja.md](docs/hnf.ja.md) に書いてあります。

### クエリ

| クエリ | 表示するもの |
|---|---|
| (なし) | 新しい `DefaultMax` 日分 |
| `?YYYYMM` | 1 か月。例: `?202403` |
| `?YYYYMMa`、`?YYYYMMb`、`?YYYYMMc` | 月の 1〜10 日、11〜20 日、21〜31 日 |
| `?YYYYMMDD` | 1 日。例: `?20240301` |
| `?MMDD` | 毎年のこの日。例: `?0101` |
| `?YYYY` | 1 年。4 桁の最初の 2 桁が 12 より大きいときに、年として読みます |
| `?title` | 新しい `DefaultTitleMax` 日分のタイトルの一覧。`TitleHtmlSeq` を使います |
| `CAT=name` | このカテゴリの節だけ。例: `?title&CAT=tech`。複数指定でき、クエリで選んだ日に適用されます |

2 つのキーが日を選ぶとき(`?0101&202403`)は、アルファベット順で最初のものを使います。

## Web サーバーでの動かし方

- プログラムの作業ディレクトリは、`config.json` と、テーマのディレクトリ `ice/`
  (または設定のパスが指すもの)のあるディレクトリでなければなりません。Web サーバーは、CGI プログラムと
  `mod_fcgid` のプログラムを、プログラムのあるディレクトリで実行します。
- `DiaryDir` は、公開ディレクトリの外に置いてください。`sample/` は、テストのため、
  日記を公開ディレクトリの中に置いていて、`sample/.htaccess` が `*.json` と `*.hnf`
  を拒否します。
- Apache: `AddHandler cgi-script .cgi` と `Options +ExecCGI`。または `mod_fcgid` で
  `AddHandler fcgid-script .fcgi`。

`sample/apache/check.sh` は、2 つのプログラムをビルドし、一般ユーザーが
`127.0.0.1:8080` で起動するテスト用の Apache(`sample/apache/httpd-test.conf`)で
`sample/` を配信し、ページを `hns/testdata` と比べます。FreeBSD の `apache24` と
`ap24-mod_fcgid` 用に書いてあります:

```sh
sh sample/apache/check.sh
```

## テスト

```sh
go test ./hns/...
```

`hns/hns_test.go` は、`sample/` の日記を 13 種類のクエリで描画し、HTML を
`hns/testdata` と比べます。保存してあるファイルは、現在の出力を記録したものです。
確かめた出力の変更のあとは、`go test ./hns -update` で更新します。ほかに、壊れた
設定、文字コード、長い行、バイトオーダーマーク、画像の位置、`!` と `URL`、並行する
要求のテストがあります。最後のものは `-race` を付けて実行してください。FreeBSD では
ASLR を切る必要があります:
`proccontrol -m aslr -s disable go test -race ./hns/...`。

FreeBSD 15.1、Go 1.27.4、Apache 2.4.71(`mpm_event`)、`mod_fcgid` 2.3.9 と
`mod_cgid` で試しました。

## 制限

- HNF のよく使う部分だけを実装しています。[docs/hnf.ja.md](docs/hnf.ja.md) にない
  コマンドは、ページに `ERROR NAME is not implemented` が出ます。
- 一度に 1 つの要求だけを処理します(要求の状態をパッケージ変数に持つため)。描画は
  数ミリ秒です。
- テンプレートと日記の URL は、エスケープされません(`{{{uri}}}`)。信頼できる日記
  だけを書いてください。
- どの HTTP メソッドでも、ページを返します。
- Shift_JIS と ISO-2022-JP の復号は、`iconv` と比べていません。EUC-JP は比べて、
  同じでした。

## 参考

元の hns は、ハイパー日記システム・プロジェクトの日記システムです。サイト
(`www.h14m.org`)はなくなりましたが、次のページが Internet Archive に残っています:

- [hns 2.20 ユーザーズ・ガイド](https://web.archive.org/web/20230512122453/http://www.h14m.org/docs/)
- [hnf の記述方法](https://web.archive.org/web/20230512122453/http://www.h14m.org/docs/hnf-j.html)。HNF 形式の仕様

## ライセンス

MIT。[LICENSE](LICENSE) を見てください。
