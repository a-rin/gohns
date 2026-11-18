package render

import (
	"github.com/a-rin/gohns/hns/config"
	g "github.com/a-rin/gohns/hns/ghns"
	"bufio"
	"fmt"
	mu "github.com/cbroglie/mustache"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type hnfState int

const (
	hsInit = iota
	hsOk
	hsErr
)

type TmplDataType map[string]interface{}

type HnfCmd struct {
	obj *RenderObj
}

type HtmlPartCmd struct {
	obj *RenderObj
}

type RenderObj struct {
	dfiles      []string
	filterCat   []string
	tmpl        *config.TmplType
	htmlSeq     *config.HtmlSeqType
	state       hnfState
	doc         TmplDataType
	dateOpen    bool
	secOpen     bool
	subOpen     bool
	fnOpen      bool
	nextFn      bool
	fnAry       []string
	fnStr       string
	fnNo        int
	cats        []string
	goOutput    bool
	htmlPartCmd HtmlPartCmd
	hnfCmd      HnfCmd
}

type DiaryDate struct {
	year  string
	month string
	day   string
}

type TextProcIf interface {
	OpenSec(id string)
	CloseSec()
	Write([]byte) (int, error)
}

func FileNameToDate(fn string) DiaryDate {
	var res DiaryDate
	re := regexp.MustCompile(`d(\d{4})(\d{2})(\d{2})\D`)
	subRes := re.FindStringSubmatch(fn)
	if subRes != nil {
		res.year, res.month, res.day = subRes[1], subRes[2], subRes[3]
	} else {
		res.year, res.month, res.day = "2000", "00", "00"
	}
	return res
}

func incStrNum(str interface{}) string {
	num, err := strconv.Atoi(str.(string))
	if err == nil {
		return strconv.Itoa(num + 1)
	} else {
		return "0"
	}
}

func GWrite(data string) {
	g.Write([]byte(data))
}

func TEWrite(data string, context ...interface{}) error {
	str, err := mu.Render(data, context...)
	if err == nil {
		GWrite(str)
	} else {
		GWrite("<!-- mustache render error -->\n")
	}
	return err
}

func (self *RenderObj) HtmlHeader() {
}

func (self *RenderObj) HtmlAll() {
	v := reflect.ValueOf(&self.htmlPartCmd)
	self.htmlPartCmd.obj = self
	//  GWrite(fmt.Sprintf("<!--- seq:%d items -->\n", len(*self.htmlSeq)))
	for i, cmd := range *self.htmlSeq {
		var typeStr string = reflect.TypeOf(cmd).String()
		GWrite(fmt.Sprintf("<!--- %d:type %s -->\n", i, typeStr))
		switch cmd := cmd.(type) {
		case string:
			method := v.MethodByName(cmd)
			if method.IsValid() {
				//        GWrite(fmt.Sprintf("<!--- %d:method %s -->\n", i, cmd))
				method.Call([]reflect.Value{})
			} else {
				GWrite(fmt.Sprintf("<!--- %s: is not method -->\n", cmd))
			}
		case []interface{}:
			if len(cmd) > 0 {
				param := []reflect.Value{}
				cmdStr := cmd[0].(string)
				method := v.MethodByName(cmdStr)
				for _, p := range cmd[1:] {
					param = append(param, reflect.ValueOf(p))
				}
				if method.IsValid() {
					//          GWrite(fmt.Sprintf("<!--- %d:method %s -->\n", i, cmdStr))
					method.Call(param)
				} else {
					GWrite(fmt.Sprintf("<!--- %s: is not method -->\n", cmdStr))
				}
			}
		}
	}
}

func (self *HtmlPartCmd) Head1() {
	TEWrite((*self.obj.tmpl)[`DocType`])
	TEWrite((*self.obj.tmpl)[`Head1`], self.obj.doc)
}

func (self *HtmlPartCmd) Head2() {
	TEWrite((*self.obj.tmpl)[`Head2`], self.obj.doc)
}

func (self *HtmlPartCmd) OpenBody() {
	//  GWrite("<!-- OpenBody: " + (*self.tmpl)[`OpenBody`] + "-->\n")
	TEWrite((*self.obj.tmpl)[`OpenBody`], self.obj.doc)
}

func (self *HtmlPartCmd) CloseBody() {
	//  GWrite("<!-- CloseBody -->\n")
	t1 := time.Now()
	self.obj.doc["procTime"] = fmt.Sprintf("%v", t1.Sub(g.StartTime))
	TEWrite((*self.obj.tmpl)[`CloseBody`], self.obj.doc)
}

func (self *HtmlPartCmd) Diaries() {
	//  GWrite(fmt.Sprintf("<!-- Diaries(): -->\n"))
	GWrite(self.obj.OpenDiaries())
	for _, fn := range self.obj.dfiles {
		file, err := os.Open(fn)
		if err == nil {
			//      GWrite(fmt.Sprintf("<!-- Diaries: open %s -->\n", fn))
			self.obj.state = hsInit
			//      GWrite("<!-- state = hsInit -->\n")
			input := bufio.NewScanner(file)
			ddate := FileNameToDate(fn)
			self.obj.NewFile(ddate)
			for input.Scan() {
				dCodeLine := input.Text()
				line, _ := g.Conv.Conv([]byte(dCodeLine))

				lineStr := string(line[:])
				re := regexp.MustCompile(`(.*)\s+$`)
				subRes := re.FindStringSubmatch(lineStr)
				if subRes != nil {
					lineStr = subRes[1]
				}
				self.obj.doc[`line`] = lineStr
				//        TEWrite(`<!-- {{line}} -->` + "\n", self.doc)
				GWrite(self.obj.Line(lineStr))
				if self.obj.state == hsErr {
					break
				}
				if self.obj.nextFn {
					self.obj.fnOpen = true
					self.obj.nextFn = false
				}
			}
		}
		GWrite(self.obj.CloseDate())
		file.Close()
		GWrite(self.obj.Ads())
	}
	GWrite(self.obj.CloseDiaries())
}

func (self *HtmlPartCmd) Header() {
	TEWrite((*self.obj.tmpl)[`Title`], self.obj.doc)
}

func (self *HtmlPartCmd) Footer() {
	TEWrite((*self.obj.tmpl)[`Footer`], self.obj.doc)
}

func (self *HtmlPartCmd) StaticFile(params ...string) {
	GWrite("<!-- StaticFile " + params[0] + "-->\n")
	file, err := os.Open(params[0])
	if err == nil {
		input := bufio.NewScanner(file)
		for input.Scan() {
			line := input.Text()
			g.Write([]byte(line + "\n"))
		}
	} else {
		GWrite("<!-- ConvFile open error: " + params[0] + "-->\n")
	}
}

func (self *HtmlPartCmd) ConvFile(params ...string) {
	GWrite("<!-- ConvFile " + params[0] + "-->\n")
	file, err := os.Open(params[0])
	if err == nil {
		input := bufio.NewScanner(file)
		for input.Scan() {
			dCodeLine := input.Text()
			line, _ := g.Conv.Conv([]byte(dCodeLine + "\n"))
			g.Write(line)
		}
	} else {
		GWrite("<!-- ConvFile open error: " + params[0] + "-->\n")
	}
}

func NewRender() *RenderObj {
	var obj RenderObj
	return &obj
}

func (self *RenderObj) Render(dfiles []string, seq *config.HtmlSeqType, tmpl *config.TmplType, cats []string, ifc TextProcIf) {
	self.doc = map[string]interface{}{}
	self.doc[`Title`] = g.Config.Title
	self.filterCat = cats
	self.htmlSeq = seq
	self.tmpl = tmpl
	self.dfiles = dfiles
	//  self.HtmlHeader()
	self.HtmlAll()
}

// return string method

func (self *RenderObj) NewFile(ddate DiaryDate) string {
	var res string
	self.doc[`year`] = ddate.year
	self.doc[`month`] = ddate.month
	self.doc[`day`] = ddate.day
	self.dateOpen = false
	self.secOpen = false
	self.subOpen = false
	self.fnOpen = false
	self.nextFn = false
	self.fnAry = []string{}
	self.fnStr = ""
	self.cats = []string{}
	self.goOutput = len(self.filterCat) == 0
	return res
}

func (self *RenderObj) Line(line string) string {
	var res string = ""
	if len(line) == 0 {
		return res
	}
	re := regexp.MustCompile(`^\s*$`)
	if re.MatchString(line) {
		return res
	}
	switch self.state {
	case hsInit:
		if regexp.MustCompile(`^OK`).MatchString(line) {
			self.state = hsOk
			//      GWrite("<!-- self.state = hsOk -->\n")
			self.doc[`secno`] = "0"
			self.doc[`subsecno`] = "0"
			self.doc[`fnno`] = "0"
		}
	case hsOk:
		subs := regexp.MustCompile(`^(\/?[A-Z]+)(?:\s+(\S.*))?$`).
			FindStringSubmatch(line)
		if subs != nil { // comannd
			cmdStr := subs[1]
			if self.goOutput || cmdStr == "CAT" || cmdStr == "NEW" || cmdStr == "LNEW" {
				res = self.Cmd(cmdStr, subs[2])
			}
		} else {
			if !self.goOutput {
				return ""
			}
			//      if regexp.MustCompile(`^\!\#`).MatchString(line) { return "" }
			subs := regexp.MustCompile(`^\!(.*)$`).FindStringSubmatch(line)
			if subs != nil { // comment
				res = self.Comment(subs[1])
			} else {
				subs := regexp.MustCompile(`(.*)~$`).FindStringSubmatch(line)
				if subs != nil { // prinLn
					self.doc[`text`] = subs[1]
					res, _ = mu.Render((*self.tmpl)[`TextLn`], self.doc)
				} else {
					self.doc[`text`] = line
					res, _ = mu.Render((*self.tmpl)[`Text`], self.doc)
				}
			}
		}
		if self.fnOpen {
			self.fnStr = self.fnStr + res
			return ""
		} else {
			return res
		}
	}
	return res
}

func (self *RenderObj) Cmd(code string, param string) string {
	var res string
	//  GWrite(fmt.Sprintf("<!-- CMD:%s param:%s -->\n", code, param))
	if code[0] == '/' {
		code = "C_" + code[1:]
	}
	v := reflect.ValueOf(&self.hnfCmd)
	self.hnfCmd.obj = self
	method := v.MethodByName(code)
	if method.IsValid() {
		res = method.Call([]reflect.Value{reflect.ValueOf(param)})[0].Interface().(string)
	} else {
		return "<font color=red><br>ERROR " + code + " is not implemented</font><br>\n"
	}
	return res
}

func (self *RenderObj) OpenDiaries() string {
	//  GWrite(fmt.Sprintf("<!-- OpenDiaries(): -->\n"))
	self.doc[`dateCnt`] = "0"
	res, _ := mu.Render((*self.tmpl)[`OpenDiaries`], self.doc)
	return res
}

func (self *RenderObj) CloseDiaries() string {
	//  GWrite(fmt.Sprintf("<!-- CloseDiaries(): -->\n"))
	res, _ := mu.Render((*self.tmpl)[`CloseDiaries`], self.doc)
	return res
}

func (self *RenderObj) OpenDate() string {
	//  GWrite(fmt.Sprintf("<!-- OpenDate(): -->\n"))
	var res string
	if self.dateOpen {
		res = self.CloseDate()
	}
	str, _ := mu.Render((*self.tmpl)[`OpenDate`], self.doc)
	res = res + str
	self.dateOpen = true
	self.doc[`dateCnt`] = incStrNum(self.doc[`dateCnt`])
	return res
}

func (self *RenderObj) CloseDate() string {
	//  GWrite(fmt.Sprintf("<!-- CloseDate(): -->\n"))
	if !self.dateOpen {
		return ""
	}
	res := self.CloseSec()
	str, _ := mu.Render((*self.tmpl)[`CloseDate`], self.doc)
	res = res + str
	self.dateOpen = false
	return res
}

func (self *RenderObj) Ads() string {
	var res string
	if (self.doc[`dateCnt`] == "1") && (self.doc[`adsDone`] == nil) {
		str, _ := mu.Render((*self.tmpl)[`Ads`], self.doc)
		res = res + str
		self.doc[`adsDone`] = true
	}
	return res
}

func (self *RenderObj) OpenSec() string {
	var res string
	if self.secOpen {
		res = self.CloseSec()
	}
	self.doc[`subsecno`] = "0"
	self.doc[`fnno`] = "0"
	self.fnAry = []string{}
	self.secOpen = true
	str, _ := mu.Render((*self.tmpl)[`OpenSec`], self.doc)
	return res + str
}

func (self *RenderObj) CloseSec() string {
	if !self.secOpen {
		return ""
	}
	res := self.CloseSub()
	if len(self.fnAry) != 0 {
		str, _ := mu.Render((*self.tmpl)[`FnProcOpen`], self.doc)
		res = res + str
		for i, fn := range self.fnAry {
			self.doc[`fnno`] = strconv.Itoa(i + 1)
			self.doc[`text`] = fn
			str, _ = mu.Render((*self.tmpl)[`FnProc`], self.doc)
			res = res + str
		}
		str, _ = mu.Render((*self.tmpl)[`FnProcClose`], self.doc)
		res = res + str
	}
	str, _ := mu.Render((*self.tmpl)[`CloseSec`], self.doc)
	res = res + str
	self.secOpen = false
	return res
}

func (self *RenderObj) OpenSub() string {
	var res string
	if !self.subOpen {
		res = self.CloseSub()
	}
	str, _ := mu.Render((*self.tmpl)[`OpenSub`], self.doc)
	res = res + str
	self.subOpen = true
	return res
}

func (self *RenderObj) CloseSub() string {
	if !self.subOpen {
		return ""
	}
	res := self.CloseFn()
	str, _ := mu.Render((*self.tmpl)[`CloseSub`], self.doc)
	res = res + str
	self.subOpen = false
	return res
}

func (self *RenderObj) OpenFn() string {
	var res string
	if !self.fnOpen {
		res = self.CloseFn()
	}
	self.nextFn = true
	self.fnNo++
	self.fnStr = ""
	return res
}

func (self *RenderObj) CloseFn() string {
	var res string
	if !self.fnOpen {
		return ""
	}
	self.fnOpen = false
	self.fnAry = append(self.fnAry, self.fnStr)
	return ""
	return res
}

func (self *RenderObj) ParamArray(param string) []string {
	return []string{}
}

func (self *HnfCmd) CAT(param string) string {
	self.obj.cats = strings.Split(param, " ")
	return ""
}

func (self *RenderObj) PrepareOpenSec() string {
	res := self.CloseSec()
	self.doc[`cats`] = self.cats
	self.doc[`catsOn`] = strconv.FormatBool(len(self.cats) != 0)
	self.doc[`secno`] = incStrNum(self.doc[`secno`])
	if len(self.filterCat) == 0 {
		self.goOutput = true
	} else {
		self.goOutput = false
	loop:
		for _, s1 := range self.cats {
			for _, s2 := range self.filterCat {
				if s2 == s1 {
					self.goOutput = true
					break loop
				}
			}
		}
	}
	self.cats = []string{}
	if !self.goOutput {
		return res
	}
	if !self.dateOpen {
		return res + self.OpenDate()
	}
	return res
}

func (self *HnfCmd) NEW(param string) string {
	res := self.obj.PrepareOpenSec()
	if !self.obj.goOutput {
		return res
	}
	self.obj.doc[`sec`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`New`], self.obj.doc)
	return res + str + self.obj.OpenSec()
}

