package pusher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	neturl "net/url"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/wahabmk/helm-pusher/pkg/generator"
	"github.com/wahabmk/helm-pusher/pkg/helm"
	"golang.org/x/sync/errgroup"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const maxAttempts = 5

type Pusher struct {
	// The following fields need to be placed at the beginning of the struct to
	// ensure 64-bit alignment for atomic operations.
	nVersions int64
	conflicts int64

	nCharts   int
	nRoutines int
	url       *url.URL
}

func New(nCharts, nVersions, nRoutines int, url string) (*Pusher, error) {
	if nRoutines <= 0 {
		return nil, fmt.Errorf("nRoutines cannot be <= 0")
	}
	if nCharts <= 0 {
		return nil, fmt.Errorf("nCharts cannot be <= 0")
	}
	if nVersions <= 0 {
		return nil, fmt.Errorf("nVersions cannot be <= 0")
	}
	if nVersions < nCharts {
		return nil, fmt.Errorf("nVersions cannot be less than nCharts")
	}

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}

	return &Pusher{
		nCharts:   nCharts,
		nVersions: int64(nVersions),
		nRoutines: nRoutines,
		url:       parsedURL,
	}, nil
}

func createTemplateChart(ctx context.Context) (tpl *chart.Chart, err error) {
	templateChart, err := ioutil.TempDir("", "helm-pusher-")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary directory: %w", err)
	}
	defer func() {
		if rmErr := os.RemoveAll(templateChart); rmErr != nil {
			rmErr = fmt.Errorf("error removing template chart: %w", rmErr)
			if err != nil {
				err = fmt.Errorf("%w; %s", err, rmErr)
			} else {
				err = rmErr
			}
		}
	}()

	if err := helmCommand(ctx, "create", templateChart); err != nil {
		return nil, fmt.Errorf("unable to create temporary Helm chart: %w", err)
	}
	return loader.LoadDir(templateChart)
}

func generateChart(template *chart.Chart, name, version string) (io.Reader, error) {
	// Make a copy, deep-copying the bits that will be modified.
	md := *template.Metadata
	md.Name = name
	md.Version = version

	c := *template
	c.Metadata = &md
	return helm.PackageChart(&c)
}

func (p *Pusher) Do(ctx context.Context) error {
	chartTempl, err := createTemplateChart(ctx)
	if err != nil {
		return err
	}

	rng := generator.New(p.nCharts)
	wg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < p.nRoutines; i++ {
		wg.Go(func() error { return p.do(ctx, chartTempl, &rng) })
	}

	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				v := p.nVersions
				if v < 0 {
					v = 0
				}
				fmt.Printf("%d chart versions remaining (%d conflicts)\n", v, p.conflicts)
			case <-ctx.Done():
				fmt.Printf("Total conflicts: %d\n", p.conflicts)
				return
			}
		}
	}()

	return wg.Wait()
}

func (p *Pusher) do(ctx context.Context, tpl *chart.Chart, rng *generator.Generator) error {
	u := p.url.String()

	for atomic.AddInt64(&p.nVersions, -1) >= 0 {
		success := false
		var attempt int
		for !success {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			attempt++
			if attempt > maxAttempts {
				return fmt.Errorf("too many failed attempts; giving up")
			}

			c, err := generateChart(tpl, rng.ChartName(), rng.Semver())
			if err != nil {
				return fmt.Errorf("error generating chart from template: %w", err)
			}

			if err := pushChart(ctx, c, u, false); err != nil {
				if errors.Is(err, ErrConflict) {
					atomic.AddInt64(&p.conflicts, 1)
					continue
				}
				return err
			}
			success = true
		}
	}
	return nil
}

func helmCommand(ctx context.Context, arg ...string) error {
	cmd := exec.CommandContext(ctx, "helm", arg...)

	_, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok {
		// If error is the result of executing the helm command,
		// then using `exec.ExitError` allows us to get the actual error msg from helm CLI.
		return fmt.Errorf("helm error: %s", ee.Stderr)
	}

	return err
}
