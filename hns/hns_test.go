package hns_test

import (
	"flag"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
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

// TestHandlerGolden renders the diaries in ../sample and compares the HTML
// with testdata. Run "go test ./hns -update" to regenerate testdata after
// a change in the output that you have checked by hand.
func TestHandlerGolden(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// The config and the diaries are read relative to the working directory.
	if err := os.Chdir(filepath.Join(wd, "..", "sample")); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			hns.Handler(rec, httptest.NewRequest("GET", "/"+c.query, nil))
			got := procTime.ReplaceAllString(rec.Body.String(), "<hr>TIME<br>")
			golden := filepath.Join(wd, "testdata", c.name+".html")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o775); err != nil {
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