func (self *HnfCmd) LNEW(param string) string {
	res := self.obj.PrepareOpenSec()
	if !self.obj.goOutput {
		return res
	}
	subs := regexp.MustCompile(`^(\S+)\s+(\S.*)$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`uri`] = subs[1]
		self.obj.doc[`sec`] = subs[2]
		str, _ := mu.Render((*self.obj.tmpl)[`LNew`], self.obj.doc)
		res = res + str
	} else {
		res = res + "<font color=red><br>ERROR \"LNEW\" must have 2 param. </font><br>\n"
	}
	return res + self.obj.OpenSec()
}

func (self *HnfCmd) SUB(param string) string {
	res := self.obj.CloseSub()
	self.obj.doc[`subsec`] = param
	self.obj.doc[`subsecno`] = incStrNum(self.obj.doc[`subsecno`])
	str, _ := mu.Render((*self.obj.tmpl)[`Sub`], self.obj.doc)

	return res + str + self.obj.OpenSub()
}

func (self *HnfCmd) LSUB(param string) string {
	res := self.obj.CloseSub()
	subs := regexp.MustCompile(`^(\S+)\s+(\S.*)$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`uri`] = subs[1]
		self.obj.doc[`subsec`] = subs[2]
		self.obj.doc[`subsecno`] = incStrNum(self.obj.doc[`subsecno`])
		str, _ := mu.Render((*self.obj.tmpl)[`LSub`], self.obj.doc)
		res = res + str
	} else {
		res = res + "<font color=red><br>ERROR \"LSUB\" must have 2 param. </font><br>\n"
	}
	return res + self.obj.OpenSub()
}

