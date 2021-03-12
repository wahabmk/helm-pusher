package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	// TODO: Parameterize these.
	nCharts   = int64(10000)
	nVersions = int64(100)
	nRoutines = int64(10)
	// url            = "https://13.126.107.180:443/charts/api/admin/myrepo/charts"
	url            = "http://65.1.113.103:8082/artifactory/myrepo"
	repeatFailures = false
	verbose        = true
	username       = ""
	password       = ""
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
