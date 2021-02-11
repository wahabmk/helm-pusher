package pusher

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"sync"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
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

type Pusher struct {
	nCharts        int64
	nVersions      int64
	nRoutines      int64
	url            string
	username       string
	password       string
	repeatFailures bool
	verbose        bool
	helmExec       string
}

func New(nCharts, nVersions, nRoutines int64, url, username, password string, repeatFailures, verbose bool) (*Pusher, error) {
	if nRoutines <= 0 {
		return nil, fmt.Errorf("nRoutines cannot be <= 0")
	}
	if nRoutines > nCharts {
		return nil, fmt.Errorf("nRoutines cannot be > nCharts")
	}
	if nCharts <= 0 {
		return nil, fmt.Errorf("nCharts cannot be <= 0")
	}
	if nCharts < nVersions {
		return nil, fmt.Errorf("nCharts cannot be less than nVersions")
	}

	// Check if helm is installed
	helmExec, err := exec.LookPath("helm")
	if err != nil {
		return nil, err
	}

	return &Pusher{
		nCharts:        nCharts,
		nVersions:      nVersions,
		nRoutines:      nRoutines,
		url:            url,
		username:       username,
		password:       password,
		repeatFailures: repeatFailures,
		verbose:        verbose,
		helmExec:       helmExec,
	}, nil
}

func (p *Pusher) Push() error {
	// Create template chart
	if err := p.helm("create", templateChart); err != nil {
		return fmt.Errorf("unable to create temporary Helm chart: %s", err)
	}

	defer func() {
		if err := os.RemoveAll(templateChart); err != nil {
			println(fmt.Sprintf("error removing template chart: %s", err))
		}
	}()

	chartTempl, err := loader.LoadDir(templateChart)
	if err != nil {
		return fmt.Errorf("failed to load template chart %q: %w", templateChart, err)
	}

	// Create objects for the number fo go-routines required.
	each := math.Ceil(float64(p.nCharts) / float64(p.nRoutines))
	routines := make([]*Routine, p.nRoutines)
	for i := int64(0); i < p.nRoutines; i++ {
		routines[i] = &Routine{
			N:              int64(each),
			Chart:          func(c chart.Chart) *chart.Chart { return &c }(*chartTempl),
			RepeatFailures: p.repeatFailures,
		}
	}

	if diff := p.nRoutines*int64(each) - p.nCharts; diff > 0 {
		routines[len(routines)-1].N -= diff
	}

	var startTime time.Time
	// Go-routine to log progress every few seconds.
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if p.verbose {
				var done int64 = 0
				for i := int64(0); i < p.nRoutines; i++ {
					// done might not be accurate because `routines[i].N` is not
					// synchronized, but this is just for logging so it is okay.
					done += routines[i].N
				}
				fmt.Printf("%d charts pushed\tin %v\n", p.nCharts-done, time.Now().Sub(startTime).Round(1*time.Millisecond))
			} else {
				fmt.Printf("... ")
			}
		}
	}()

	fmt.Printf("Pushing %d charts:\n", p.nCharts)
	fmt.Printf("* To '%s/charts/api/%s/%s/charts'\n", host, namespace, repository)
	fmt.Printf("* With each chart having random number of versions between 1 to %d\n", p.nVersions)
	fmt.Printf("* With go-routines = %d\n", p.nRoutines)
	fmt.Printf("* With repeat failues = %v\n", p.repeatFailures)
	fmt.Printf("* With verbose logging = %v\n", p.verbose)
	fmt.Printf("\n\n")

	startTime = time.Now()
	var wg sync.WaitGroup
	for i := int64(0); i < p.nRoutines; i++ {
		wg.Add(1)
		go func(idx int64) {
			routines[idx].Do(p.nVersions)
			wg.Done()
		}(i)
	}
	wg.Wait()
	endTime := time.Now()

	// Output results.
	var errors int64 = 0
	for i := int64(0); i < p.nRoutines; i++ {
		errors += routines[i].Errors
	}
	fmt.Printf("\n\nResults:\n")
	fmt.Printf("* Errors encountered: %d\n", errors)
	fmt.Printf("* Charts successfully pushed: %d\n", p.nCharts-errors)
	fmt.Printf("* Time elapsed: %v\n", endTime.Sub(startTime).Round(1*time.Millisecond))

	return nil
}

func (p *Pusher) helm(arg ...string) error {
	cmd := exec.Command(p.helmExec, arg...)

	_, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok {
		// If error is the result of executing the helm command,
		// then using `exec.ExitError` allows us to get the actual error msg from helm CLI.
		return fmt.Errorf("helm error: %s", ee.Stderr)
	}

	return err
}
