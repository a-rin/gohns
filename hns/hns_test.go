package hns_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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

// withDiary runs the rest of a test in a directory that has the config of
// ../sample and one diary, d20240101.hnf.
func withDiary(t *testing.T, diary string) {
	t.Helper()
	withDiaryConfig(t, diary, nil)
}

// withDiaryConfig is withDiary, with a change of the templates of the config.
func withDiaryConfig(t *testing.T, diary string, edit func(tmpl map[string]any)) {
	t.Helper()
	config, err := os.ReadFile(filepath.Join(testdata, "..", "..", "sample", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		var c map[string]any
		if err := json.Unmarshal(config, &c); err != nil {
			t.Fatal(err)
		}
		edit(c["Tmpl"].(map[string]any))
		if config, err = json.Marshal(c); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "diary", "2024"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), config, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "diary", "2024", "d20240101.hnf"), []byte(diary), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, dir)
}

// A line of more than 64 KB used to end the reading of the file: the lines
// after it were lost.
func TestLongLine(t *testing.T) {
	withDiary(t, "OK\nNEW t\n"+strings.Repeat("x", 200000)+"\nafter the long line\n")
	if got := render("?20240101"); !strings.Contains(got, "after the long line") {
		t.Error("the line after the long line is missing")
	}
}

// A byte order mark before OK used to hide the whole diary.
func TestByteOrderMark(t *testing.T) {
	withDiary(t, string([]byte{0xef, 0xbb, 0xbf})+"OK\nNEW t\nbody of the diary\n")
	if got := render("?20240101"); !strings.Contains(got, "body of the diary") {
		t.Error("the diary with a byte order mark is empty")
	}
}

// Two keys that both select diaries gave a different page on each request,
// because a map is read in a random order.
func TestQueryOrder(t *testing.T) {
	first := render("?0101&202403")
	for i := 0; i < 30; i++ {
		if render("?0101&202403") != first {
			t.Fatal("the same query gave different pages")
		}
	}
}

// IMG {r|l|n} file [alt] and LIMG url {r|l|n} file [alt] are the forms of the
// original hns; the forms without the alignment word are kept.
func TestImageAlign(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{"IMG r a.png alt text", `<img src="a.png" alt="alt text" class="align-right">`},
		{"IMG l a.png alt", `<img src="a.png" alt="alt" class="align-left">`},
		{"IMG l a.png", `<img src="a.png" alt="" class="align-left">`},
		{"IMG n a.png alt", `<img src="a.png" alt="alt">`},
		{"IMG a.png alt", `<img src="a.png" alt="alt">`},
		{"IMG a.png", `<img src="a.png" alt="">`},
		{"LIMG http://x/ r a.png alt", `<a href="http://x/"><img src="a.png" alt="alt" class="align-right"></a>`},
		{"LIMG http://x/ n a.png alt", `<a href="http://x/"><img src="a.png" alt="alt"></a>`},
		{"LIMG http://x/ a.png alt", `<a href="http://x/"><img src="a.png" alt="alt"></a>`},
		// the alignment of one image must not stay for the next
		{"IMG r a.png x\nIMG b.png y", `<img src="b.png" alt="y">`},
	} {
		withDiary(t, "OK\nNEW t\n"+c.line+"\n")
		if got := render("?20240101"); !strings.Contains(got, c.want) {
			t.Errorf("%q: the page does not contain %s", c.line, c.want)
		}
	}
}

// ! is written by the template Comment, !# never.
func TestComments(t *testing.T) {
	withDiary(t, "OK\nNEW t\n! a visible note\n!# a hidden note\n! a --> <script>x</script>\n")
	got := render("?20240101")
	if !strings.Contains(got, "<!-- a visible note-->") {
		t.Error("the note of ! is not written as a comment")
	}
	if strings.Contains(got, "hidden note") {
		t.Error("the note of !# is written")
	}
	if !strings.Contains(got, "a --&gt; &lt;script&gt;x&lt;/script&gt;") {
		t.Error("the text of a comment is not escaped")
	}
	// the config of a site that keeps its notes private
	withDiaryConfig(t, "OK\nNEW t\n! a visible note\n", func(tmpl map[string]any) { tmpl["Comment"] = "" })
	if strings.Contains(render("?20240101"), "visible note") {
		t.Error("a note is written although the template Comment is empty")
	}
}

// URL url text is a link that shows the url.
func TestURL(t *testing.T) {
	withDiary(t, "OK\nNEW t\nURL http://x/ some text\nURL onlyone\n")
	got := render("?20240101")
	if !strings.Contains(got, `<a href="http://x/">some text(http://x/)</a>`) {
		t.Error("URL does not write the link with the url")
	}
	if !strings.Contains(got, `ERROR "URL" must have 2 param.`) {
		t.Error("URL without a text is not an error")
	}
	// a config from before URL, without the template Url, uses the template Link
	withDiaryConfig(t, "OK\nNEW t\nURL http://x/ some text\n", func(tmpl map[string]any) { delete(tmpl, "Url") })
	if !strings.Contains(render("?20240101"), `<a href="http://x/">some text</a>`) {
		t.Error("URL does not use the template Link when there is no template Url")
	}
}
