package ghns

import (
	"../config"
	iconv "github.com/s-yata/go-iconv"
	"time"
)

var (
	Conv      *iconv.Handle
	Config    *config.ConfigType
	StartTime time.Time
	Write     func([]byte) (int, error)
)

func InitG() bool {
	var ierr error
	if Config == nil {
		Config = config.ReadConfig()
		if Config != nil {
			Conv, ierr = iconv.Open("UTF8", Config.DiaryCode)
			if ierr != nil {
				panic(ierr)
			}
		}
	}
	return Config != nil
}

func TermG() {
	_ = Conv.Close()
	Config = nil
}

func init() {
}
