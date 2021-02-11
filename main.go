package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	// TODO: Parameterize these.
	nCharts        = int64(10)
	nVersions      = int64(2)
	nRoutines      = int64(2)
	url            = "https://52.66.227.158/charts/api/admin/myrepo/charts"
	repeatFailures = true
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
