package main

import (
	"./hns"
	_ "./hns/ghns"
	"net/http"
	"net/http/cgi"
)

func main() {
	err := cgi.Serve(http.HandlerFunc(hns.Handler))
	if err != nil {
		panic(err)
	}
}
