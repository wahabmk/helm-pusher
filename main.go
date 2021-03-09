package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	// TODO: Parameterize these.
	nCharts   = int64(50000)
	nVersions = int64(100)
	nRoutines = int64(20)
	// this
	url            = "http://127.0.0.1:8080/api/charts"
	repeatFailures = false
	verbose        = true
	username       = "admin"
	password       = "password1234"
	templateChart  = "/tmp/testchart"
)

func main() {
	p, err := pusher.New(nCharts, nVersions, nRoutines, url, username, password, repeatFailures, verbose)
	if err != nil {
		println(err)
	}

	if err := p.Push(); err != nil {
		fmt.Println(err)
	}
}
