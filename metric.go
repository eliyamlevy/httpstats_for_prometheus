package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	httpstat "github.com/tcnksm/go-httpstat"
)

const url string = "http://www.google.com"

func getStats(w http.ResponseWriter, req *http.Request) {
	//Creates request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Create go-httpstat powered context and pass it to http.Request
	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
	result.End(time.Now())

	//Prints result
	strResult := fmt.Sprintf("%+v\n", result)
	io.WriteString(w, strResult)
}

func main() {
	http.HandleFunc("/metric", getStats)
	log.Fatal(http.ListenAndServe(":7777", nil))
}