func (self *HnfCmd) C_SUB(param string) string {
	return self.obj.CloseSub()
}

func (self *HnfCmd) P(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`P`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_P(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_P`], self.obj.doc)
	return str
}

func (self *HnfCmd) LINK(param string) string {
	subs := regexp.MustCompile(`^(\S+)\s+(\S.*)$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`uri`] = subs[1]
		self.obj.doc[`text`] = subs[2]
		str, _ := mu.Render((*self.obj.tmpl)[`Link`], self.obj.doc)
		return str
	} else {
		return "<font color=red><br>ERROR \"LINK\" must have 2 param.</font><br>\n"
	}
}

func (self *HnfCmd) STRIKE(param string) string {
	self.obj.doc[`text`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`Strike`], self.obj.doc)
	return str
}

func (self *HnfCmd) LSTRIKE(param string) string {
	subs := regexp.MustCompile(`^(\S+)\s+(\S.*)$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`uri`] = subs[1]
		self.obj.doc[`text`] = subs[2]
		str, _ := mu.Render((*self.obj.tmpl)[`LStrike`], self.obj.doc)
		return str
	} else {
		return "<font color=red><br>ERROR \"LSTRIKE\" must have 2 param.</font><br>\n"
	}
}

func (self *HnfCmd) CITE(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`Cite`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_CITE(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_Cite`], self.obj.doc)
	return str
}

