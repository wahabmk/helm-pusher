package main

import (
	"fmt"

	"github.com/wahabmk/helm-pusher/pusher"
)

const (
	defaultRoutines = 10
	host            = "52.66.227.158"
	username        = "admin"
	password        = "password1234"
	namespace       = "admin"
	repository      = "myrepo"
	templateChart   = "/tmp/testchart"
)

func main() {
	nCharts := int64(500)
	nVersions := int64(10)
	nRoutines := int64(10)
	username := "admin"
	password := "password1234"
	url := ""
	repeatFailures := true
	verbose := true
	p, err := pusher.New(nCharts, nVersions, nRoutines, url, username, password, repeatFailures, verbose)
	if err != nil {
		println(err)
	}

	if err := p.Push(); err != nil {
		fmt.Println(err)
	}
}
