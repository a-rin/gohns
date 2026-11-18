// CUI for debug
package main

import (
	"./hns"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
)

func write([]byte) (int, error) {
	return 0, nil
}

func main() {
	ts := httptest.NewServer(http.HandlerFunc(hns.Handler))
	defer ts.Close()
	res, err := http.Get(ts.URL)
	//  res, err := http.Get(ts.URL + `/?20111008`)
	//	res, err := http.Get(ts.URL + `/?title`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Header:\n%v\n", res.Header)
	greeting, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", greeting)
}
