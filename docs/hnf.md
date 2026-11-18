# The HNF format as gohns reads it

English | [日本語](hnf.ja.md)

gohns implements the commonly used part of HNF, not all of it. This describes
what `hns/render/render.go` reads, and was checked with the diaries in
`sample/diary`. A command that is not listed here gives
`ERROR NAME is not implemented` in the page.

The specification of HNF is the page [How to write an hnf](https://web.archive.org/web/20230512122453/http://www.h14m.org/docs/hnf-j.html)
of the documentation of hns 2.20 (in Japanese). See
[Differences from the original](#differences-from-the-original) for where gohns
does not follow it.

## Files

The diary of a day is `DiaryDir/YYYY/dYYYYMMDD.hnf`. The file is in the
character set `DiaryCode` of the configuration. A UTF-8 byte order mark at the
start of the file is accepted. A line can be up to 1 MB long.

## Lines

- The text before the first line `OK` is ignored. The diary starts after it.
- Blank lines are ignored. White space at the end of a line is removed.
- A line that starts with `!` is a comment. It is written by the template
  `Comment`, as `<!--{{text}}-->` in the sample, so it is in the HTML source of
  the page. A site that wants to keep its notes out of the page makes the
  template `Comment` empty. A line that starts with `!#` is never written.
- A line that ends with `~` is written with the template `TextLn` (the sample
  adds `<br>`), the `~` removed. Any other line is written with `Text`.
- A line that is an upper-case word, or an upper-case word and a space and
  anything, is a command: `NEW Title`. A word starting with `/` closes
  something: `/UL`. **A text line must not start with such a word**, for
  example `A new day`: the `A` is read as a command and gives
  `ERROR A is not implemented`. A word with a lower-case letter, such as `The`,
  is text.

## Commands

A day starts with its first `NEW` or `LNEW`.

| Command | Parameters | Writes |
|---|---|---|
| `CAT` | names, separated by spaces | sets the categories of the next `NEW` or `LNEW`; `CAT=name` in a query shows only these sections |
| `NEW` | title | a section |
| `LNEW` | `uri` title | a section whose title is a link |
| `SUB`, `/SUB` | title | a sub-section; `/SUB` closes it |
| `LSUB` | `uri` title | a sub-section whose title is a link |
| `P`, `/P` | | a paragraph |
| `LINK` | `uri` text | a link |
| `URL` | `uri` text | a link that shows the `uri`, by the template `Url`. A config without that template uses `Link` |
| `STRIKE` | text | struck-out text |
| `LSTRIKE` | `uri` text | a link of struck-out text |
| `STRONG` | text | strong text |
| `SPAN` | class text | a span with the class |
| `CITE`, `/CITE` | | a quotation |
| `PRE`, `/PRE` | | preformatted text |
| `UL`, `/UL`, `OL`, `/OL` | | lists |
| `LI` | text | a list item |
| `DL`, `/DL` | | a definition list |
| `DT`, `DD` | text | a term, a description |
| `DIV`, `/DIV` | class | a block with the class |
| `FN`, `/FN` | | the lines between them are a footnote, written at the end of the section |
| `IMG` | [`r`\|`l`\|`n`] `uri` [alt] | an image. `r`, `l` and `n` are the alignment of the original: right, left and none. Without the word, `IMG uri [alt]` also works |
| `LIMG` | link-`uri` [`r`\|`l`\|`n`] image-`uri` [alt] | an image that is a link; the same for the alignment |
| `PHOTO` | id text | the template `Photo` (not in the sample) |

`LINK`, `LSTRIKE`, `LNEW`, `LSUB` and `LIMG` need their parameters; without
them the page has `ERROR "LINK" must have 2 param.` and so on. An unknown
command gives `ERROR NAME is not implemented`.

What a command writes is decided by the template of the same name in the
configuration (`New`, `Link`, `Ul`, `_Ul`, ...), so it can be changed there.

## Differences from the original

Checked against the specification above, not by running the original hns.

- `IMG` and `LIMG` read the form of the original, `IMG {r|l|n} file alt` and
  `LIMG url {r|l|n} file alt`, and also the form without the alignment word,
  `IMG file [alt]` and `LIMG url file [alt]`. The alignment is not written as
  the `align` attribute: the template gets the variable `align` (`right`,
  `left` or empty), and the sample adds the class `align-right` or `align-left`.
- `!` is written by the template `Comment`, which can be made empty. The
  original always writes an HTML comment for it.
- These commands of the original are not implemented: `RLNEW`, `RLSUB`,
  `RLINK`, `GRP`, `FONT`, `MARK`, `ALIAS`, `RT` and `/RT`, and the special
  forms of the URL of `LINK` and others (`#YYYYMMDD`, `ISBN:`, `CD:`, ...). The
  variables in the header, before `OK`, are ignored.
- `PHOTO` and `/SUB` are in gohns, and not on that page.

The original also warns that a word of upper-case letters at the start of a
line may become a command in the future, even if it is none now.

## Example

```
OK
CAT tech
NEW The title
The line of text.~
Another line.
SUB A sub-section
LINK https://example.com/ a link
UL
LI first
LI second
/UL
! this line is a comment
```

The `SUB` line is a command, so `A sub-section` after it is the parameter, not
text. A text line that began with `A ` would be read as the command `A`.
