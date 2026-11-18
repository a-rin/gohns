# gohns

A Go implementation of the hns renderer, running as CGI or FastCGI.

hns stands for ハイパー日記システム (Hyper Nikki System), and HNF for Hyper Nikki
File (ハイパー日記ファイル). gohns implements the commonly used part of the HNF
format, not all of it: see [docs/hnf.md](docs/hnf.md) for what it reads and how
it differs from the original.

gohns reads diary files in the HNF format, one file per day, and renders them
as HTML pages with [mustache](https://github.com/cbroglie/mustache) templates.
Which days are shown is chosen by the query string of the URL (`?202403`,
`?0101`, `?title`, ...). There is no database and no writing: the program only
reads the configuration and the diary files.

## Programs

| Source | Program | What it is |
|---|---|---|
| `cgihns.go` | `index.cgi` | CGI program |
| `fcgihns.go` | `index.fcgi` | FastCGI program (for example for `mod_fcgid`) |
| `server.go` | | standalone HTTP server on port 12345, for trying it out |
| `cuihns.go` | | requests `/` once and prints the header and the HTML, for debugging |

All four use the same code in `hns/`.

## Build

You need Go (see `go` in `go.mod`). cgo is not needed.

```sh
go build -o index.cgi cgihns.go
go build -o index.fcgi fcgihns.go
CGO_ENABLED=0 go build -o index.fcgi fcgihns.go   # a static binary
```

`make` builds all four; `server` and `cuihns` are built without optimization
(`-N -l`) so that a debugger can be used.

The dependencies are `github.com/cbroglie/mustache` and `golang.org/x/text`.

## Try it

`sample/` has a configuration, templates and some diary entries. The program
reads its files relative to the working directory, so run it in `sample/`:

```sh
go build -o /tmp/gohns-server server.go
cd sample
/tmp/gohns-server &
curl 'http://localhost:12345/?202403'
```

## Configuration

The configuration is a JSON file in the working directory: `config.json`, or
the file named by `ConfigPath` in `configpath.json`:

```json
{ "ConfigPath": "/path/to/gohns.json" }
```

The file is read again on every request. If it cannot be read or is invalid,
the program answers `500` and writes the reason to its standard error (the
error log of the web server). A broken `configpath.json` is logged and
`config.json` is used.

`sample/config.json` is a complete example. The fields that the program uses:

| Field | Meaning |
|---|---|
| `Title` | title of the site, `{{Title}}` in the templates |
| `DiaryDir` | directory of the diaries (see [Diary files](#diary-files)). Relative to the working directory, or absolute |
| `DiaryCode` | character set of the diary files: `eucjp`, `sjis` (also `shiftjis`, `cp932`, `windows31j`; all are decoded as Shift_JIS), `iso2022jp` (`jis`), `utf8` (or empty). Case, `-` and `_` do not matter. The output is always UTF-8 |
| `StartYear` | the first year to look at |
| `DefaultMax`, `DefaultTitleMax` | number of days shown without a query, and in the list of titles |
| `RecentReverse`, `YMDReverse` | newest first (`true`) or oldest first (`false`), for the recent days and for the days of a query |
| `HtmlSeq`, `Tmpl` | the page and its mustache templates |
| `TitleHtmlSeq`, `TitleTmpl` | the same for the list of titles (`?title`) |

`UserMail`, `Auther`, `MyDiaryURI`, `Theme`, `CacheDir`, `LogDir`,
`RecentTitleReverse`, `YMDTitleReverse`, `FNCountStyle`, `Style` and `PIM` are
read but not used by the program.

### The page

`HtmlSeq` is a list of the parts of the page, written in this order:

| Entry | Writes |
|---|---|
| `"Head1"` | the templates `DocType` and `Head1` |
| `"Head2"` | the template `Head2` |
| `"OpenBody"`, `"CloseBody"` | the templates of the same name (`CloseBody` can show `{{procTime}}`) |
| `"Header"`, `"Footer"` | the templates `Title` and `Footer` |
| `"Diaries"` | the days: `OpenDiaries`, then for every day `OpenDate`, the sections and `CloseDate`, then `CloseDiaries` |
| `["StaticFile", "path"]` | the file, as it is |
| `["ConvFile", "path"]` | the file, converted from `DiaryCode` to UTF-8 |

The templates for the diary commands (`New`, `Link`, `Img`, ...) and the
variables that the program sets (`year`, `month`, `day`, `sec`, `uri`,
`text`, ...) are those of `sample/config.json` and `hns/render/render.go`.

## Diary files

The diary of a day is the file `DiaryDir/YYYY/dYYYYMMDD.hnf`, for example
`diary/2024/d20240301.hnf`. The format is described in [docs/hnf.md](docs/hnf.md).

### Queries

| Query | Shows |
|---|---|
| (none) | the newest `DefaultMax` days |
| `?YYYYMM` | a month, for example `?202403` |
| `?YYYYMMa`, `?YYYYMMb`, `?YYYYMMc` | days 1 to 10, 11 to 20, 21 to 31 of a month |
| `?YYYYMMDD` | one day, for example `?20240301` |
| `?MMDD` | this day of every year, for example `?0101` |
| `?YYYY` | a year. Four digits are a year when the first two are over 12 |
| `?title` | a list of the titles of the newest `DefaultTitleMax` days, with `TitleHtmlSeq` |
| `CAT=name` | only the sections with this category, for example `?title&CAT=tech`. It can be given more than once, and is applied to the days of the query |

When two keys select days (`?0101&202403`), the first one in alphabetical
order is used.

## Running it under a web server

- The working directory of the program must be the directory with `config.json`
  and `ice/` (or whatever the paths in the configuration name). A web server
  runs a CGI program and a `mod_fcgid` program in the directory of the program.
- Put `DiaryDir` outside the document root. `sample/` keeps the diaries inside
  it, for the test, and `sample/.htaccess` refuses `*.json` and `*.hnf`.
- Apache: `AddHandler cgi-script .cgi` with `Options +ExecCGI`, or with
  `mod_fcgid`, `AddHandler fcgid-script .fcgi`.

`sample/apache/check.sh` builds both programs, serves `sample/` with a test
instance of Apache that an ordinary user starts on `127.0.0.1:8080`
(`sample/apache/httpd-test.conf`), and checks the pages against
`hns/testdata`. It is written for FreeBSD with `apache24` and `ap24-mod_fcgid`:

```sh
sh sample/apache/check.sh
```

## Tests

```sh
go test ./hns/...
```

`hns/hns_test.go` renders the diaries of `sample/` for 13 queries and compares
the HTML with `hns/testdata`; the golden files record the current output.
After a change of the output that you have checked, update them with
`go test ./hns -update`. There are also tests for a broken configuration, for
the character sets, for long lines, for a byte order mark, and for concurrent
requests. Use `-race` for the last one; on FreeBSD ASLR must be off:
`proccontrol -m aslr -s disable go test -race ./hns/...`.

It was tested on FreeBSD 15.1 with Go 1.27.4, Apache 2.4.71 (`mpm_event`) with
`mod_fcgid` 2.3.9 and `mod_cgid`.

## Limits

- Only the commonly used part of HNF is implemented. A command that is not in
  [docs/hnf.md](docs/hnf.md) gives `ERROR NAME is not implemented` in the page.
- One request is handled at a time (the state of a request is kept in package
  variables). Rendering takes a few milliseconds.
- The URLs in templates and diaries are not escaped (`{{{uri}}}`). Only write
  diaries that you trust.
- Any HTTP method gets the page.
- The decoders for Shift_JIS and ISO-2022-JP were not compared with `iconv`;
  those for EUC-JP were, and they are the same.

## References

The original hns is a diary system of the ハイパー日記システム・プロジェクト.
Its site, www.h14m.org, is gone; these pages are in the Internet Archive:

- [hns 2.20 user guide](https://web.archive.org/web/20230512122453/http://www.h14m.org/docs/) (in Japanese)
- [How to write an hnf](https://web.archive.org/web/20230512122453/http://www.h14m.org/docs/hnf-j.html) (in Japanese), the specification of the HNF format

## License

MIT, see [LICENSE](LICENSE).
