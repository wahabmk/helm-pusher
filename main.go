package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	// TODO: Parameterize these.
	nCharts        = int64(100000)
	nVersions      = int64(50)
	nRoutines      = int64(100)
	url            = "https://52.66.227.158/charts/api/admin/myrepo/charts"
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
