package ghns

import (
	"fmt"
	"github.com/a-rin/gohns/hns/config"
	iconv "github.com/s-yata/go-iconv"
	"time"
)

var (
	Conv      *iconv.Handle
	Config    *config.ConfigType
	StartTime time.Time
	Write     func([]byte) (int, error)
)

// InitG reads the configuration and opens the converter. Call TermG when it
// returns nil.
func InitG() error {
	if Config != nil {
		return nil
	}
	config, err := config.ReadConfig()
	if err != nil {
		return err
	}
	conv, err := iconv.Open("UTF8", config.DiaryCode)
	if err != nil {
		return fmt.Errorf("DiaryCode %q: %w", config.DiaryCode, err)
	}
	Config, Conv = config, conv
	return nil
}

func TermG() {
	_ = Conv.Close()
	Config = nil
}

func init() {
}
