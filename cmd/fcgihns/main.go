package main

import (
	"github.com/a-rin/gohns/hns"
	_ "github.com/a-rin/gohns/hns/ghns"
	"net/http"
	"net/http/fcgi"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	var active atomic.Int64 // requests that are being handled

	// install signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	// Spawn request handler
	go func() {
		err := fcgi.Serve(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			active.Add(1)
			defer active.Add(-1)
			hns.Handler(w, r)
		}))
		if err != nil {
			panic(err)
		}
	}()

	// catch signal
	_ = <-c

	// give pending requests in fcgi.Serve() some time to enter the request handler
	time.Sleep(time.Millisecond * 100)

	// wait for the requests that are being handled, at most 3 seconds
	for i := 0; i < 30 && active.Load() > 0; i++ {
		time.Sleep(time.Millisecond * 100)
	}
}
