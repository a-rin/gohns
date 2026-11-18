package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const (
	kFN_configpath    = "configpath.json"
	kFN_defaultconfig = "config.json"
)

type StyleType struct {
	CalenderUse bool
}

type PIMType struct {
	Style    map[string]string
	Schedule map[string]string
}

// type HtmlBodySeqType []string
type HtmlSeqType []interface{}

type TmplType map[string]string

type ConfigType struct {
	Title              string
	UserMail           string
	Auther             string
	DiaryDir           string
	DiaryCode          string
	MyDiaryURI         string
	Theme              string
	DefaultMax         int
	DefaultTitleMax    int
	CacheDir           string
	LogDir             string
	StartYear          int
	RecentReverse      bool
	YMDReverse         bool
	RecentTitleReverse bool
	YMDTitleReverse    bool
	FNCountStyle       bool
	Style              StyleType
	PIM                PIMType
	HtmlSeq            HtmlSeqType
	Tmpl               TmplType
	TitleHtmlSeq       HtmlSeqType
	TitleTmpl          TmplType
}

type ConfigPathType struct {
	ConfigPath string
}

// ReadConfig reads the configuration file: the one named in configpath.json,
// or config.json when configpath.json is missing or cannot be read.
func ReadConfig() (*ConfigType, error) {
	var configPath string = kFN_defaultconfig
	var config ConfigType

	file1, err := os.Open(kFN_configpath)
	if err == nil {
		defer file1.Close()
		dec := json.NewDecoder(bufio.NewReader(file1))
		var path ConfigPathType
		for dec.More() {
			err = dec.Decode(&path)
			if err != nil {
				log.Printf("%s: %v; using %s", kFN_configpath, err, kFN_defaultconfig)
				break
			}
		}
		if (err == nil) && (path.ConfigPath != "") {
			configPath = path.ConfigPath
		}
	}
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	dec := json.NewDecoder(bufio.NewReader(file))
	for dec.More() {
		if err := dec.Decode(&config); err != nil {
			return nil, fmt.Errorf("%s: %w", configPath, err)
		}
	}
	return &config, nil
}
