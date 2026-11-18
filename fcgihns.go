package main

import (
	"github.com/a-rin/gohns/hns"
	_ "github.com/a-rin/gohns/hns/ghns"
	"net/http"
	"net/http/fcgi"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	n := runtime.NumGoroutine() + 1 // initial number of Goroutines

	// install signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	// Spawn request handler
	go func() {
		err := fcgi.Serve(nil, http.HandlerFunc(hns.Handler))
		if err != nil {
			panic(err)
		}
	}()

	// catch signal
	_ = <-c

	// give pending requests in fcgi.Serve() some time to enter the request handler
	time.Sleep(time.Millisecond * 100)

	// wait at most 3 seconds for request handlers to finish
	for i := 0; i < 30; i++ {
		if runtime.NumGoroutine() <= n {
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
}

