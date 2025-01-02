package timecheck

import (
	"custom_lib/common"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	httpstat "github.com/digitaljanitors/go-httpstat"
)

func GetLatency(destination string) int {
	//req, err := http.NewRequest("GET", "http://"+destination+":8080", nil)
	req, err := http.NewRequest("GET", "http://"+destination+":"+common.Port+"/ping", nil)
	if err != nil {
		log.Fatal(err)
	}

	//create go-httpstat powered context and pass it to http.Request
	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return 999999
	}

	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Fatal(err)
	}
	res.Body.Close()
	result.End(time.Now())
	latency := int(result.Total / time.Millisecond)
	return latency
}

func isConnectionRefusedError(err error) bool {
	// Check if the error indicates a connection refused error
	netErr, ok := err.(*net.OpError)
	if ok && netErr != nil {
		return true
	}
	return false
}
