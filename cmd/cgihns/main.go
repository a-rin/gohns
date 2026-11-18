package main

import (
	"github.com/a-rin/gohns/hns"
	_ "github.com/a-rin/gohns/hns/ghns"
	"net/http"
	"net/http/cgi"
)

func main() {
	err := cgi.Serve(http.HandlerFunc(hns.Handler))
	if err != nil {
		panic(err)
	}
}
