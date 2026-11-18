package collector

import (
	"fmt"
	g "github.com/a-rin/gohns/hns/ghns"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The patterns are compiled once, not for every line of every request.
var (
	diaryFileName = regexp.MustCompile(`^d\d{8}\.hnf$`)
	diaryFileDate = regexp.MustCompile(`d(\d{4})(\d{2})(\d{2})\.hnf$`)
)

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func CollectYear(year int) []string {
	//  g.Write([]byte(fmt.Sprintf("func CollectYear(%d)<br>", year)))
	files := []string{}
	dirName := fmt.Sprintf("%s/%d", g.Config.DiaryDir, year)
	fInfo, err := os.Stat(dirName)
	if err != nil || !fInfo.Mode().IsDir() {
		return files
	}
	ents, err := os.ReadDir(dirName)
	if err != nil {
		return files
	}
	resp := diaryFileName
	for _, entry := range ents {
		if resp.MatchString(entry.Name()) {
			files = append(files, fmt.Sprintf("%s/%s", dirName, entry.Name()))
		}
	}
	return files
}

func CollectDay(month, day int) []string {
	files := []string{}
	for y := g.Config.StartYear; y <= time.Now().Year(); y++ {
		path := fmt.Sprintf("%s/%d/d%04d%02d%02d.hnf", g.Config.DiaryDir, y, y, month, day)
		info, err := os.Stat(path)
		if (err == nil) && (info.Mode().IsRegular()) {
			files = append(files, path)
		}
	}
	return files
}

func RecentFiles(maxDays int, reverseF bool) []string {
	//  g.Write([]byte(fmt.Sprintf("func RecentFiles(%d)<br>", maxDays)))
	var cnt = 0
	dfiles := []string{}
	now := time.Now()
	year := now.Year()
	for (cnt < maxDays) && (g.Config.StartYear <= year) {
		dfiles = append(dfiles, CollectYear(year)...)
		cnt = len(dfiles)
		year--
	}
	sort.Strings(dfiles)
	reverse(dfiles)
	if cnt > maxDays {
		dfiles = dfiles[:maxDays]
	}
	if !reverseF {
		reverse(dfiles)
	}
	return dfiles
}

func YMDFiles(year, month int, daySpec string, reverseF bool) []string {
	var res = []string{}
	var id int
	var err error
	var daymin, daymax int
	days := strings.ToLower(daySpec)
	id, err = strconv.Atoi(days)
	if err != nil {
		id = 0
	}
	if days == "a" {
		daymin, daymax = 1, 10
	} else if days == "b" {
		daymin, daymax = 11, 20
	} else if days == "c" {
		daymin, daymax = 21, 31
	} else if id == 0 {
		daymin, daymax = 1, 31
	} else {
		daymin, daymax = id, id
	}
	dfiles := CollectYear(year)
	if month == 0 {
		res = dfiles
	} else {
		for _, fn := range dfiles {
			subs := diaryFileDate.FindStringSubmatch(fn)
			if subs != nil {
				dyear, _ := strconv.Atoi(subs[1])
				dmonth, _ := strconv.Atoi(subs[2])
				dday, _ := strconv.Atoi(subs[3])
				if (dyear == year) && (dmonth == month) && (daymin <= dday) && (dday <= daymax) {
					res = append(res, fn)
				}
			}
		}
	}
	sort.Strings(res)
	reverse(res)
	if reverseF {
		reverse(res)
	}
	return res
}

func DayFiles(month, day int, reverseF bool) []string {
	res := CollectDay(month, day)
	sort.Strings(res)
	if reverseF {
		reverse(res)
	}
	return res
}

func LastMod(dfiles []string) time.Time {
	var res time.Time
	for _, fn := range dfiles {
		finfo, err := os.Stat(fn)
		if err == nil {
			ftime := finfo.ModTime()
			if res.Before(ftime) {
				res = ftime
			}
		}

	}
	return res
}
