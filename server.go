package main

import (
	"github.com/a-rin/gohns/hns"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", hns.Handler)
	log.Fatal(http.ListenAndServe(":12345", nil))
}
