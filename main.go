package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	// TODO: Parameterize these.
	nCharts        = int64(1000)
	nVersions      = int64(10)
	nRoutines      = int64(5)
	url            = "https://13.126.107.180:443/charts/api/admin/myrepo/charts"
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
