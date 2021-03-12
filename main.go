package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	// TODO: Parameterize these.
	nCharts   = int64(1)
	nVersions = int64(1)
	nRoutines = int64(1)
	// url            = "https://13.126.107.180:443/charts/api/admin/myrepo/charts"
	url            = "http://127.0.0.1:8082/artifactory/myrepo"
	repeatFailures = false
	verbose        = true
	username       = "admin"
	password       = "SwiftNinja420"
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
