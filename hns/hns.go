package hns

import (
	"net/http"
	"net/url"
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

type TextProcType struct {
}

func (self *TextProcType) Write(data []byte) (int, error) {
	return g.Write(data)
}

func (self *TextProcType) OpenSec(_ string) {
}

func (self *TextProcType) CloseSec() {
}

func Handler(w http.ResponseWriter, r *http.Request) {
	var dfiles []string
	var recentF = true
	g.Write = nil
	g.StartTime = time.Now()
	needTerm := g.InitG()
	if needTerm {
		defer g.TermG()
	}
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
	for qVar, _ := range qStrings {
		//      g.Write([]byte(fmt.Sprintf("<!--\n%v: %v\n-->\n", qVar, qVal)))
		if !doneCollect {
			subs := regexp.MustCompile(`^(\d{4})(\d{2})(\d*|[abc])?$`).FindStringSubmatch(qVar)
			if subs != nil {
				y, _ := strconv.Atoi(subs[1])
				m, _ := strconv.Atoi(subs[2])
				dfiles = collector.YMDFiles(y, m, subs[3], g.Config.YMDReverse)
				recentF = false
				doneCollect = true
			}
			subs = regexp.MustCompile(`^(\d{2})(\d{2})$`).FindStringSubmatch(qVar)
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
