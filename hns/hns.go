package hns

import (
	"bytes"
	"log"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
	//    "fmt"
	"github.com/a-rin/gohns/hns/collector"
	"github.com/a-rin/gohns/hns/config"
	g "github.com/a-rin/gohns/hns/ghns"
	"github.com/a-rin/gohns/hns/render"
	"regexp"
	"strconv"
	"strings"
)

// The patterns are compiled once, not for every line of every request.
var (
	ymQuery = regexp.MustCompile(`^(\d{4})(\d{2})(\d*|[abc])?$`)
	mdQuery = regexp.MustCompile(`^(\d{2})(\d{2})$`)
)

type TextProcType struct {
}

func (self *TextProcType) Write(data []byte) (int, error) {
	return g.Write(data)
}

func (self *TextProcType) OpenSec(_ string) {
}

func (self *TextProcType) CloseSec() {
}

// mu lets one request run at a time: the state of a request is kept in the
// package variables of ghns (Config, Conv, Write and StartTime).
// ponytail: no parallel requests. Rendering takes milliseconds, which is
// enough for a personal diary. To serve requests in parallel, move that state
// into a value that is made per request and passed to render and collector.
var mu sync.Mutex

// response keeps the output of one request, so that mu is not held while a
// slow client reads it.
type response struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (res *response) Header() http.Header         { return res.header }
func (res *response) Write(p []byte) (int, error) { return res.body.Write(p) }
func (res *response) WriteHeader(code int) {
	if res.status == 0 {
		res.status = code
	}
}

func Handler(w http.ResponseWriter, r *http.Request) {
	res := &response{header: http.Header{}}
	func() {
		mu.Lock()
		defer mu.Unlock()
		handle(res, r)
	}()
	for k, v := range res.header {
		w.Header()[k] = v
	}
	if res.status != 0 {
		w.WriteHeader(res.status)
	}
	w.Write(res.body.Bytes())
}

func handle(w http.ResponseWriter, r *http.Request) {
	var dfiles []string
	var recentF = true
	g.Write = nil
	g.StartTime = time.Now()
	if err := g.InitG(); err != nil {
		log.Printf("gohns: %v", err)
		http.Error(w, "gohns: cannot read the configuration", http.StatusInternalServerError)
		return
	}
	defer g.TermG()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var qStrings = map[string][]string{}
	u := r.URL
	if u != nil {
		qStr := u.RawQuery
		//      g.Write([]byte(fmt.Sprintf("<!--\nQueryStr: %v\n-->\n", qStr)))
		qStrings, _ = url.ParseQuery(qStr)
	}

	doneCollect := false
	titleF := false
	// A map is read in a random order, so two keys that both select diaries
	// (?0101&202403) gave a different page on each request: take them in order.
	keys := make([]string, 0, len(qStrings))
	for k := range qStrings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, qVar := range keys {
		//      g.Write([]byte(fmt.Sprintf("<!--\n%v: %v\n-->\n", qVar, qVal)))
		if !doneCollect {
			subs := ymQuery.FindStringSubmatch(qVar)
			if subs != nil {
				y, _ := strconv.Atoi(subs[1])
				m, _ := strconv.Atoi(subs[2])
				dfiles = collector.YMDFiles(y, m, subs[3], g.Config.YMDReverse)
				recentF = false
				doneCollect = true
			}
			subs = mdQuery.FindStringSubmatch(qVar)
			if subs != nil {
				m, _ := strconv.Atoi(subs[1])
				d, _ := strconv.Atoi(subs[2])
				if m > 12 {
					dfiles = collector.YMDFiles(m*100+d, 0, "", g.Config.YMDReverse)
				} else {
					dfiles = collector.DayFiles(m, d, g.Config.YMDReverse)
				}
				recentF = false
				doneCollect = true
			}
		}
		if strings.ToLower(qVar) == "title" {
			titleF = true
		}
	}

	if recentF {
		if titleF {
			dfiles = collector.RecentFiles(g.Config.DefaultTitleMax, g.Config.RecentReverse)
		} else {
			dfiles = collector.RecentFiles(g.Config.DefaultMax, g.Config.RecentReverse)
		}
	}
	g.Write = w.Write
	if len(dfiles) > 0 {
		dateStr := collector.LastMod(dfiles).UTC().Format(http.TimeFormat)
		w.Header().Set("Last-Modified", dateStr)
		w.WriteHeader(http.StatusOK)
		//      g.Write([]byte(fmt.Sprintf("<!--\nLast-Modified:%s\n-->\n", dateStr)))
	}
	if r.Method == "HEAD" {
		return
	}
	//    g.Write([]byte(fmt.Sprintf("<!--\n%v\n-->\n", dfiles)))

	var seq *config.HtmlSeqType
	var tmpl *config.TmplType
	if titleF {
		seq = &g.Config.TitleHtmlSeq
		tmpl = &g.Config.TitleTmpl
	} else {
		seq = &g.Config.HtmlSeq
		tmpl = &g.Config.Tmpl
	}

	var tp TextProcType
	ro := render.NewRender()
	ro.Render(dfiles, seq, tmpl, qStrings[`CAT`], &tp)
}
