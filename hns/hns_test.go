package hns_test

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/a-rin/gohns/hns"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// The processing time in the footer changes on every run.
var procTime = regexp.MustCompile(`<hr>[^<]*<br>`)

var cases = []struct{ name, query string }{
	{"recent", ""},
	{"monthday_0101", "?0101"},
	{"month_202403", "?202403"},
	{"month_202403a", "?202403a"},
	{"month_202403b", "?202403b"},
	{"month_202403c", "?202403c"},
	{"day_20240301", "?20240301"},
	{"year_2023", "?2023"},
	{"japanese_202405", "?202405"},
	{"errors_202406", "?202406"},
	{"title", "?title"},
	{"title_cat_tech", "?title&CAT=tech"},
	{"recent_cat_diary", "?CAT=diary"},
}

var testdata string

// The config and the diaries are read relative to the working directory,
// so every test runs in ../sample. Tests that use it must not run in parallel
// with each other, but the handler itself is called concurrently below.
func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err == nil {
		testdata = filepath.Join(wd, "testdata")
		err = os.Chdir(filepath.Join(wd, "..", "sample"))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func render(query string) string {
	rec := httptest.NewRecorder()
	hns.Handler(rec, httptest.NewRequest("GET", "/"+query, nil))
	return procTime.ReplaceAllString(rec.Body.String(), "<hr>TIME<br>")
}

// TestHandlerGolden renders the diaries in ../sample and compares the HTML
// with testdata. Run "go test ./hns -update" to regenerate testdata after
// a change in the output that you have checked by hand.
func TestHandlerGolden(t *testing.T) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := render(c.query)
			golden := filepath.Join(testdata, c.name+".html")
			if *update {
				if err := os.MkdirAll(testdata, 0o775); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, []byte(got), 0o664); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("%q differs from %s", c.query, golden)
			}
		})
	}
}

// TestHandlerConcurrent sends the same queries from several goroutines, as a
// FastCGI server does, and expects the same output as TestHandlerGolden.
// Run it with -race.
func TestHandlerConcurrent(t *testing.T) {
	if *update {
		t.Skip("updating")
	}
	want := map[string]string{}
	for _, c := range cases {
		b, err := os.ReadFile(filepath.Join(testdata, c.name+".html"))
		if err != nil {
			t.Fatal(err)
		}
		want[c.name] = string(b)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, c := range cases {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("%q: panic: %v", c.query, r)
						}
					}()
					if got := render(c.query); got != want[c.name] {
						t.Errorf("%q: output differs", c.query)
					}
				}()
			}
		}()
	}
	wg.Wait()
}

func inDir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
}

// TestBadConfig expects an error page, not a panic, and not the end of the
// process (a FastCGI server must survive a broken configuration).
func TestBadConfig(t *testing.T) {
	for _, c := range []struct{ name, config string }{
		{"missing", ""},
		{"broken json", "{"},
		{"unknown DiaryCode", `{"DiaryCode": "no-such-code"}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if c.config != "" {
				if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(c.config), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			inDir(t, dir)
			rec := httptest.NewRecorder()
			hns.Handler(rec, httptest.NewRequest("GET", "/", nil))
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
			}
		})
	}
}

// TestEUCJPDiary renders a diary file in EUC-JP, the character set of the
// diaries that this program was written for.
func TestEUCJPDiary(t *testing.T) {
	config, err := os.ReadFile("config.json")
	if err != nil {
		t.Fatal(err)
	}
	config = bytes.Replace(config, []byte(`"DiaryCode": "UTF-8"`), []byte(`"DiaryCode": "eucjp"`), 1)
	title, _ := hex.DecodeString("c6fccbdcb8ec") // 日本語
	body, _ := hex.DecodeString("cbdccab8")      // 本文
	diary := append([]byte("OK\nNEW "), title...)
	diary = append(diary, '\n')
	diary = append(diary, body...)
	diary = append(diary, '\n')

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "diary", "2024"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"config.json":              config,
		"diary/2024/d20240101.hnf": diary,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inDir(t, dir)
	got := render("?20240101")
	for _, want := range []string{"日本語", "本文"} {
		if !strings.Contains(got, want) {
			t.Errorf("the output does not contain %q", want)
		}
	}
}