func (self *HnfCmd) PRE(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`Pre`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_PRE(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_Pre`], self.obj.doc)
	return str
}

func (self *HnfCmd) UL(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`Ul`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_UL(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_Ul`], self.obj.doc)
	return str
}

func (self *HnfCmd) LI(param string) string {
	self.obj.doc[`text`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`Li`], self.obj.doc)
	return str
}

func (self *HnfCmd) DL(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`Dl`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_DL(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_Dl`], self.obj.doc)
	return str
}

func (self *HnfCmd) OL(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`Ol`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_OL(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_Ol`], self.obj.doc)
	return str
}

func (self *HnfCmd) DT(param string) string {
	self.obj.doc[`text`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`Dt`], self.obj.doc)
	return str
}

func (self *HnfCmd) DD(param string) string {
	self.obj.doc[`text`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`Dd`], self.obj.doc)
	return str
}

func (self *HnfCmd) DIV(param string) string {
	self.obj.doc[`text`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`Div`], self.obj.doc)
	return str
}

func (self *HnfCmd) C_DIV(param string) string {
	str, _ := mu.Render((*self.obj.tmpl)[`_Div`], self.obj.doc)
	return str
}

func (self *HnfCmd) STRONG(param string) string {
	self.obj.doc[`text`] = param
	str, _ := mu.Render((*self.obj.tmpl)[`Strong`], self.obj.doc)
	return str
}

