package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wahabmk/helm-pusher/pusher"
)

var (
	nVersions = flag.Int("versions", 1000, "number of chart versions")
	nCharts   = flag.Int("charts", 100, "number of uniquely named charts to insert")
	nRoutines = flag.Int("concurrency", 20, "number of concurrent operations")
)

func init() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] URL\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample:\n  %s https://admin:password1234@localhost:8443/charts/api/admin/repo/charts\n", os.Args[0])
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		flag.CommandLine.Usage()
		os.Exit(1)
	}

	p, err := pusher.New(*nCharts, *nVersions, *nRoutines, flag.Arg(0))
	if err != nil {
		fmt.Fprintln(flag.CommandLine.Output(), err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sig
		cancel()
	}()

	if err := p.Do(ctx); err != nil {
		fmt.Fprintln(flag.CommandLine.Output(), err)
		os.Exit(1)
	}
}
