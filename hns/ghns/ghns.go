package ghns

import (
	"fmt"
	"strings"
	"time"

	"github.com/a-rin/gohns/hns/config"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
)

var (
	// Conv decodes the diary files, in the character set named by DiaryCode,
	// to UTF-8.
	Conv      *encoding.Decoder
	Config    *config.ConfigType
	StartTime time.Time
	Write     func([]byte) (int, error)
)

// newDecoder returns a decoder from the character set named by code to UTF-8.
// Case, "-" and "_" in the name do not matter. An empty name means UTF-8.
// ponytail: cp932 and windows-31j are decoded as Shift_JIS, so the NEC and
// IBM extension characters become U+FFFD. Add a table for them if a diary
// needs it.
func newDecoder(code string) (*encoding.Decoder, error) {
	var enc encoding.Encoding
	var jis bool
	switch strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(code)) {
	case "", "utf8":
		enc = unicode.UTF8
	case "eucjp":
		enc, jis = japanese.EUCJP, true
	case "sjis", "shiftjis", "cp932", "windows31j":
		enc = japanese.ShiftJIS
	case "iso2022jp", "jis":
		enc = japanese.ISO2022JP
	default:
		return nil, fmt.Errorf("unknown character set %q", code)
	}
	d := enc.NewDecoder()
	if jis {
		d = &encoding.Decoder{Transformer: transform.Chain(d, runes.Map(jisX0208))}
	}
	return d, nil
}

// jisX0208 turns the seven characters that x/text takes from the Microsoft
// table (wave dash as U+FF5E and so on) into those of the JIS X 0208 table,
// which iconv, used before, gives for EUC-JP. Compared over all two-byte codes
// of EUC-JP: 6872 are the same, these 7 differ.
func jisX0208(r rune) rune {
	switch r {
	case 0x2015:
		return 0x2014 // A1BD
	case 0xff5e:
		return 0x301c // A1C1 wave dash
	case 0x2225:
		return 0x2016 // A1C2
	case 0xff0d:
		return 0x2212 // A1DD
	case 0xffe0:
		return 0x00a2 // A1F1
	case 0xffe1:
		return 0x00a3 // A1F2
	case 0xffe2:
		return 0x00ac // A2CC
	}
	return r
}

// InitG reads the configuration and makes the decoder. Call TermG when it
// returns nil.
func InitG() error {
	if Config != nil {
		return nil
	}
	config, err := config.ReadConfig()
	if err != nil {
		return err
	}
	conv, err := newDecoder(config.DiaryCode)
	if err != nil {
		return fmt.Errorf("DiaryCode: %w", err)
	}
	Config, Conv = config, conv
	return nil
}

func TermG() {
	Config, Conv = nil, nil
}

func init() {
}