func (self *HnfCmd) SPAN(param string) string {
	subs := regexp.MustCompile(`^(\S+)\s+(\S.*)$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`class`] = subs[1]
		self.obj.doc[`text`] = subs[2]
		str, _ := mu.Render((*self.obj.tmpl)[`Span`], self.obj.doc)
		return str
	} else {
		return ""
	}
}

func (self *RenderObj) Comment(str string) string {
	var res string
	return res
}

func (self *HnfCmd) FN(param string) string {
	self.obj.doc[`fnno`] = incStrNum(self.obj.doc[`fnno`])
	str, _ := mu.Render((*self.obj.tmpl)[`Fn`], self.obj.doc)
	return str + self.obj.OpenFn()
}

func (self *HnfCmd) C_FN(param string) string {
	return self.obj.CloseFn()
}

func (self *HnfCmd) IMG(param string) string {
	subs := regexp.MustCompile(`^(\S+)(?:\s+(\S.*))?$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`imgURI`] = subs[1]
		self.obj.doc[`text`] = subs[2]
	} else {
		self.obj.doc[`imgURI`] = param
		self.obj.doc[`text`] = ""
	}
	str, _ := mu.Render((*self.obj.tmpl)[`Img`], self.obj.doc)
	return str
}

func (self *HnfCmd) LIMG(param string) string {
	subs := regexp.MustCompile(`^(\S+)(?:\s+(\S+))(?:\s+(\S.*))?$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`imgURI`] = subs[1]
		self.obj.doc[`text`] = subs[2]
	} else {
		self.obj.doc[`imgURI`] = param
		self.obj.doc[`text`] = ""
	}
	str, _ := mu.Render((*self.obj.tmpl)[`Img`], self.obj.doc)
	return str
}

func (self *HnfCmd) PHOTO(param string) string {
	subs := regexp.MustCompile(`^(\S+)\s+(\S.*)$`).FindStringSubmatch(param)
	if subs != nil {
		self.obj.doc[`photoid`] = subs[1]
		self.obj.doc[`text`] = subs[2]
	} else {
		self.obj.doc[`photoid`] = param
		self.obj.doc[`text`] = ""
	}
	str, _ := mu.Render((*self.obj.tmpl)[`Photo`], self.obj.doc)
	return str
}
