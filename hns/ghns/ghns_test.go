package ghns

import (
	"encoding/hex"
	"testing"
)

func TestNewDecoder(t *testing.T) {
	const want = "日本語"
	for _, c := range []struct{ code, hex string }{
		{"eucjp", "c6fccbdcb8ec"},
		{"EUC-JP", "c6fccbdcb8ec"},
		{"Shift_JIS", "93fa967b8cea"},
		{"sjis", "93fa967b8cea"},
		{"ISO-2022-JP", "1b2442467c4b5c386c1b2842"},
		{"UTF-8", "e697a5e69cace8aa9e"},
		{"", "e697a5e69cace8aa9e"},
	} {
		b, err := hex.DecodeString(c.hex)
		if err != nil {
			t.Fatal(err)
		}
		d, err := newDecoder(c.code)
		if err != nil {
			t.Errorf("%q: %v", c.code, err)
			continue
		}
		got, err := d.Bytes(b)
		if err != nil || string(got) != want {
			t.Errorf("%q: got %q, %v; want %q", c.code, got, err, want)
		}
	}
	if _, err := newDecoder("no-such-code"); err == nil {
		t.Error("an unknown character set must be an error")
	}
}

// The seven characters on which x/text and iconv differ.
func TestEUCJPAsIconv(t *testing.T) {
	b, err := hex.DecodeString("a1bda1c1a1c2a1dda1f1a1f2a2cc")
	if err != nil {
		t.Fatal(err)
	}
	d, err := newDecoder("eucjp")
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.Bytes(b)
	const want = "—〜‖−¢£¬"
	if err != nil || string(got) != want {
		t.Errorf("got %+q, %v; want %+q", got, err, want)
	}
}
